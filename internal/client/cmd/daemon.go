package cmd

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/pion/stun"
	"github.com/rudransh-shrivastava/peer-it/internal/client/db"
	"github.com/rudransh-shrivastava/peer-it/internal/shared/protocol"
	"github.com/rudransh-shrivastava/peer-it/internal/shared/store"
	"github.com/rudransh-shrivastava/peer-it/internal/shared/utils"
	"github.com/spf13/cobra"
)

type Daemon struct {
	Ctx context.Context

	TrackerConn      net.Conn
	TrackerConnMutex sync.Mutex

	FileStore *store.FileStore

	PendingRequests  map[string]net.Conn
	IPCSocketIndex   string
	LocalAddr        string
	PublicListenPort string
	PublicIP         string
	Mode             string // can be dev or prod
}

func newDaemon(ctx context.Context, conn net.Conn, fileStore *store.FileStore, ipcSocketIndex string, localAddr string, mode string) *Daemon {
	return &Daemon{
		Ctx:             ctx,
		TrackerConn:     conn,
		FileStore:       fileStore,
		PendingRequests: make(map[string]net.Conn),
		IPCSocketIndex:  ipcSocketIndex,
		LocalAddr:       localAddr,
		Mode:            mode, // Can be dev or prod
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
		log.Printf("Daemon port: %s", daemonPort)
		ipcSocketIndex := args[1]
		log.Printf("IPC Socket Index: %s", ipcSocketIndex)
		trackerAddr := args[2]
		log.Printf("Tracker Address: %s", trackerAddr)
		daemonMode := args[3]
		if daemonMode == "prod" {
			// Run in production mode
			log.Println("Running Daemon in Production Mode")
		} else {
			// Run in development mode
			daemonMode = "dev"
			log.Println("Running Daemon in Development Mode")
		}
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		db, err := db.NewDB()
		if err != nil {
			log.Fatal(err)
			return
		}

		conn, err := net.Dial("tcp", trackerAddr)
		if err != nil {
			log.Fatal(err)
			return
		}
		fileStore := store.NewFileStore(db)
		daemon := newDaemon(ctx, conn, fileStore, ipcSocketIndex, daemonAddr, daemonMode)
		daemon.startDaemon()
	},
}

func (d *Daemon) startDaemon() {
	// Set up signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)

	log.Println("Daemon starting...")

	go d.startIPCServer()
	go d.listenTrackerMessages()
	go d.listenPeerConn()

	d.initConnMsgs()

	log.Printf("Daemon ready, running on %s", d.LocalAddr)

	<-sigChan
	log.Println("Shutting down daemon...")

	d.TrackerConn.Close()

	log.Println("Daemon stopped")
}

func (d *Daemon) startIPCServer() {
	socketUrl := "/tmp/pit-daemon-" + d.IPCSocketIndex + ".sock"
	os.Remove(socketUrl)

	l, err := net.Listen("unix", socketUrl)
	if err != nil {
		panic(err)
	}
	log.Println("IPC Server started successfuly")
	for {
		cliConn, err := l.Accept()
		log.Println("Accepted a new socket connection")
		if err != nil {
			continue
		}
		go d.handleCLIRequest(cliConn)
	}
}

func (d *Daemon) handleCLIRequest(conn net.Conn) {
	netMsg := utils.ReceiveNetMsg(conn)
	switch msg := netMsg.MessageType.(type) {
	case *protocol.NetworkMessage_Announce:
		// Transfer the announce message to the tracker
		log.Printf("Received Announce message from CLI: %+v", msg.Announce)
		log.Printf("Sending Announce message to Tracker: %+v ", msg.Announce)

		d.TrackerConnMutex.Lock()
		err := utils.SendAnnounceMsg(d.TrackerConn, msg.Announce)
		if err != nil {
			log.Fatal(err)
		}
		d.TrackerConnMutex.Unlock()

	case *protocol.NetworkMessage_PeerListRequest:
		// Transfer the peer list request to the tracker
		log.Printf("Received PeerListRequest message from CLI: %+v", msg.PeerListRequest)
		d.PendingRequests[msg.PeerListRequest.GetFileHash()] = conn
		log.Printf("Sending PeerListRequest message to Tracker: %+v", msg.PeerListRequest)

		d.TrackerConnMutex.Lock()
		utils.SendPeerListRequestMsg(d.TrackerConn, msg.PeerListRequest)
		d.TrackerConnMutex.Unlock()
	}
}

func (d *Daemon) listenTrackerMessages() {
	for {
		select {
		case <-d.Ctx.Done():
			log.Println("Stopping the tracker message listener")
			return
		default:
			netMsg := utils.ReceiveNetMsg(d.TrackerConn)
			switch msg := netMsg.MessageType.(type) {
			case *protocol.NetworkMessage_PeerListResponse:
				log.Printf("Received peer list response from tracker: %+v", msg.PeerListResponse)
				cliConn, exists := d.PendingRequests[msg.PeerListResponse.GetFileHash()]
				if !exists {
					log.Printf("No Requests for file hash: %s", msg.PeerListResponse.GetFileHash())
				}
				delete(d.PendingRequests, msg.PeerListResponse.GetFileHash())

				err := utils.SendNetMsg(cliConn, netMsg)
				if err != nil {
					log.Fatal(err)
				}
				log.Printf("Sent peer list response to CLI")
			}
		}
	}
}

func (d *Daemon) listenPeerConn() {
	listen, err := net.Listen("tcp", d.LocalAddr)
	if err != nil {
		log.Println("Error starting Peer TCP listener:", err)
		return
	}
	defer listen.Close()

	log.Printf("Listening for peers on addr: %s", d.LocalAddr)

	for {
		conn, err := listen.Accept()
		if err != nil {
			log.Println("Error accepting connection:", err)
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

	log.Printf("New peer connected: %s\n", remoteAddr)

}

type STUNClient struct {
	stunServers []string
	timeout     time.Duration
}

func NewSTUNClient() *STUNClient {
	return &STUNClient{
		// List of STUN servers to try
		stunServers: []string{
			"stun.l.google.com:19302",
			"stun1.l.google.com:19302",
			"stun2.l.google.com:19302",
		},
		timeout: time.Second * 5,
	}
}

func (sc *STUNClient) DiscoverEndpoint() (string, error) {
	// Parse a STUN URI
	u, err := stun.ParseURI("stun:stun.l.google.com:19302")
	if err != nil {
		return "", err
	}

	// Creating a "connection" to STUN server.
	c, err := stun.DialURI(u, &stun.DialConfig{})
	if err != nil {
		return "", err
	}
	// Building binding request with random transaction id.
	message := stun.MustBuild(stun.TransactionID, stun.BindingRequest)
	// Sending request to STUN server, waiting for response message.
	var ipAddr string
	if err := c.Do(message, func(res stun.Event) {
		if res.Error != nil {
			log.Println("Error: ", res.Error)
			return
		}
		// Decoding XOR-MAPPED-ADDRESS attribute from message.
		var xorAddr stun.XORMappedAddress
		if err := xorAddr.GetFrom(res.Message); err != nil {
			log.Println("Error: ", err)
			return
		}
		fmt.Println("your public IP is", xorAddr.IP)
		fmt.Println("your public port is", xorAddr.Port)
		ipAddr = xorAddr.IP.String()
		// port = xorAddr.Port

	}); err != nil {
		log.Fatal(err)
	}
	return ipAddr, nil
}

func (d *Daemon) initConnMsgs() {
	// Send a Register message to the tracker so that the tracker can save our public listneer port
	// Send the daemon port if running as development
	// Hit a STUN server and send the public port if running as production
	log.Printf("Sending Register message to Tracker")
	d.PublicListenPort = strings.Split(d.LocalAddr, ":")[1]
	if d.Mode == "prod" {
		// Hit a STUN server and get the public port
		log.Printf("Trying to hit STUN servers to get public listen port")
		stunClient := NewSTUNClient()
		publicIP, err := stunClient.DiscoverEndpoint()
		if err != nil {
			log.Fatal(err)
		}
		d.PublicIP = publicIP
		log.Printf("Found Public IP %s", publicIP)
	}
	registerMsg := &protocol.RegisterMessage{
		ListenPort: d.PublicListenPort,
	}
	utils.SendRegisterMsg(d.TrackerConn, registerMsg)
	log.Printf("Sent Register message to Tracker: %+v", registerMsg)

	// Send an Announce message to the tracker the first time we connect
	files, err := d.FileStore.GetFiles()
	if err != nil {
		log.Println("Error getting files:", err)
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
		log.Fatal(err)
	}
	d.TrackerConnMutex.Unlock()

	log.Printf("Sent initial Announce message to Tracker: %+v", announceMsg)
	// Send heartbeats every n seconds
	go d.sendHeartBeats()
}

func (d *Daemon) sendHeartBeats() {
	ticker := time.NewTicker(heartbeatInterval * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-d.Ctx.Done():
			log.Println("Stopping the heart")
			return
		case <-ticker.C:
			if d.TrackerConn == nil {
				return
			}

			log.Println("Sending a heartbeat")
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
				log.Fatal("Error sending heartbeat:", err)
			}
			d.TrackerConnMutex.Unlock()
		}
	}
}
