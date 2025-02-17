package tracker

import (
	"net"
	"strconv"
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

	Logger   *logrus.Logger
	Channels map[string]Channels

	DaemonRouters map[string]*prouter.MessageRouter
}

type Channels struct {
	HeartbeatCh       chan *protocol.NetworkMessage_Heartbeat
	GoodbyeCh         chan *protocol.NetworkMessage_Goodbye
	AnnounceCh        chan *protocol.NetworkMessage_Announce
	PeerListRequestCh chan *protocol.NetworkMessage_PeerListRequest
	SignalingCh       chan *protocol.NetworkMessage_Signaling
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

	return &Tracker{
		PeerStore:     peerStore,
		FileStore:     fileStore,
		ChunkStore:    chunkStore,
		Logger:        logger,
		Channels:      make(map[string]Channels),
		DaemonRouters: make(map[string]*prouter.MessageRouter),
	}
}

func (t *Tracker) Start() {
	listen, err := net.Listen("tcp", ":59000")
	if err != nil {
		t.Logger.Fatalf("Error starting TCP server: %+v", err)
		return
	}
	defer listen.Close()

	t.Logger.Infof("Server listening on port 59000")

	for {
		conn, err := listen.Accept()
		if err != nil {
			t.Logger.Warnf("Error accepting connection: %+v", err)
			continue
		}

		// Setup prouter
		heartbeatCh := make(chan *protocol.NetworkMessage_Heartbeat, 100)
		goodbyeCh := make(chan *protocol.NetworkMessage_Goodbye, 100)
		announceCh := make(chan *protocol.NetworkMessage_Announce, 100)
		peerListRequestCh := make(chan *protocol.NetworkMessage_PeerListRequest, 100)
		signalingCh := make(chan *protocol.NetworkMessage_Signaling, 100)
		channels := Channels{
			HeartbeatCh:       heartbeatCh,
			GoodbyeCh:         goodbyeCh,
			AnnounceCh:        announceCh,
			PeerListRequestCh: peerListRequestCh,
			SignalingCh:       signalingCh,
		}
		t.Channels[conn.RemoteAddr().String()] = channels
		prouter := prouter.NewMessageRouter(conn)
		prouter.AddRoute(heartbeatCh, func(msg proto.Message) bool {
			_, ok := msg.(*protocol.NetworkMessage).MessageType.(*protocol.NetworkMessage_Heartbeat)
			return ok
		})
		prouter.AddRoute(goodbyeCh, func(msg proto.Message) bool {
			_, ok := msg.(*protocol.NetworkMessage).MessageType.(*protocol.NetworkMessage_Goodbye)
			return ok
		})
		prouter.AddRoute(announceCh, func(msg proto.Message) bool {
			_, ok := msg.(*protocol.NetworkMessage).MessageType.(*protocol.NetworkMessage_Announce)
			return ok
		})
		prouter.AddRoute(peerListRequestCh, func(msg proto.Message) bool {
			_, ok := msg.(*protocol.NetworkMessage).MessageType.(*protocol.NetworkMessage_PeerListRequest)
			return ok
		})
		prouter.AddRoute(signalingCh, func(msg proto.Message) bool {
			_, ok := msg.(*protocol.NetworkMessage).MessageType.(*protocol.NetworkMessage_Signaling)
			return ok
		})

		// Listen for incoming msgs
		go prouter.Start()
		go t.handleDaemonMsgs(prouter)
	}
}

func (t *Tracker) handleDaemonMsgs(prouter *prouter.MessageRouter) {
	daemonAddr := prouter.Conn.RemoteAddr().String()
	daemonIP, daemonPort, _ := net.SplitHostPort(prouter.Conn.RemoteAddr().String())
	t.Logger.Infof("New peer connected: %s:%s\n", daemonIP, daemonPort)
	daemonId, err := t.PeerStore.CreatePeer(daemonIP, daemonPort)

	idNetMsg := &protocol.NetworkMessage{
		MessageType: &protocol.NetworkMessage_Id{
			Id: &protocol.IDMessage{
				Id: daemonId,
			},
		},
	}
	prouter.WriteMessage(idNetMsg)
	t.Logger.Infof("Sent %s back its ID: %s", daemonAddr, daemonId)
	if err != nil {
		t.Logger.Warnf("Error creating peer: %v", err)
		return
	}

	t.DaemonRouters[daemonId] = prouter

	// Start a timer to track client timeout
	timeout := time.NewTicker(ClientTimeout * time.Second)
	defer timeout.Stop()
	channels, exists := t.Channels[daemonAddr]
	if !exists {
		t.Logger.Warn("Daemon channels do not existw")
		return
	}

	// Send it back its ID

	for {
		select {
		// If the client times out, delete the client from the db
		case <-timeout.C:
			t.Logger.Infof("Peer %s timed out, deleting \n", daemonAddr)
			delete(t.DaemonRouters, daemonId) // remove if issue occur
			err := t.PeerStore.DeletePeer(daemonIP, daemonPort)
			if err != nil {
				t.Logger.Warnf("Error deleting peer: %v", err)
			}
			return

		case signalingMsg := <-channels.SignalingCh:
			targetPeerId := signalingMsg.Signaling.GetTargetPeerId()
			forwardMsg := &protocol.NetworkMessage{
				MessageType: &protocol.NetworkMessage_Signaling{
					Signaling: &protocol.SignalingMessage{
						SourcePeerId: daemonId,
						TargetPeerId: targetPeerId,
						Message:      signalingMsg.Signaling.GetMessage(),
					},
				},
			}

			if targetRouter, exists := t.DaemonRouters[targetPeerId]; exists {
				err = targetRouter.WriteMessage(forwardMsg)
				if err != nil {
					t.Logger.Warnf("Error forwarding signaling message: %v", err)
				}
			}

		case <-channels.HeartbeatCh:
			timeout.Reset(ClientTimeout * time.Second)

		case <-channels.GoodbyeCh:
			// Remove the client from db
			t.PeerStore.DeletePeer(daemonIP, daemonPort)
			t.Logger.Infof("Peer %s disconnected", daemonAddr)
			// prouter.Stop()
			return

		case announce := <-channels.AnnounceCh:
			t.Logger.Debugf("Received Announce message from %s: %v", daemonAddr, announce.Announce)
			files := announce.Announce.GetFiles()
			for _, file := range files {
				totalChunks := int(file.GetTotalChunks())
				maxChunkSize := int(file.GetChunkSize())
				schemaFile := &schema.File{
					Name:         file.GetFileName(),
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

		case peerListRequest := <-channels.PeerListRequestCh:
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

				peers = append(peers, &protocol.PeerInfo{
					Id: strconv.Itoa(int(peer.ID)),
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
