package cmd

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"math"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/rudransh-shrivastava/peer-it/internal/client/db"
	"github.com/rudransh-shrivastava/peer-it/internal/shared/protocol"
	"github.com/rudransh-shrivastava/peer-it/internal/shared/prouter"
	"github.com/rudransh-shrivastava/peer-it/internal/shared/schema"
	"github.com/rudransh-shrivastava/peer-it/internal/shared/store"
	"github.com/rudransh-shrivastava/peer-it/internal/shared/utils"
	"github.com/rudransh-shrivastava/peer-it/internal/shared/utils/logger"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/proto"
)

type Daemon struct {
	Ctx context.Context

	TrackerRouter *prouter.MessageRouter

	FileStore  *store.FileStore
	ChunkStore *store.ChunkStore

	LocalAddr               string // Local Peer Listen Addr
	PublicIP                string
	PublicListenPort        string
	Mode                    string // can be dev or prod
	IPCSocketIndex          string
	PendingPeerListRequests map[string]chan *protocol.NetworkMessage_PeerListResponse

	Logger *logrus.Logger

	TrackerPeerListResponseCh chan *protocol.NetworkMessage_PeerListResponse

	CLISignalRegisterCh chan *protocol.NetworkMessage_SignalRegister
	CLISignalDownloadCh chan *protocol.NetworkMessage_SignalDownload
}

func newDaemon(ctx context.Context, trackerAddr string, ipcSocketIndex string, localAddr string, mode string) (*Daemon, error) {
	db, err := db.NewDB()
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
	signalRegisterCh := make(chan *protocol.NetworkMessage_SignalRegister, 100)
	signalDownloadCh := make(chan *protocol.NetworkMessage_SignalDownload, 100)

	trackerProuter := prouter.NewMessageRouter(conn)
	trackerProuter.AddRoute(peerListResponseCh, func(msg proto.Message) bool {
		_, ok := msg.(*protocol.NetworkMessage).MessageType.(*protocol.NetworkMessage_PeerListResponse)
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
		LocalAddr:                 localAddr,
		Mode:                      mode, // Can be dev or prod
		Logger:                    logger,
		TrackerPeerListResponseCh: peerListResponseCh,
		CLISignalRegisterCh:       signalRegisterCh,
		CLISignalDownloadCh:       signalDownloadCh,
	}, nil
}

var daemonCmd = &cobra.Command{
	Use:   "daemon port ipc-socket-index remote-address mode(dev/prod)",
	Short: "runs peer-it daemon",
	Long:  `runs the peer-it daemon in the background, the daemon uses unix sockets to communicate with the CLI`,
	Args:  cobra.ExactArgs(4),
	Run: func(cmd *cobra.Command, args []string) {
		daemonPort := args[0]
		daemonAddr := "localhost:8080"
		if daemonPort != "" {
			daemonAddr = "localhost" + ":" + daemonPort
		}
		logger := logger.NewLogger()
		logger.Debugf("Daemon port: %s", daemonPort)
		ipcSocketIndex := args[1]
		logger.Debugf("IPC Socket Index: %s", ipcSocketIndex)
		trackerAddr := args[2]
		logger.Debugf("Tracker Address: %s", trackerAddr)
		daemonMode := args[3]
		if daemonMode == "prod" {
			// Run in production mode
			logger.Info("Running Daemon in Production Mode")
		} else {
			// Run in development mode
			daemonMode = "dev"
			logger.Info("Running Daemon in Development Mode")
		}
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		daemon, err := newDaemon(ctx, trackerAddr, ipcSocketIndex, daemonAddr, daemonMode)
		if err != nil {
			logger.Fatal(err)
			return
		}
		daemon.startDaemon()
	},
}

func (d *Daemon) startDaemon() {
	// Set up signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)

	d.Logger.Info("Daemon starting...")

	go d.startIPCServer()
	go d.handleTrackerMsgs()
	// go d.listenPeerConn()

	d.initConnMsgs()

	d.Logger.Infof("Daemon ready, running on %s", d.LocalAddr)

	<-sigChan
	d.Logger.Info("Shutting down daemon...")

	d.Logger.Info("Daemon stopped")
}

func (d *Daemon) startIPCServer() {
	socketUrl := "/tmp/pit-daemon-" + d.IPCSocketIndex + ".sock"
	os.Remove(socketUrl)

	l, err := net.Listen("unix", socketUrl)
	if err != nil {
		panic(err)
	}
	d.Logger.Info("IPC Server started successfuly")
	for {
		cliConn, err := l.Accept()
		d.Logger.Info("Accepted a new socket connection")
		if err != nil {
			continue
		}
		cliRouter := prouter.NewMessageRouter(cliConn)
		cliRouter.AddRoute(d.CLISignalRegisterCh, func(msg proto.Message) bool {
			_, ok := msg.(*protocol.NetworkMessage).MessageType.(*protocol.NetworkMessage_SignalRegister)
			return ok
		})
		cliRouter.AddRoute(d.CLISignalDownloadCh, func(msg proto.Message) bool {
			_, ok := msg.(*protocol.NetworkMessage).MessageType.(*protocol.NetworkMessage_SignalDownload)
			return ok
		})

		go cliRouter.Start()
		go d.handleCLIMsgs(cliRouter) // problem here // using multiple go routines for same channels
	}
}

func (d *Daemon) handleCLIMsgs(msgRouter *prouter.MessageRouter) {
	cliAddr := msgRouter.Conn.RemoteAddr().String()
	for {
		select {
		case <-d.Ctx.Done():
			d.Logger.Info("Stopping the CLI message listener")
			return
		case downloadSignal := <-d.CLISignalDownloadCh:
			d.Logger.Debugf("Received a new Download Signal from %s: %v", cliAddr, downloadSignal)
			fileHash := downloadSignal.SignalDownload.GetFileHash()
			peerListReqMsg := &protocol.NetworkMessage{
				MessageType: &protocol.NetworkMessage_PeerListRequest{
					PeerListRequest: &protocol.PeerListRequest{
						FileHash: fileHash,
					},
				},
			}
			channel, exists := d.PendingPeerListRequests[fileHash]
			if !exists {
				channel = make(chan *protocol.NetworkMessage_PeerListResponse, 100)
				d.PendingPeerListRequests[fileHash] = channel
			}

			err := d.TrackerRouter.WriteMessage(peerListReqMsg)
			if err != nil {
				d.Logger.Warnf("Error sending message to Tracker: %v", err)
			}

			select {
			case peerListResponse := <-channel:
				d.Logger.Debugf("Received peer list response from tracker using a channel: %v", peerListResponse)
				if fileHash != peerListResponse.PeerListResponse.GetFileHash() {
					d.Logger.Warnf("Response file hash does not match requested file hash, response: %s requested: %s",
						peerListResponse.PeerListResponse.GetFileHash(), fileHash)
					continue
				}
				peers := peerListResponse.PeerListResponse.GetPeers()
				d.Logger.Debugf("Received peer list %+v", peers)

				fileInfo, err := d.FileStore.GetFileByHash(fileHash)
				if err != nil {
					d.Logger.Warnf("Error getting file info: %v", err)
					continue
				}

				chunksMap := make([]int32, fileInfo.TotalChunks)
				fileChunks, err := d.FileStore.GetChunks(fileHash)
				if err != nil {
					d.Logger.Warnf("Error getting chunks: %v ", err)
					continue
				}
				for _, chunk := range fileChunks {
					chunksMap[chunk.Index] = 1
				}
			case <-time.After(10 * time.Second):
				d.Logger.Warn("Timeout waiting for peer list response")
				continue
			}

		case registerSignal := <-d.CLISignalRegisterCh:
			filePath := registerSignal.SignalRegister.GetFilePath()
			d.Logger.Debugf("Received a new Register Signal from %s: %v", cliAddr, registerSignal)
			file, err := os.Open(filePath)
			if err != nil {
				d.Logger.Warnf("Error opening file: %v", err)
				return
			}
			defer file.Close()

			fileName := strings.Split(file.Name(), "/")[len(strings.Split(file.Name(), "/"))-1]
			fileInfo, err := file.Stat()
			if err != nil {
				d.Logger.Warnf("Error getting file info: %v", err)
				return
			}
			fileSize := fileInfo.Size()
			fileTotalChunks := int((fileSize + maxChunkSize - 1) / maxChunkSize)
			hash := sha256.New()
			if _, err := io.Copy(hash, file); err != nil {
				d.Logger.Warnf("Error copying file hash to hash: %v", err)
				return
			}
			fileHash := fmt.Sprintf("%x", hash.Sum(nil))

			schemaFile := schema.File{
				Size:         fileSize,
				MaxChunkSize: maxChunkSize,
				TotalChunks:  fileTotalChunks,
				Hash:         fileHash,
				CreatedAt:    time.Now().Unix(),
			}
			// log the stats of the file
			d.Logger.Debugf("file: %s with size %d and hash %s\n", fileName, fileSize, fileHash)
			created, err := d.FileStore.CreateFile(&schemaFile)
			if !created {
				d.Logger.Info("file already registered with tracker")
				return
			}
			if err != nil {
				d.Logger.Warnf("Error creating file: %v", err)
				return
			}

			d.Logger.Info("Attempting to create chunks with metadata...")

			buffer := make([]byte, maxChunkSize)
			chunkIndex := 0
			file.Seek(0, 0)
			for {
				n, err := file.Read(buffer)
				if err != nil && err != io.EOF {
					d.Logger.Warnf("Error reading buffer: %v", err)
					return
				}
				if n == 0 {
					break
				}

				hash := generateHash(buffer[:n])
				err = d.ChunkStore.CreateChunk(&schemaFile, n, chunkIndex, hash, false)
				if err != nil {
					d.Logger.Warnf("Error creating chunk: %v", err)
					return
				}
				d.Logger.Debugf("chunk %d: %s with size %d\n", chunkIndex, hash, n)
				chunkIndex++
			}

			// copy the file to downlaoads/complete
			downloadDirPath := fmt.Sprintf("downloads/daemon-%s/", d.IPCSocketIndex)
			err = os.MkdirAll(downloadDirPath, os.ModePerm)
			if err != nil {
				d.Logger.Warnf("Error creating directory: %v", err)
				return
			}

			createdFile, err := os.Create(downloadDirPath + fileName)
			if err != nil {
				d.Logger.Warnf("Error creating file: %v", err)
				return
			}
			defer createdFile.Close()

			file.Seek(0, 0)
			_, err = io.Copy(createdFile, file)
			if err != nil {
				d.Logger.Warnf("Error copying file: %v", err)
				return
			}

			// send announce message to tracker to tell it to add the file
			d.Logger.Infof("Preparing to send Announce message to Tracker with newly created file")

			fileInfoMsgs := make([]*protocol.FileInfo, 0)
			fileInfoMsgs = append(fileInfoMsgs, &protocol.FileInfo{
				FileSize:    schemaFile.Size,
				ChunkSize:   int32(schemaFile.MaxChunkSize),
				FileHash:    schemaFile.Hash,
				TotalChunks: int32(schemaFile.TotalChunks),
			})

			// announceMsg := &protocol.AnnounceMessage{
			// 	Files: fileInfoMsgs,
			// }
			d.Logger.Warn("START THE REGISTER PROCESS ")
		}
	}
}

func (d *Daemon) handleTrackerMsgs() {
	d.Logger.Info("Connected to tracker server")
	for {
		select {
		case <-d.Ctx.Done():
			d.Logger.Info("Stopping the tracker message listener")
			return
		case peerlistResponse := <-d.TrackerPeerListResponseCh:
			d.Logger.Debugf("Received peer list response from tracker: %+v", peerlistResponse.PeerListResponse)
			channel, exists := d.PendingPeerListRequests[peerlistResponse.PeerListResponse.GetFileHash()]
			if !exists {
				d.Logger.Warnf("No Requests for file hash: %s", peerlistResponse.PeerListResponse.GetFileHash())
			}

			responseMsg := &protocol.NetworkMessage_PeerListResponse{
				PeerListResponse: peerlistResponse.PeerListResponse,
			}

			channel <- responseMsg
			d.Logger.Info("Sent peer list response to CLI")
		}
	}
}

func (d *Daemon) initConnMsgs() {
	// Send a Register message to the tracker so that the tracker can save our public listneer port
	// Send the daemon port if running as development
	// Hit a STUN server and send the public port if running as production

	d.Logger.Info("Sending Register message to Tracker")
	d.PublicListenPort = strings.Split(d.LocalAddr, ":")[1]
	d.PublicIP = "::1"
	if d.Mode == "prod" {
		// Hit a STUN server and get the public port
		// TODO: Implement this
		d.Logger.Info("Trying to hit STUN servers to get public listen port")
	}
	// Send a Register message to the tracker the first time we connect
	// The tracker will register our public IP:PORT
	maxRetries := 3
	registerMsg := &protocol.RegisterMessage{
		PublicIpAddress: d.PublicIP,
		ListenPort:      d.PublicListenPort,
		ClientId:        uuid.New().String(),
	}

	for i := 0; i < maxRetries; i++ {
		netMsg := &protocol.NetworkMessage{
			MessageType: &protocol.NetworkMessage_Register{
				Register: registerMsg,
			},
		}

		// err := d.TrackerRouter.WriteMessage(netMsg)
		// if err == nil {
		// 	break
		// }
		err := utils.SendNetMsg(d.TrackerRouter.Conn, netMsg)
		if err == nil {
			break
		}

		d.Logger.Warnf("Registration attempt %d failed: %v", i+1, err)
		time.Sleep(time.Duration(math.Pow(2, float64(i))) * time.Second)
	}
	d.Logger.Debugf("Sent Register message to Tracker: %+v", registerMsg)

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

func (d *Daemon) sendHeartBeatsToTracker() {
	ticker := time.NewTicker(heartbeatInterval * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-d.Ctx.Done():
			d.Logger.Info("Stopping the heart")
			return
		case <-ticker.C:
			d.Logger.Info("Sending a heartbeat")
			hb := &protocol.HeartbeatMessage{
				Timestamp: time.Now().Unix(),
			}

			netMsg := &protocol.NetworkMessage{
				MessageType: &protocol.NetworkMessage_Heartbeat{
					Heartbeat: hb,
				},
			}

			err := d.TrackerRouter.WriteMessage(netMsg)
			if err != nil {
				d.Logger.Warnf("Error sending message to Tracker: %v", err)
			}
		}
	}
}
