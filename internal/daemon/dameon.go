package daemon

import (
	"context"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/pion/webrtc/v3"
	"github.com/rudransh-shrivastava/peer-it/internal/client/db"
	"github.com/rudransh-shrivastava/peer-it/internal/shared/protocol"
	"github.com/rudransh-shrivastava/peer-it/internal/shared/prouter"
	"github.com/rudransh-shrivastava/peer-it/internal/shared/store"
	"github.com/rudransh-shrivastava/peer-it/internal/shared/utils/logger"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/proto"
)

const maxChunkSize = 256 * 1024
const heartbeatInterval = 5

type Daemon struct {
	Ctx context.Context
	ID  string

	TrackerRouter *prouter.MessageRouter

	FileStore  *store.FileStore
	ChunkStore *store.ChunkStore

	IPCSocketIndex          string
	PendingPeerListRequests map[string]chan *protocol.NetworkMessage_PeerListResponse

	Logger *logrus.Logger

	TrackerPeerListResponseCh chan *protocol.NetworkMessage_PeerListResponse
	TrackerIdMessageCh        chan *protocol.NetworkMessage_Id
	TrackerSignalingCh        chan *protocol.NetworkMessage_Signaling

	CLISignalRegisterCh chan *protocol.NetworkMessage_SignalRegister
	CLISignalDownloadCh chan *protocol.NetworkMessage_SignalDownload

	PeerConnections  map[string]*webrtc.PeerConnection
	PeerDataChannels map[string]*webrtc.DataChannel
	PeerChunkMap     map[string]map[string][]int32

	mu sync.Mutex
}

func NewDaemon(ctx context.Context, trackerAddr string, ipcSocketIndex string) (*Daemon, error) {
	db, err := db.NewDB(ipcSocketIndex)
	if err != nil {
		return &Daemon{}, err
	}
	fileStore := store.NewFileStore(db)
	chunkStore := store.NewChunkStore(db)

	logger := logger.NewLogger()
	conn, err := net.Dial("tcp", trackerAddr)
	if err != nil {
		return &Daemon{}, err
	}

	peerListResponseCh := make(chan *protocol.NetworkMessage_PeerListResponse, 100)
	idMessageCh := make(chan *protocol.NetworkMessage_Id, 100)
	signalingMsgCh := make(chan *protocol.NetworkMessage_Signaling, 100)

	signalRegisterCh := make(chan *protocol.NetworkMessage_SignalRegister, 100)
	signalDownloadCh := make(chan *protocol.NetworkMessage_SignalDownload, 100)

	trackerProuter := prouter.NewMessageRouter(conn)
	trackerProuter.AddRoute(peerListResponseCh, func(msg proto.Message) bool {
		_, ok := msg.(*protocol.NetworkMessage).MessageType.(*protocol.NetworkMessage_PeerListResponse)
		return ok
	})
	trackerProuter.AddRoute(idMessageCh, func(msg proto.Message) bool {
		_, ok := msg.(*protocol.NetworkMessage).MessageType.(*protocol.NetworkMessage_Id)
		return ok
	})
	trackerProuter.AddRoute(signalingMsgCh, func(msg proto.Message) bool {
		_, ok := msg.(*protocol.NetworkMessage).MessageType.(*protocol.NetworkMessage_Signaling)
		return ok
	})

	trackerProuter.Start()

	return &Daemon{
		Ctx:           ctx,
		TrackerRouter: trackerProuter,
		FileStore:     fileStore,
		ChunkStore:    chunkStore,
		// Pending requests maps a file hash with a channel that the listener
		// will send the messsages to
		PendingPeerListRequests:   make(map[string]chan *protocol.NetworkMessage_PeerListResponse),
		IPCSocketIndex:            ipcSocketIndex,
		Logger:                    logger,
		TrackerPeerListResponseCh: peerListResponseCh,
		TrackerIdMessageCh:        idMessageCh,
		TrackerSignalingCh:        signalingMsgCh,
		CLISignalRegisterCh:       signalRegisterCh,
		CLISignalDownloadCh:       signalDownloadCh,
		PeerConnections:           make(map[string]*webrtc.PeerConnection),
		PeerDataChannels:          make(map[string]*webrtc.DataChannel),
		PeerChunkMap:              make(map[string]map[string][]int32),
	}, nil
}

func (d *Daemon) Start() {
	// Set up signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)

	d.Logger.Info("Daemon starting...")

	go d.startIPCServer()
	go d.handleTrackerMsgs()

	d.initDaemon()

	d.Logger.Info("Daemon is now running...")

	<-sigChan
	d.Logger.Info("Shutting down daemon...")

	d.Logger.Info("Daemon stopped")
}

func (d *Daemon) initDaemon() {
	// Send an Announce message to the tracker the first time we connect
	files, err := d.FileStore.GetFiles()
	if err != nil {
		d.Logger.Fatalf("Error getting files: %+v", err)
		return
	}
	fileInfoMsgs := make([]*protocol.FileInfo, 0)
	for _, file := range files {
		fileInfoMsgs = append(fileInfoMsgs, &protocol.FileInfo{
			FileSize:    file.Size,
			ChunkSize:   int32(file.MaxChunkSize),
			FileHash:    file.Hash,
			TotalChunks: int32(file.TotalChunks),
		})
	}
	announceMsg := &protocol.AnnounceMessage{
		Files: fileInfoMsgs,
	}

	announceNetMsg := &protocol.NetworkMessage{
		MessageType: &protocol.NetworkMessage_Announce{
			Announce: announceMsg,
		},
	}
	err = d.TrackerRouter.WriteMessage(announceNetMsg)
	if err != nil {
		d.Logger.Warnf("Error sending message to Tracker: %v", err)
	}

	d.Logger.Debugf("Sent initial Announce message to Tracker: %+v", announceMsg)
	// Send heartbeats every n seconds
	go d.sendHeartBeatsToTracker()
}
