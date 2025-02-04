package cmd

import (
	"context"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/rudransh-shrivastava/peer-it/internal/client/db"
	"github.com/rudransh-shrivastava/peer-it/internal/shared/protocol"
	"github.com/rudransh-shrivastava/peer-it/internal/shared/prouter"
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

	FileStore *store.FileStore

	LocalAddr        string // Local Peer Listen Addr
	PublicIP         string
	PublicListenPort string
	Mode             string // can be dev or prod
	IPCSocketIndex   string
	PendingRequests  map[string]net.Conn

	Logger *logrus.Logger

	TrackerPeerListResponseCh chan *protocol.NetworkMessage_PeerListResponse
}

func newDaemon(ctx context.Context, trackerAddr string, ipcSocketIndex string, localAddr string, mode string) (*Daemon, error) {
	db, err := db.NewDB()
	if err != nil {
		return &Daemon{}, err
	}
	fileStore := store.NewFileStore(db)
	logger := logger.NewLogger()
	conn, err := net.Dial("tcp", trackerAddr)
	if err != nil {
		return &Daemon{}, err
	}

	peerListResponseCh := make(chan *protocol.NetworkMessage_PeerListResponse, 100)

	trackerProuter := prouter.NewMessageRouter(conn)
	trackerProuter.AddRoute(peerListResponseCh, func(msg proto.Message) bool {
		_, ok := msg.(*protocol.NetworkMessage).MessageType.(*protocol.NetworkMessage_PeerListResponse)
		return ok
	})
	trackerProuter.Start()

	return &Daemon{
		Ctx:                       ctx,
		TrackerRouter:             trackerProuter,
		FileStore:                 fileStore,
		PendingRequests:           make(map[string]net.Conn),
		IPCSocketIndex:            ipcSocketIndex,
		LocalAddr:                 localAddr,
		Mode:                      mode, // Can be dev or prod
		Logger:                    logger,
		TrackerPeerListResponseCh: peerListResponseCh,
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
		go d.handleCLIMsgs(cliConn)
	}
}

func (d *Daemon) handleCLIMsgs(conn net.Conn) {
	netMsg, err := utils.UnsafeReceiveNetMsg(conn)
	if err != nil {
		d.Logger.Fatal(err)
	}
	// TODO: Send announce first thing
	// TODO: Change implementation how announce from cli is handled, make new msg type of register fiel or somethigns
	switch msg := netMsg.MessageType.(type) {
	case *protocol.NetworkMessage_Announce:
		// Transfer the announce message to the tracker
		d.Logger.Debugf("Received Announce message from CLI: %+v", msg.Announce)
		d.Logger.Debugf("Sending Announce message to Tracker: %+v ", msg.Announce)
		netMsg := &protocol.NetworkMessage{
			MessageType: &protocol.NetworkMessage_Announce{
				Announce: msg.Announce,
			},
		}
		err := d.TrackerRouter.WriteMessage(netMsg)
		if err != nil {
			d.Logger.Warnf("Error sending message to Tracker: %v", err)
		}

	case *protocol.NetworkMessage_PeerListRequest:
		// Transfer the peer list request to the tracker
		d.Logger.Debugf("Received PeerListRequest message from CLI: %+v", msg.PeerListRequest)
		d.PendingRequests[msg.PeerListRequest.GetFileHash()] = conn
		d.Logger.Debugf("Sending PeerListRequest message to Tracker: %+v", msg.PeerListRequest)

		netMsg := &protocol.NetworkMessage{
			MessageType: &protocol.NetworkMessage_PeerListRequest{
				PeerListRequest: msg.PeerListRequest,
			},
		}
		err := d.TrackerRouter.WriteMessage(netMsg)
		if err != nil {
			d.Logger.Warnf("Error sending message to Tracker: %v", err)
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
			cliConn, exists := d.PendingRequests[peerlistResponse.PeerListResponse.GetFileHash()]
			if !exists {
				d.Logger.Warnf("No Requests for file hash: %s", peerlistResponse.PeerListResponse.GetFileHash())
			}
			delete(d.PendingRequests, peerlistResponse.PeerListResponse.GetFileHash())

			netMsg := &protocol.NetworkMessage{
				MessageType: &protocol.NetworkMessage_PeerListResponse{
					PeerListResponse: peerlistResponse.PeerListResponse,
				},
			}

			err := utils.SendNetMsg(cliConn, netMsg)
			if err != nil {
				d.Logger.Warnf("Error sending message back to daemon: %v", err)
			}
			d.Logger.Info("Sent peer list response to CLI")
		}
	}
}

// func (d *Daemon) listenPeerConn() {
// 	listen, err := net.Listen("tcp", d.LocalAddr)
// 	if err != nil {
// 		d.Logger.Fatalf("Error starting Peer TCP listener: %+v", err)
// 		return
// 	}
// 	defer listen.Close()

// 	d.Logger.Infof("Listening for peers on addr: %s", d.LocalAddr)

// 	for {
// 		conn, err := listen.Accept()
// 		if err != nil {
// 			d.Logger.Warnf("Error accepting connection: %+v", err)
// 			continue
// 		}
// 		// Listen for incoming msgs
// 		go d.handlePeerMsgs(conn)
// 	}
// }

func (d *Daemon) initConnMsgs() {
	// Send a Register message to the tracker so that the tracker can save our public listneer port
	// Send the daemon port if running as development
	// Hit a STUN server and send the public port if running as production
	d.Logger.Info("Sending Register message to Tracker")
	d.PublicListenPort = strings.Split(d.LocalAddr, ":")[1]
	d.PublicIP = "localhost"
	if d.Mode == "prod" {
		// Hit a STUN server and get the public port
		// TODO: Implement this
		d.Logger.Info("Trying to hit STUN servers to get public listen port")
	}
	// Send a Register message to the tracker the first time we connect
	// The tracker will register our public IP:PORT
	registerMsg := &protocol.RegisterMessage{
		PublicIpAddress: d.PublicIP,
		ListenPort:      d.PublicListenPort,
	}
	netMsg := &protocol.NetworkMessage{
		MessageType: &protocol.NetworkMessage_Register{
			Register: registerMsg,
		},
	}
	err := d.TrackerRouter.WriteMessage(netMsg)
	if err != nil {
		d.Logger.Warnf("Error sending message to Tracker: %v", err)
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
