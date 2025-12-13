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
	Channels      map[string]Channels
	ChunkStore    *store.ChunkStore
	DaemonRouters map[string]*prouter.MessageRouter
	FileStore     *store.FileStore
	Logger        *logrus.Logger
	PeerStore     *store.PeerStore
}

type Channels struct {
	AnnounceCh        chan *protocol.NetworkMessage_Announce
	GoodbyeCh         chan *protocol.NetworkMessage_Goodbye
	HeartbeatCh       chan *protocol.NetworkMessage_Heartbeat
	PeerListRequestCh chan *protocol.NetworkMessage_PeerListRequest
	SignalingCh       chan *protocol.NetworkMessage_Signaling
}

func NewTracker() *Tracker {
	log := logger.NewLogger()
	database, err := db.NewDB()
	if err != nil {
		log.Fatal(err)
		return &Tracker{}
	}
	peerStore := store.NewPeerStore(database)
	fileStore := store.NewFileStore(database)
	chunkStore := store.NewChunkStore(database)

	return &Tracker{
		Channels:      make(map[string]Channels),
		ChunkStore:    chunkStore,
		DaemonRouters: make(map[string]*prouter.MessageRouter),
		FileStore:     fileStore,
		Logger:        log,
		PeerStore:     peerStore,
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
			AnnounceCh:        announceCh,
			GoodbyeCh:         goodbyeCh,
			HeartbeatCh:       heartbeatCh,
			PeerListRequestCh: peerListRequestCh,
			SignalingCh:       signalingCh,
		}
		t.Channels[conn.RemoteAddr().String()] = channels
		router := prouter.NewMessageRouter(conn)
		router.AddRoute(heartbeatCh, func(msg proto.Message) bool {
			_, ok := msg.(*protocol.NetworkMessage).MessageType.(*protocol.NetworkMessage_Heartbeat)
			return ok
		})
		router.AddRoute(goodbyeCh, func(msg proto.Message) bool {
			_, ok := msg.(*protocol.NetworkMessage).MessageType.(*protocol.NetworkMessage_Goodbye)
			return ok
		})
		router.AddRoute(announceCh, func(msg proto.Message) bool {
			_, ok := msg.(*protocol.NetworkMessage).MessageType.(*protocol.NetworkMessage_Announce)
			return ok
		})
		router.AddRoute(peerListRequestCh, func(msg proto.Message) bool {
			_, ok := msg.(*protocol.NetworkMessage).MessageType.(*protocol.NetworkMessage_PeerListRequest)
			return ok
		})
		router.AddRoute(signalingCh, func(msg proto.Message) bool {
			_, ok := msg.(*protocol.NetworkMessage).MessageType.(*protocol.NetworkMessage_Signaling)
			return ok
		})

		// Listen for incoming msgs
		go router.Start()
		go t.handleDaemonMsgs(router)
	}
}

func (t *Tracker) handleDaemonMsgs(router *prouter.MessageRouter) {
	daemonAddr := router.Conn.RemoteAddr().String()
	daemonIP, daemonPort, err := net.SplitHostPort(router.Conn.RemoteAddr().String())
	if err != nil {
		t.Logger.Warnf("Error parsing daemon address: %v", err)
		return
	}
	t.Logger.Infof("New peer connected: %s:%s", daemonIP, daemonPort)
	daemonID, err := t.PeerStore.CreatePeer(daemonIP, daemonPort)
	if err != nil {
		t.Logger.Warnf("Error creating peer: %v", err)
		return
	}

	idNetMsg := &protocol.NetworkMessage{
		MessageType: &protocol.NetworkMessage_Id{
			Id: &protocol.IDMessage{
				Id: daemonID,
			},
		},
	}
	if err := router.WriteMessage(idNetMsg); err != nil {
		t.Logger.Warnf("Error sending ID message: %v", err)
		return
	}
	t.Logger.Infof("Sent %s back its ID: %s", daemonAddr, daemonID)

	t.DaemonRouters[daemonID] = router

	// Start a timer to track client timeout
	timeout := time.NewTicker(ClientTimeout * time.Second)
	defer timeout.Stop()
	channels, exists := t.Channels[daemonAddr]
	if !exists {
		t.Logger.Warn("Daemon channels do not exist")
		return
	}

	for {
		select {
		// If the client times out, delete the client from the db
		case <-timeout.C:
			t.Logger.Infof("Peer %s timed out, deleting", daemonAddr)
			delete(t.DaemonRouters, daemonID)
			if err := t.PeerStore.DeletePeer(daemonIP, daemonPort); err != nil {
				t.Logger.Warnf("Error deleting peer: %v", err)
			}
			return

		case signalingMsg := <-channels.SignalingCh:
			targetPeerID := signalingMsg.Signaling.GetTargetPeerId()
			forwardMsg := &protocol.NetworkMessage{
				MessageType: &protocol.NetworkMessage_Signaling{
					Signaling: &protocol.SignalingMessage{
						SourcePeerId: daemonID,
						TargetPeerId: targetPeerID,
						Message:      signalingMsg.Signaling.GetMessage(),
					},
				},
			}

			if targetRouter, ok := t.DaemonRouters[targetPeerID]; ok {
				if err := targetRouter.WriteMessage(forwardMsg); err != nil {
					t.Logger.Warnf("Error forwarding signaling message: %v", err)
				}
			}

		case <-channels.HeartbeatCh:
			timeout.Reset(ClientTimeout * time.Second)

		case <-channels.GoodbyeCh:
			// Remove the client from db
			if err := t.PeerStore.DeletePeer(daemonIP, daemonPort); err != nil {
				t.Logger.Warnf("Error deleting peer on goodbye: %v", err)
			}
			t.Logger.Infof("Peer %s disconnected", daemonAddr)
			return

		case announce := <-channels.AnnounceCh:
			t.Logger.Debugf("Received Announce message from %s: %v", daemonAddr, announce.Announce)
			files := announce.Announce.GetFiles()
			for _, file := range files {
				totalChunks := int(file.GetTotalChunks())
				maxChunkSize := int(file.GetChunkSize())
				schemaFile := &schema.File{
					CreatedAt:    time.Now().Unix(),
					Hash:         file.GetFileHash(),
					MaxChunkSize: maxChunkSize,
					Name:         file.GetFileName(),
					Size:         file.GetFileSize(),
					TotalChunks:  totalChunks,
				}
				created, createErr := t.FileStore.CreateFile(schemaFile)
				if createErr != nil {
					t.Logger.Fatalf("Error creating file: %v", createErr)
				}
				if !created {
					t.Logger.Debugf("File %+v already exists, adding peer to swarm", schemaFile)
				} else {
					t.Logger.Debugf("Registered file: %s, Size: %d, Chunks: %d Max Chunk Size: %d",
						file.GetFileHash(), file.GetFileSize(), file.GetTotalChunks(), file.GetChunkSize())
				}
				// Add client to swarm of peers
				if swarmErr := t.PeerStore.AddPeerToSwarm(daemonIP, daemonPort, file.GetFileHash()); swarmErr != nil {
					t.Logger.Warnf("Error adding peer to swarm: %v", swarmErr)
				}
			}

		case peerListRequest := <-channels.PeerListRequestCh:
			t.handlePeerListRequest(router, daemonAddr, daemonIP, daemonPort, peerListRequest)
		}
	}
}

func (t *Tracker) handlePeerListRequest(router *prouter.MessageRouter, daemonAddr, daemonIP, daemonPort string, peerListRequest *protocol.NetworkMessage_PeerListRequest) {
	t.Logger.Debugf("Received peer list request from %s: %v", daemonAddr, peerListRequest.PeerListRequest)
	fileHash := peerListRequest.PeerListRequest.GetFileHash()

	dbPeers, err := t.PeerStore.GetPeersByFileHash(fileHash)
	if err != nil {
		t.Logger.Errorf("Error getting peers: %v", err)
		return
	}

	peers := make([]*protocol.PeerInfo, 0, len(dbPeers))
	for _, peer := range dbPeers {
		if peer.IPAddress == daemonIP && peer.Port == daemonPort {
			t.Logger.Debugf("Skipping self: %s", daemonAddr)
			continue
		}

		peers = append(peers, &protocol.PeerInfo{
			Id: strconv.Itoa(int(peer.ID)),
		})
	}

	peerListResponse := &protocol.PeerListResponse{
		ChunkSize:   0,
		FileHash:    fileHash,
		Peers:       peers,
		TotalChunks: 0,
	}
	response := &protocol.NetworkMessage{
		MessageType: &protocol.NetworkMessage_PeerListResponse{
			PeerListResponse: peerListResponse,
		},
	}
	t.Logger.Debugf("Sending back the peers list to %s: %+v", daemonAddr, peerListResponse)
	if err := router.WriteMessage(response); err != nil {
		t.Logger.Warnf("Error sending peer list response: %v", err)
	}
}

func (t *Tracker) Stop() {
	t.Logger.Infof("Stopping the tracker...")
	if err := t.PeerStore.DropAllPeers(); err != nil {
		t.Logger.Warnf("Error deleting all peers: %v", err)
	}
}
