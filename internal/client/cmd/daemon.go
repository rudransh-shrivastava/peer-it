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
		daemon := newDaemon(ctx, fileStore)
		daemon.startDaemon()
	},
}

type Daemon struct {
	Conn      *net.TCPConn
	ConnMutex sync.Mutex
	FileStore *store.FileStore
	Ctx       context.Context
}

func newDaemon(ctx context.Context, fileStore *store.FileStore) *Daemon {
	return &Daemon{
		FileStore: fileStore,
		Ctx:       ctx,
	}
}

func (d *Daemon) startDaemon() {
	// Set up signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)

	log.Println("Daemon starting...")

	go d.connectToTracker()
	go d.startIPCServer()

	log.Println("Daemon ready")

	<-sigChan
	log.Println("Shutting down daemon...")

	d.Conn.Close()

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
	defer conn.Close()
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
		log.Printf("Received Announce message in the daemon: %+v", msg.Announce)
		d.ConnMutex.Lock()
		defer d.ConnMutex.Unlock()
		d.AnnounceFile(msg.Announce)
	}
}

func (d *Daemon) AnnounceFile(msg *protocol.AnnounceMessage) {
	netMsg := &protocol.NetworkMessage{
		MessageType: &protocol.NetworkMessage_Announce{
			Announce: msg,
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
	if err := binary.Write(d.Conn, binary.BigEndian, msgLen); err != nil {
		log.Printf("Error sending message length: %v", err)
		return err
	}

	if _, err := d.Conn.Write(data); err != nil {
		log.Printf("Error sending message: %v", err)
		return err
	}
	return nil
}

func (d *Daemon) connectToTracker() {
	for {
		select {
		case <-d.Ctx.Done():
			log.Println("Disconnecting from tracker")
			return
		default:
			remoteAddr := "localhost:8080"
			raddr, err := net.ResolveTCPAddr("tcp", remoteAddr)
			if err != nil {
				log.Println("Error resolving remote address:", err)
				os.Exit(0)
			}
			laddr := &net.TCPAddr{
				IP:   net.ParseIP("0.0.0.0"),
				Port: 0,
			}

			conn, err := net.DialTCP("tcp", laddr, raddr)
			if err != nil {
				log.Println("Error dialing:", err)
				log.Println("Reconnecting in 10 seconds...")
				select {
				case <-time.After(10 * time.Second):
					continue
				case <-d.Ctx.Done():
					log.Println("Disconnecting from tracker")
					return
				}
			}
			d.Conn = conn

			log.Println("Connected to", conn.RemoteAddr())
			d.handleConn()
			return
		}
	}
}

func (d *Daemon) handleConn() {
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
	msgLen := uint32(len(data))
	if err := binary.Write(d.Conn, binary.BigEndian, msgLen); err != nil {
		log.Printf("Error sending message length: %v", err)
		return
	}

	if _, err := d.Conn.Write(data); err != nil {
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
			log.Println("Sending a heartbeat")
			hb := &protocol.HeartbeatMessage{
				Timestamp: time.Now().Unix(),
			}

			netMsg := &protocol.NetworkMessage{
				MessageType: &protocol.NetworkMessage_Heartbeat{
					Heartbeat: hb,
				},
			}

			data, err := proto.Marshal(netMsg)
			if err != nil {
				log.Println("Error marshalling message:", err)
				return
			}
			msgLen := uint32(len(data))
			if err := binary.Write(d.Conn, binary.BigEndian, msgLen); err != nil {
				log.Printf("Error sending message length: %v", err)
				return
			}

			if _, err := d.Conn.Write(data); err != nil {
				log.Printf("Error sending message: %v", err)
				return
			}
		}
	}
}
