package tracker

import (
	"encoding/binary"
	"io"
	"log"
	"net"
	"time"

	"github.com/rudransh-shrivastava/peer-it/internal/shared/protocol"
	"github.com/rudransh-shrivastava/peer-it/internal/shared/store"
	"google.golang.org/protobuf/proto"
)

const (
	ClientTimeout = 10
)

type Tracker struct {
	ClientStore *store.ClientStore
}

func NewTracker(clientStore *store.ClientStore) *Tracker {
	return &Tracker{
		ClientStore: clientStore,
	}
}

func (t *Tracker) Start() {
	listen, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Println("Error starting TCP server:", err)
		return
	}
	defer listen.Close()

	log.Println("Server listening on port 8080")

	for {
		conn, err := listen.Accept()
		if err != nil {
			log.Println("Error accepting connection:", err)
			continue
		}
		// Handle connection
		go t.HandleConn(conn)
	}
}

func (t *Tracker) HandleConn(conn net.Conn) {
	defer conn.Close()

	remoteAddr := conn.RemoteAddr().String()
	clientIP, clientPort, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		log.Fatal(err)
		return
	}

	log.Printf("New client connected: %s\n", remoteAddr)
	err = t.ClientStore.CreateClient(clientIP, clientPort)
	if err != nil {
		log.Printf("Error creating client: %v", err)
		return
	}

	timeout := time.NewTicker(ClientTimeout * time.Second)
	defer timeout.Stop()

	for {
		select {
		// If the client times out, delete the client from the db
		case <-timeout.C:
			log.Printf("Client %s timed out\n", remoteAddr)
			err := t.ClientStore.DeleteClient(clientIP, clientPort)
			if err != nil {
				log.Printf("Error deleting client: %v", err)
			}
			return
		default:
			var msgLen uint32
			if err := binary.Read(conn, binary.BigEndian, &msgLen); err != nil {
				if err != io.EOF {
					log.Printf("Error reading message length: %v", err)
				}
				break
			}

			data := make([]byte, msgLen)
			if _, err := io.ReadFull(conn, data); err != nil {
				log.Printf("Error reading message body: %v", err)
				break
			}

			var netMsg protocol.NetworkMessage
			if err := proto.Unmarshal(data, &netMsg); err != nil {
				log.Printf("Error unmarshaling message: %v", err)
				continue
			}

			switch msg := netMsg.MessageType.(type) {
			// Reset the timer if the message is a heartbeat
			case *protocol.NetworkMessage_Heartbeat:
				log.Printf("Received heartbeat from %s: %v", clientIP, msg.Heartbeat)
				timeout.Reset(ClientTimeout * time.Second)
			default:
				log.Printf("Received unsupported message type from %s", clientIP)
			}
		}
	}
}
