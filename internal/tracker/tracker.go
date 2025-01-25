package tracker

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"

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
		fmt.Println("Error starting TCP server:", err)
		return
	}
	defer listen.Close()

	fmt.Println("Server listening on port 8080")

	for {
		conn, err := listen.Accept()
		if err != nil {
			fmt.Println("Error accepting connection:", err)
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

	fmt.Printf("New client connected: %s\n", remoteAddr)
	t.ClientStore.CreateClient(clientIP, clientPort)

	for {
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
		case *protocol.NetworkMessage_Heartbeat:
			// Update last heartbeat time
			log.Printf("Received heartbeat from %s: %v", clientIP, msg.Heartbeat)

			// Check client timeout in background
			// go t.CheckClientTimeout(clientIP)

		default:
			log.Printf("Received unsupported message type from %s", clientIP)
		}
	}
}
