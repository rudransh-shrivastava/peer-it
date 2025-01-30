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

const remoteAddr = "localhost:8080"

type Daemon struct {
	Ctx context.Context

	TrackerConn      net.Conn
	TrackerConnMutex sync.Mutex

	FileStore *store.FileStore

	PendingRequests map[string]net.Conn
}

func newDaemon(ctx context.Context, conn net.Conn, fileStore *store.FileStore) *Daemon {
	return &Daemon{
		Ctx:             ctx,
		TrackerConn:     conn,
		FileStore:       fileStore,
		PendingRequests: make(map[string]net.Conn),
	}
}

var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "runs peer-it daemon",
	Long:  `runs the peer-it daemon in the background, the daemon uses unix sockets to communicate with the CLI`,
	Run: func(cmd *cobra.Command, args []string) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		db, err := db.NewDB()
		if err != nil {
			log.Fatal(err)
			return
		}
		fileStore := store.NewFileStore(db)

		conn, err := net.Dial("tcp", remoteAddr)
		if err != nil {
			log.Fatal(err)
			return
		}
		daemon := newDaemon(ctx, conn, fileStore)
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

	log.Println("Daemon ready")

	<-sigChan
	log.Println("Shutting down daemon...")

	d.TrackerConn.Close()

	log.Println("Daemon stopped")
}

func (d *Daemon) startIPCServer() {
	os.Remove("/tmp/pit-daemon.sock")
	l, err := net.Listen("unix", "/tmp/pit-daemon.sock")
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
		d.SendAnnounceMsg(msg.Announce)
	case *protocol.NetworkMessage_PeerListRequest:
		// Transfer the peer list request to the tracker
		log.Printf("Received PeerListRequest message from CLI: %+v", msg.PeerListRequest)
		d.PendingRequests[msg.PeerListRequest.GetFileHash()] = conn
		d.SendPeerListRequestMsg(msg.PeerListRequest)
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

func (d *Daemon) SendAnnounceMsg(msg *protocol.AnnounceMessage) {
	netMsg := &protocol.NetworkMessage{
		MessageType: &protocol.NetworkMessage_Announce{
			Announce: msg,
		},
	}
	d.TrackerConnMutex.Lock()
	err := utils.SendNetMsg(d.TrackerConn, netMsg)
	if err != nil {
		log.Fatal(err)
	}
	d.TrackerConnMutex.Unlock()
}

func (d *Daemon) SendPeerListRequestMsg(msg *protocol.PeerListRequest) {
	netMsg := &protocol.NetworkMessage{
		MessageType: &protocol.NetworkMessage_PeerListRequest{
			PeerListRequest: msg,
		},
	}
	d.TrackerConnMutex.Lock()
	err := utils.SendNetMsg(d.TrackerConn, netMsg)
	if err != nil {
		log.Fatal(err)
	}
	d.TrackerConnMutex.Unlock()
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
	netMsg := &protocol.NetworkMessage{
		MessageType: &protocol.NetworkMessage_Announce{
			Announce: announceMsg,
		},
	}

	d.TrackerConnMutex.Lock()
	err = utils.SendNetMsg(d.TrackerConn, netMsg)
	if err != nil {
		log.Fatal(err)
	}
	d.TrackerConnMutex.Unlock()

	log.Printf("Sent announce to tracker: %+v", announceMsg)
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
