package node

import (
	"context"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/pion/webrtc/v3"
	"github.com/rudransh-shrivastava/peer-it/internal/client/db"
	internaldb "github.com/rudransh-shrivastava/peer-it/internal/db"
	"github.com/rudransh-shrivastava/peer-it/internal/shared/protocol"
	"github.com/rudransh-shrivastava/peer-it/internal/shared/prouter"
	"github.com/rudransh-shrivastava/peer-it/internal/shared/utils/logger"
	"github.com/rudransh-shrivastava/peer-it/internal/store"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/proto"
)

const (
	maxChunkSize      = 32 * 1024
	heartbeatInterval = 5
)

type FileRepository interface {
	CreateFile(ctx context.Context, name string, size int64, maxChunkSize, totalChunks int, hash string) (internaldb.File, bool, error)
	GetFiles(ctx context.Context) ([]internaldb.File, error)
	GetFileByHash(ctx context.Context, hash string) (internaldb.File, error)
	GetFileNameByHash(ctx context.Context, hash string) (string, error)
}

type ChunkRepository interface {
	CreateChunk(ctx context.Context, fileID int64, index, size int, hash string, isAvailable bool) (internaldb.Chunk, error)
	GetChunk(ctx context.Context, fileHash string, chunkIndex int) (internaldb.Chunk, error)
	GetChunks(ctx context.Context, fileHash string) ([]internaldb.Chunk, error)
	MarkChunkAvailable(ctx context.Context, fileHash string, chunkIndex int) error
}

type Options struct {
	TrackerAddr    string
	IPCSocketIndex string
	Files          FileRepository
	Chunks         ChunkRepository
	Logger         *logrus.Logger
}

type Node struct {
	ctx context.Context
	id  string

	trackerRouter *prouter.MessageRouter
	cliRouter     *prouter.MessageRouter
	files         FileRepository
	chunks        ChunkRepository

	ipcSocketIndex          string
	pendingPeerListRequests map[string]chan *protocol.NetworkMessage_PeerListResponse

	logger *logrus.Logger

	trackerPeerListResponseCh chan *protocol.NetworkMessage_PeerListResponse
	trackerIDMessageCh        chan *protocol.NetworkMessage_Id
	trackerSignalingCh        chan *protocol.NetworkMessage_Signaling

	cliSignalRegisterCh chan *protocol.NetworkMessage_SignalRegister
	channels            map[string]Channels
	cliSignalDownloadCh chan *protocol.NetworkMessage_SignalDownload

	peerConnections  map[string]*webrtc.PeerConnection
	peerDataChannels map[string]*webrtc.DataChannel
	peerChunkMap     map[string]map[string][]int32
	activeDownloads  map[string]*FileDownload

	mu sync.Mutex
}

type Channels struct {
	LogCh     chan *protocol.NetworkMessage_Log
	GoodbyeCh chan *protocol.NetworkMessage_Goodbye
}

func New(ctx context.Context, opts Options) (*Node, error) {
	var files FileRepository
	var chunks ChunkRepository

	if opts.Files != nil {
		files = opts.Files
		chunks = opts.Chunks
	} else {
		sqlDB, err := db.NewDB(opts.IPCSocketIndex)
		if err != nil {
			return nil, err
		}
		files = store.NewFileStore(sqlDB)
		chunks = store.NewChunkStore(sqlDB)
	}

	log := opts.Logger
	if log == nil {
		log = logger.NewLogger()
	}

	conn, err := net.Dial("tcp", opts.TrackerAddr)
	if err != nil {
		return nil, err
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

	return &Node{
		ctx:                       ctx,
		trackerRouter:             trackerProuter,
		files:                     files,
		chunks:                    chunks,
		pendingPeerListRequests:   make(map[string]chan *protocol.NetworkMessage_PeerListResponse),
		ipcSocketIndex:            opts.IPCSocketIndex,
		logger:                    log,
		trackerPeerListResponseCh: peerListResponseCh,
		trackerIDMessageCh:        idMessageCh,
		trackerSignalingCh:        signalingMsgCh,
		channels:                  make(map[string]Channels),
		cliSignalRegisterCh:       signalRegisterCh,
		cliSignalDownloadCh:       signalDownloadCh,
		peerConnections:           make(map[string]*webrtc.PeerConnection),
		peerDataChannels:          make(map[string]*webrtc.DataChannel),
		peerChunkMap:              make(map[string]map[string][]int32),
		activeDownloads:           make(map[string]*FileDownload),
	}, nil
}

func (n *Node) Start() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)

	n.logger.Info("Node starting...")

	go n.startIPCServer()
	go n.handleTrackerMsgs()

	n.init()

	n.logger.Info("Node is now running...")

	<-sigChan
	n.logger.Info("Shutting down node...")

	n.logger.Info("Node stopped")
}

func (n *Node) init() {
	ctx := context.Background()

	fileList, err := n.files.GetFiles(ctx)
	if err != nil {
		n.logger.Fatalf("Error getting files: %+v", err)
		return
	}
	for _, file := range fileList {
		chunkList, err := n.chunks.GetChunks(ctx, file.Hash)
		if err != nil {
			n.logger.Fatalf("Error getting chunks: %+v", err)
			return
		}
		chunkMap := make([]int32, file.TotalChunks)
		for _, chunk := range chunkList {
			if chunk.IsAvailable == 1 {
				chunkMap[chunk.ChunkIndex] = 1
			}
		}
		n.mu.Lock()
		if _, exists := n.peerChunkMap[n.id]; !exists {
			n.peerChunkMap[n.id] = make(map[string][]int32)
		}
		n.peerChunkMap[n.id][file.Hash] = chunkMap
		n.mu.Unlock()
	}

	fileInfoMsgs := make([]*protocol.FileInfo, 0, len(fileList))
	for _, file := range fileList {
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
	err = n.trackerRouter.WriteMessage(announceNetMsg)
	if err != nil {
		n.logger.Warnf("Error sending message to Tracker: %v", err)
	}

	n.logger.Debugf("Sent initial Announce message to Tracker: %+v", announceMsg)

	n.mu.Lock()
	n.peerChunkMap[n.id] = make(map[string][]int32)
	n.mu.Unlock()

	go n.sendHeartBeatsToTracker()
}

func (n *Node) messageCLI(msg string) {
	n.logger.Debugf("Sending LOG: %s to CLI", msg)
	cliRouter := n.cliRouter
	if cliRouter == nil {
		n.logger.Warnf("CLI router not found")
		return
	}

	if err := cliRouter.WriteMessage(BuildLogMessage(msg)); err != nil {
		n.logger.Warnf("Failed to send message to CLI: %v", err)
	}
}
