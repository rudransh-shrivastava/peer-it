package cmd

import (
	"context"
	"encoding/binary"
	"log"
	"net"
	"os"
	"os/signal"
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

	log.Println("Daemon ready")

	<-sigChan
	log.Println("Shutting down daemon...")

	d.Conn.Close()

	log.Println("Daemon stopped")
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
	db, err := db.NewDB()
	if err != nil {
		log.Fatal(err)
		return
	}
	fileStore := store.NewFileStore(db)
	// Send an Announce message to the tracker the first time we connect
	files, err := fileStore.GetFiles()
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
