package tracker

import (
	"net"
	"time"

	"github.com/rudransh-shrivastava/peer-it/internal/shared/protocol"
	"github.com/rudransh-shrivastava/peer-it/internal/shared/prouter"
	"github.com/rudransh-shrivastava/peer-it/internal/shared/schema"
	"github.com/rudransh-shrivastava/peer-it/internal/shared/store"
	"github.com/rudransh-shrivastava/peer-it/internal/shared/utils/logger"
	"github.com/rudransh-shrivastava/peer-it/internal/tracker/db"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/proto"
)

const (
	ClientTimeout = 10
)

type Tracker struct {
	PeerStore  *store.PeerStore
	FileStore  *store.FileStore
	ChunkStore *store.ChunkStore

	Logger *logrus.Logger

	HeartbeatCh       chan *protocol.NetworkMessage_Heartbeat
	GoodbyeCh         chan *protocol.NetworkMessage_Goodbye
	RegisterCh        chan *protocol.NetworkMessage_Register
	AnnounceCh        chan *protocol.NetworkMessage_Announce
	PeerListRequestCh chan *protocol.NetworkMessage_PeerListRequest
}

func NewTracker() *Tracker {
	logger := logger.NewLogger()
	db, err := db.NewDB()
	if err != nil {
		logger.Fatal(err)
		return &Tracker{}
	}
	peerStore := store.NewPeerStore(db)
	fileStore := store.NewFileStore(db)
	chunkStore := store.NewChunkStore(db)

	heartbeatCh := make(chan *protocol.NetworkMessage_Heartbeat, 100)
	goodbyeCh := make(chan *protocol.NetworkMessage_Goodbye, 100)
	registerCh := make(chan *protocol.NetworkMessage_Register, 100)
	announceCh := make(chan *protocol.NetworkMessage_Announce, 100)
	peerListRequestCh := make(chan *protocol.NetworkMessage_PeerListRequest, 100)

	return &Tracker{
		PeerStore:         peerStore,
		FileStore:         fileStore,
		ChunkStore:        chunkStore,
		Logger:            logger,
		HeartbeatCh:       heartbeatCh,
		GoodbyeCh:         goodbyeCh,
		RegisterCh:        registerCh,
		AnnounceCh:        announceCh,
		PeerListRequestCh: peerListRequestCh,
	}
}

func (t *Tracker) Start() {
	listen, err := net.Listen("tcp", ":42069")
	if err != nil {
		t.Logger.Fatalf("Error starting TCP server: %+v", err)
		return
	}
	defer listen.Close()

	t.Logger.Infof("Server listening on port 42069")

	for {
		conn, err := listen.Accept()
		if err != nil {
			t.Logger.Warnf("Error accepting connection: %+v", err)
			continue
		}

		// Setup prouter
		prouter := prouter.NewMessageRouter(conn)
		prouter.AddRoute(t.HeartbeatCh, func(msg proto.Message) bool {
			_, ok := msg.(*protocol.NetworkMessage).MessageType.(*protocol.NetworkMessage_Heartbeat)
			return ok
		})
		prouter.AddRoute(t.GoodbyeCh, func(msg proto.Message) bool {
			_, ok := msg.(*protocol.NetworkMessage).MessageType.(*protocol.NetworkMessage_Goodbye)
			return ok
		})
		prouter.AddRoute(t.RegisterCh, func(msg proto.Message) bool {
			_, ok := msg.(*protocol.NetworkMessage).MessageType.(*protocol.NetworkMessage_Register)
			return ok
		})
		prouter.AddRoute(t.AnnounceCh, func(msg proto.Message) bool {
			_, ok := msg.(*protocol.NetworkMessage).MessageType.(*protocol.NetworkMessage_Announce)
			return ok
		})
		prouter.AddRoute(t.PeerListRequestCh, func(msg proto.Message) bool {
			_, ok := msg.(*protocol.NetworkMessage).MessageType.(*protocol.NetworkMessage_PeerListRequest)
			return ok
		})

		remoteAddr := conn.RemoteAddr().String()
		daemonIP, daemonPort, err := net.SplitHostPort(remoteAddr)
		if err != nil {
			t.Logger.Fatal(err)
			return
		}

		// Listen for incoming msgs
		go prouter.Start()
		go t.handleDaemonMsgs(daemonIP, daemonPort, prouter)
	}
}

func (t *Tracker) handleDaemonMsgs(daemonIP, daemonPort string, prouter *prouter.MessageRouter) {
	t.Logger.Infof("New peer connected: %s:%s\n", daemonIP, daemonPort)
	daemonAddr := daemonIP + ":" + daemonPort
	err := t.PeerStore.CreatePeer(daemonIP, daemonPort)
	if err != nil {
		t.Logger.Warnf("Error creating peer: %v", err)
		return
	}

	// Start a timer to track client timeout
	timeout := time.NewTicker(ClientTimeout * time.Second)
	defer timeout.Stop()

	for {
		select {
		// If the client times out, delete the client from the db
		case <-timeout.C:
			t.Logger.Infof("Peer %s timed out, deleting \n", daemonAddr)
			err := t.PeerStore.DeletePeer(daemonIP, daemonPort)
			if err != nil {
				t.Logger.Warnf("Error deleting peer: %v", err)
			}
			// prouter.Stop()
			return

		case heartbeat := <-t.HeartbeatCh:
			t.Logger.Debugf("Received heartbeat from %s: %v", daemonAddr, heartbeat.Heartbeat)
			timeout.Reset(ClientTimeout * time.Second)

		case <-t.GoodbyeCh:
			// Remove the client from db
			t.PeerStore.DeletePeer(daemonIP, daemonPort)
			t.Logger.Infof("Peer %s disconnected", daemonAddr)
			// prouter.Stop()
			return

		case register := <-t.RegisterCh:
			clientID := register.Register.GetClientId()
			// Save the public listener port of the client in DB
			t.Logger.Debugf("Register message received from connection %s", prouter.Conn.RemoteAddr())
			t.Logger.Debugf("Received register message from %s: with UUID %s%v", daemonAddr, clientID, register)
			err := t.PeerStore.RegisterPeer(clientID, daemonIP, daemonPort, register.Register.GetPublicIpAddress(), register.Register.GetListenPort())
			if err != nil {
				t.Logger.Fatalf("Error registering client: %v", err)
			}
			t.Logger.Debugf("Peer %s registered with public listener port %s with UUID %s", daemonAddr, clientID, register.Register.GetListenPort())

		case announce := <-t.AnnounceCh:
			t.Logger.Debugf("Received Announce message from %s: %v", daemonAddr, announce.Announce)
			files := announce.Announce.GetFiles()
			for _, file := range files {
				totalChunks := int(file.GetTotalChunks())
				maxChunkSize := int(file.GetChunkSize())
				schemaFile := &schema.File{
					Size:         file.GetFileSize(),
					MaxChunkSize: maxChunkSize,
					TotalChunks:  totalChunks,
					Hash:         file.GetFileHash(),
					CreatedAt:    time.Now().Unix(),
				}
				created, err := t.FileStore.CreateFile(schemaFile)
				if err != nil {
					t.Logger.Fatalf("Error creating file: %v", err)
				}
				if !created {
					// File already existed in db
					t.Logger.Debugf("File %+v already exists, adding peer to swarm", schemaFile)
				} else {
					t.Logger.Debugf("Registered file: %s, Size: %d, Chunks: %d Max Chunk Size: %d", file.GetFileHash(), file.GetFileSize(), file.GetTotalChunks(), file.GetChunkSize())
				}
				// Add client to swarm of peers
				err = t.PeerStore.AddPeerToSwarm(daemonIP, daemonPort, file.GetFileHash())
			}

		case peerListRequest := <-t.PeerListRequestCh:
			t.Logger.Debugf("Received peer list request from %s: %v", daemonAddr, peerListRequest.PeerListRequest)
			fileHash := peerListRequest.PeerListRequest.GetFileHash()

			dbPeers, err := t.PeerStore.GetPeersByFileHash(fileHash)
			if err != nil {
				t.Logger.Fatalf("Error getting peers: %v", err)
			}

			peers := make([]*protocol.PeerInfo, 0)
			for _, peer := range dbPeers {
				if peer.IPAddress == daemonIP && peer.Port == daemonPort {
					t.Logger.Debugf("Skipping %s", daemonAddr)
					t.Logger.Debugf("Skipped: %+v", peer)
					continue
				}

				publicIPAddr, publicListenPort, err := t.PeerStore.FindPublicListenPort(peer.IPAddress, peer.Port)
				if err != nil {
					t.Logger.Warnf("Error finding public listen port: %v", err)
					continue
				}
				t.Logger.Debugf("Got public IP:PORT for %+v from db %s:%s", peer, publicIPAddr, publicListenPort)
				peers = append(peers, &protocol.PeerInfo{
					IpAddress: publicIPAddr,
					Port:      publicListenPort,
				})
			}
			peerListResponse := &protocol.PeerListResponse{
				FileHash:    fileHash,
				TotalChunks: 0,
				ChunkSize:   0,
				Peers:       peers,
			}
			response := &protocol.NetworkMessage{
				MessageType: &protocol.NetworkMessage_PeerListResponse{
					PeerListResponse: peerListResponse,
				},
			}
			t.Logger.Debugf("Sending back the peers list to %s : %+v", daemonAddr, peerListResponse)
			prouter.WriteMessage(response)
			if err != nil {
				t.Logger.Warnf("Error sending peer list response: %v", err)
			}
		}
	}
}

func (t *Tracker) Stop() {
	// cleanup peers
	t.Logger.Infof("Stopping the tracker...")
	err := t.PeerStore.DropAllPeers()
	if err != nil {
		t.Logger.Warnf("Error deleting all peers %v", err)
	}
}
