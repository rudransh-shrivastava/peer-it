package cmd

import (
	"context"
	"encoding/binary"
	"io"
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
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/proto"
)

const remoteAddr = "localhost:8080"

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

type Daemon struct {
	Ctx              context.Context
	TrackerConn      net.Conn
	TrackerConnMutex sync.Mutex
	FileStore        *store.FileStore
	PendingRequests  map[string]net.Conn
}

func newDaemon(ctx context.Context, conn net.Conn, fileStore *store.FileStore) *Daemon {
	return &Daemon{
		Ctx:             ctx,
		TrackerConn:     conn,
		FileStore:       fileStore,
		PendingRequests: make(map[string]net.Conn),
	}
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
		conn, err := l.Accept()
		log.Println("Accepted a new socket connection")
		if err != nil {
			continue
		}
		go d.handleCLIRequest(conn)
	}
}

func (d *Daemon) handleCLIRequest(conn net.Conn) {
	var msgLen uint32
	if err := binary.Read(conn, binary.BigEndian, &msgLen); err != nil {
		if err != io.EOF {
			log.Printf("Error reading message length: %v", err)
		}
	}

	data := make([]byte, msgLen)
	if _, err := io.ReadFull(conn, data); err != nil {
		log.Printf("Error reading message body: %v", err)
	}

	var netMsg protocol.NetworkMessage
	if err := proto.Unmarshal(data, &netMsg); err != nil {
		log.Printf("Error unmarshaling message: %v", err)
	}

	// see tracker to understand how its done
	// if i forget how to do it

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
			var msgLen uint32
			if err := binary.Read(d.TrackerConn, binary.BigEndian, &msgLen); err != nil {
				if err != io.EOF {
					log.Printf("Error reading message length: %v", err)
				}
				break
			}
			data := make([]byte, msgLen)
			if _, err := io.ReadFull(d.TrackerConn, data); err != nil {
				log.Printf("Error reading message body: %v", err)
				break
			}

			var netMsg protocol.NetworkMessage
			if err := proto.Unmarshal(data, &netMsg); err != nil {
				log.Printf("Error unmarshaling message: %v", err)
				continue
			}

			switch msg := netMsg.MessageType.(type) {
			case *protocol.NetworkMessage_PeerListResponse:
				log.Printf("Received peer list response from tracker: %+v", msg.PeerListResponse)
				conn, exists := d.PendingRequests[msg.PeerListResponse.GetFileHash()]
				if !exists {
					log.Printf("No Requests for file hash: %s", msg.PeerListResponse.GetFileHash())
				}
				delete(d.PendingRequests, msg.PeerListResponse.GetFileHash())

				data, err := proto.Marshal(&netMsg)
				if err != nil {
					log.Println("Error marshalling message:", err)
				}
				msgLen := uint32(len(data))
				if err := binary.Write(conn, binary.BigEndian, msgLen); err != nil {
					log.Printf("Error sending message length: %v", err)
				}

				if _, err := conn.Write(data); err != nil {
					log.Printf("Error sending message: %v", err)
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
	d.sendMessage(netMsg)
}

func (d *Daemon) SendPeerListRequestMsg(msg *protocol.PeerListRequest) {
	netMsg := &protocol.NetworkMessage{
		MessageType: &protocol.NetworkMessage_PeerListRequest{
			PeerListRequest: msg,
		},
	}
	d.sendMessage(netMsg)
}

func (d *Daemon) sendMessage(msg *protocol.NetworkMessage) error {
	data, err := proto.Marshal(msg)
	if err != nil {
		log.Println("Error marshalling message:", err)
		return err
	}
	msgLen := uint32(len(data))
	if err := binary.Write(d.TrackerConn, binary.BigEndian, msgLen); err != nil {
		log.Printf("Error sending message length: %v", err)
		return err
	}

	d.TrackerConnMutex.Lock()
	defer d.TrackerConnMutex.Unlock()
	if _, err := d.TrackerConn.Write(data); err != nil {
		log.Printf("Error sending message: %v", err)
		return err
	}
	return nil
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
	log.Printf("Preparing to send announce to tracker with files: %+v", fileInfoMsgs)
	announceMsg := &protocol.AnnounceMessage{
		Files: fileInfoMsgs,
	}
	netMsg := &protocol.NetworkMessage{
		MessageType: &protocol.NetworkMessage_Announce{
			Announce: announceMsg,
		},
	}
	data, err := proto.Marshal(netMsg)
	if err != nil {
		log.Println("Error marshalling message:", err)
		return
	}
	d.TrackerConnMutex.Lock()
	defer d.TrackerConnMutex.Unlock()
	msgLen := uint32(len(data))
	if err := binary.Write(d.TrackerConn, binary.BigEndian, msgLen); err != nil {
		log.Printf("Error sending message length: %v", err)
		return
	}

	if _, err := d.TrackerConn.Write(data); err != nil {
		log.Printf("Error sending message: %v", err)
		return
	}
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
			err := d.sendMessage(netMsg)
			if err != nil {
				log.Println("Error sending heartbeat:", err)
				return
			}
		}
	}
}
