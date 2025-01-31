package cmd

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

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

	PendingRequests map[string]net.Conn
	IPCSocketIndex  string
	Addr            string
}

func newDaemon(ctx context.Context, conn net.Conn, fileStore *store.FileStore, ipcSocketIndex string, addr string) *Daemon {
	return &Daemon{
		Ctx:             ctx,
		TrackerConn:     conn,
		FileStore:       fileStore,
		PendingRequests: make(map[string]net.Conn),
		IPCSocketIndex:  ipcSocketIndex,
		Addr:            addr,
	}
}

var daemonCmd = &cobra.Command{
	Use:   "daemon port ipc-socket-index remote-address",
	Short: "runs peer-it daemon",
	Long:  `runs the peer-it daemon in the background, the daemon uses unix sockets to communicate with the CLI`,
	Args:  cobra.ExactArgs(3),
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
		daemon := newDaemon(ctx, conn, fileStore, ipcSocketIndex, daemonAddr)
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

	d.initConnMsgs()

	log.Printf("Daemon ready, running on %s", d.Addr)

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

func (d *Daemon) initConnMsgs() {
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
