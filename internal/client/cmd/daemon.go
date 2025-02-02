package cmd

import (
	"context"
	"net"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/rudransh-shrivastava/peer-it/internal/client/db"
	"github.com/rudransh-shrivastava/peer-it/internal/shared/protocol"
	"github.com/rudransh-shrivastava/peer-it/internal/shared/store"
	"github.com/rudransh-shrivastava/peer-it/internal/shared/utils"
	"github.com/rudransh-shrivastava/peer-it/internal/shared/utils/logger"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type Daemon struct {
	Ctx context.Context

	TrackerConn      net.Conn
	TrackerConnMutex sync.Mutex

	FileStore *store.FileStore

	LocalAddr        string // Local Peer Listen Addr
	PublicIP         string
	PublicListenPort string
	Mode             string // can be dev or prod
	IPCSocketIndex   string
	PendingRequests  map[string]net.Conn

	Logger *logrus.Logger
}

func newDaemon(ctx context.Context, conn net.Conn, fileStore *store.FileStore, ipcSocketIndex string, localAddr string, mode string, logger *logrus.Logger) *Daemon {
	return &Daemon{
		Ctx:             ctx,
		TrackerConn:     conn,
		FileStore:       fileStore,
		PendingRequests: make(map[string]net.Conn),
		IPCSocketIndex:  ipcSocketIndex,
		LocalAddr:       localAddr,
		Mode:            mode, // Can be dev or prod
		Logger:          logger,
	}
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

		db, err := db.NewDB()
		if err != nil {
			logger.Fatal(err)
			return
		}

		conn, err := net.Dial("tcp", trackerAddr)
		if err != nil {
			logger.Fatal(err)
			return
		}
		fileStore := store.NewFileStore(db)
		daemon := newDaemon(ctx, conn, fileStore, ipcSocketIndex, daemonAddr, daemonMode, logger)
		daemon.startDaemon()
	},
}

func (d *Daemon) startDaemon() {
	// Set up signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)

	d.Logger.Info("Daemon starting...")

	go d.startIPCServer()
	go d.listenTrackerMessages()
	go d.listenPeerConn()

	d.initConnMsgs()

	d.Logger.Infof("Daemon ready, running on %s", d.LocalAddr)

	<-sigChan
	d.Logger.Info("Shutting down daemon...")

	d.TrackerConn.Close()

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
		go d.handleCLIRequest(cliConn)
	}
}

func (d *Daemon) handleCLIRequest(conn net.Conn) {
	netMsg, err := utils.ReceiveNetMsg(conn)
	if err != nil {
		d.Logger.Fatal(err)
	}
	switch msg := netMsg.MessageType.(type) {
	case *protocol.NetworkMessage_Announce:
		// Transfer the announce message to the tracker
		d.Logger.Debugf("Received Announce message from CLI: %+v", msg.Announce)
		d.Logger.Debugf("Sending Announce message to Tracker: %+v ", msg.Announce)

		d.TrackerConnMutex.Lock()
		err := utils.SendAnnounceMsg(d.TrackerConn, msg.Announce)
		if err != nil {
			d.Logger.Fatal(err)
		}
		d.TrackerConnMutex.Unlock()

	case *protocol.NetworkMessage_PeerListRequest:
		// Transfer the peer list request to the tracker
		d.Logger.Debugf("Received PeerListRequest message from CLI: %+v", msg.PeerListRequest)
		d.PendingRequests[msg.PeerListRequest.GetFileHash()] = conn
		d.Logger.Debugf("Sending PeerListRequest message to Tracker: %+v", msg.PeerListRequest)

		d.TrackerConnMutex.Lock()
		utils.SendPeerListRequestMsg(d.TrackerConn, msg.PeerListRequest)
		d.TrackerConnMutex.Unlock()
	}
}

func (d *Daemon) listenTrackerMessages() {
	for {
		select {
		case <-d.Ctx.Done():
			d.Logger.Info("Stopping the tracker message listener")
			return
		default:
			netMsg, err := utils.ReceiveNetMsg(d.TrackerConn)
			if err != nil {
				d.Logger.Fatal(err)
			}
			switch msg := netMsg.MessageType.(type) {
			case *protocol.NetworkMessage_PeerListResponse:
				d.Logger.Debugf("Received peer list response from tracker: %+v", msg.PeerListResponse)
				cliConn, exists := d.PendingRequests[msg.PeerListResponse.GetFileHash()]
				if !exists {
					d.Logger.Warnf("No Requests for file hash: %s", msg.PeerListResponse.GetFileHash())
				}
				delete(d.PendingRequests, msg.PeerListResponse.GetFileHash())

				err := utils.SendNetMsg(cliConn, netMsg)
				if err != nil {
					d.Logger.Fatal(err)
				}
				d.Logger.Info("Sent peer list response to CLI")
			}
		}
	}
}

func (d *Daemon) listenPeerConn() {
	listen, err := net.Listen("tcp", d.LocalAddr)
	if err != nil {
		d.Logger.Fatalf("Error starting Peer TCP listener: %+v", err)
		return
	}
	defer listen.Close()

	d.Logger.Infof("Listening for peers on addr: %s", d.LocalAddr)

	for {
		conn, err := listen.Accept()
		if err != nil {
			d.Logger.Warnf("Error accepting connection: %+v", err)
			continue
		}
		// Listen for incoming msgs
		go d.handlePeerMsgs(conn)
	}
}

func (d *Daemon) handlePeerMsgs(peerConn net.Conn) {
	defer peerConn.Close()

	remoteAddr := peerConn.RemoteAddr().String()
	// clientIP, clientPort, err := net.SplitHostPort(remoteAddr)
	// if err != nil {
	// 	log.Fatal(err)
	// 	return
	// }

	d.Logger.Infof("New peer connected: %s\n", remoteAddr)
}

func (d *Daemon) initConnMsgs() {
	// Send a Register message to the tracker so that the tracker can save our public listneer port
	// Send the daemon port if running as development
	// Hit a STUN server and send the public port if running as production
	d.Logger.Info("Sending Register message to Tracker")
	d.PublicListenPort = strings.Split(d.LocalAddr, ":")[1]
	d.PublicIP = "localhost"
	if d.Mode == "prod" {
		// Hit a STUN server and get the public port
		d.Logger.Info("Trying to hit STUN servers to get public listen port")
	}
	registerMsg := &protocol.RegisterMessage{
		PublicIpAddress: d.PublicIP,
		ListenPort:      d.PublicListenPort,
	}
	utils.SendRegisterMsg(d.TrackerConn, registerMsg)
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
			FileHash:    file.Checksum,
			TotalChunks: int32(file.TotalChunks),
		})
	}
	announceMsg := &protocol.AnnounceMessage{
		Files: fileInfoMsgs,
	}

	d.TrackerConnMutex.Lock()
	err = utils.SendAnnounceMsg(d.TrackerConn, announceMsg)
	if err != nil {
		d.Logger.Fatal(err)
	}
	d.TrackerConnMutex.Unlock()

	d.Logger.Debugf("Sent initial Announce message to Tracker: %+v", announceMsg)
	// Send heartbeats every n seconds
	go d.sendHeartBeats()
}

func (d *Daemon) sendHeartBeats() {
	ticker := time.NewTicker(heartbeatInterval * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-d.Ctx.Done():
			d.Logger.Info("Stopping the heart")
			return
		case <-ticker.C:
			if d.TrackerConn == nil {
				return
			}

			d.Logger.Info("Sending a heartbeat")
			hb := &protocol.HeartbeatMessage{
				Timestamp: time.Now().Unix(),
			}

			netMsg := &protocol.NetworkMessage{
				MessageType: &protocol.NetworkMessage_Heartbeat{
					Heartbeat: hb,
				},
			}

			d.TrackerConnMutex.Lock()
			err := utils.SendNetMsg(d.TrackerConn, netMsg)
			if err != nil {
				d.Logger.Fatal("Error sending heartbeat:", err)
			}
			d.TrackerConnMutex.Unlock()
		}
	}
}
