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
	"github.com/rudransh-shrivastava/peer-it/internal/shared/utils/logger"
	"github.com/rudransh-shrivastava/peer-it/internal/store"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/proto"
)

const maxChunkSize = 32 * 1024
const heartbeatInterval = 5

type Daemon struct {
	Ctx context.Context
	ID  string

	TrackerRouter *prouter.MessageRouter
	CLIRouter     *prouter.MessageRouter
	FileStore     *store.FileStore
	ChunkStore    *store.ChunkStore

	IPCSocketIndex          string
	PendingPeerListRequests map[string]chan *protocol.NetworkMessage_PeerListResponse

	Logger *logrus.Logger

	TrackerPeerListResponseCh chan *protocol.NetworkMessage_PeerListResponse
	TrackerIdMessageCh        chan *protocol.NetworkMessage_Id
	TrackerSignalingCh        chan *protocol.NetworkMessage_Signaling

	CLISignalRegisterCh chan *protocol.NetworkMessage_SignalRegister
	Channels            map[string]Channels
	CLISignalDownloadCh chan *protocol.NetworkMessage_SignalDownload

	PeerConnections  map[string]*webrtc.PeerConnection
	PeerDataChannels map[string]*webrtc.DataChannel
	PeerChunkMap     map[string]map[string][]int32
	ActiveDownloads  map[string]*FileDownload

	mu sync.Mutex
}

type Channels struct {
	LogCh     chan *protocol.NetworkMessage_Log
	GoodbyeCh chan *protocol.NetworkMessage_Goodbye
}

func NewDaemon(ctx context.Context, trackerAddr string, ipcSocketIndex string) (*Daemon, error) {
	sqlDB, err := db.NewDB(ipcSocketIndex)
	if err != nil {
		return &Daemon{}, err
	}
	fileStore := store.NewFileStore(sqlDB)
	chunkStore := store.NewChunkStore(sqlDB)

	log := logger.NewLogger()
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
		Ctx:                       ctx,
		TrackerRouter:             trackerProuter,
		FileStore:                 fileStore,
		ChunkStore:                chunkStore,
		PendingPeerListRequests:   make(map[string]chan *protocol.NetworkMessage_PeerListResponse),
		IPCSocketIndex:            ipcSocketIndex,
		Logger:                    log,
		TrackerPeerListResponseCh: peerListResponseCh,
		TrackerIdMessageCh:        idMessageCh,
		TrackerSignalingCh:        signalingMsgCh,
		Channels:                  make(map[string]Channels),
		CLISignalRegisterCh:       signalRegisterCh,
		CLISignalDownloadCh:       signalDownloadCh,
		PeerConnections:           make(map[string]*webrtc.PeerConnection),
		PeerDataChannels:          make(map[string]*webrtc.DataChannel),
		PeerChunkMap:              make(map[string]map[string][]int32),
		ActiveDownloads:           make(map[string]*FileDownload),
	}, nil
}

func (d *Daemon) Start() {
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
	ctx := context.Background()

	files, err := d.FileStore.GetFiles(ctx)
	if err != nil {
		d.Logger.Fatalf("Error getting files: %+v", err)
		return
	}
	for _, file := range files {
		chunks, err := d.ChunkStore.GetChunks(ctx, file.Hash)
		if err != nil {
			d.Logger.Fatalf("Error getting chunks: %+v", err)
			return
		}
		chunkMap := make([]int32, file.TotalChunks)
		for _, chunk := range chunks {
			if chunk.IsAvailable == 1 {
				chunkMap[chunk.ChunkIndex] = 1
			}
		}
		d.mu.Lock()
		if _, exists := d.PeerChunkMap[d.ID]; !exists {
			d.PeerChunkMap[d.ID] = make(map[string][]int32)
		}
		d.PeerChunkMap[d.ID][file.Hash] = chunkMap
		d.mu.Unlock()
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

	d.mu.Lock()
	d.PeerChunkMap[d.ID] = make(map[string][]int32)
	d.mu.Unlock()

	go d.sendHeartBeatsToTracker()
}
