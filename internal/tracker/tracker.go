package tracker

import (
	"encoding/binary"
	"io"
	"log"
	"net"
	"time"

	"github.com/rudransh-shrivastava/peer-it/internal/shared/protocol"
	"github.com/rudransh-shrivastava/peer-it/internal/shared/schema"
	"github.com/rudransh-shrivastava/peer-it/internal/shared/store"
	"google.golang.org/protobuf/proto"
)

const (
	ClientTimeout = 10
)

type Tracker struct {
	PeerStore  *store.PeerStore
	FileStore  *store.FileStore
	ChunkStore *store.ChunkStore
}

func NewTracker(peerStore *store.PeerStore, fileStore *store.FileStore, chunkStore *store.ChunkStore) *Tracker {
	return &Tracker{
		PeerStore:  peerStore,
		FileStore:  fileStore,
		ChunkStore: chunkStore,
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
	err = t.PeerStore.CreatePeer(clientIP, clientPort)
	if err != nil {
		log.Printf("Error creating client: %v", err)
		return
	}

	// Start a timer to track client timeout
	timeout := time.NewTicker(ClientTimeout * time.Second)
	defer timeout.Stop()

	for {
		select {
		// If the client times out, delete the client from the db
		case <-timeout.C:
			log.Printf("Client %s timed out, deleting \n", remoteAddr)
			err := t.PeerStore.DeletePeer(clientIP, clientPort)
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
				log.Printf("Received heartbeat from %s: %v", remoteAddr, msg.Heartbeat)
				timeout.Reset(ClientTimeout * time.Second)
			case *protocol.NetworkMessage_Announce:
				log.Printf("Received announce from %s: %v", remoteAddr, msg.Announce)
				files := msg.Announce.GetFiles()
				log.Printf("Announce Files: %+v", files)
				for _, file := range files {
					totalChunks := int(file.GetTotalChunks())
					maxChunkSize := int(file.GetChunkSize())
					lastChunkSize := int(file.GetFileSize() % int64(maxChunkSize))
					schemaFile := &schema.File{
						Size:         file.GetFileSize(),
						MaxChunkSize: maxChunkSize,
						TotalChunks:  totalChunks,
						Checksum:     file.GetFileHash(),
						CreatedAt:    time.Now().Unix(),
					}
					created, err := t.FileStore.CreateFile(schemaFile)
					if err != nil {
						log.Printf("Error creating file: %v", err)
					}
					if !created {
						// File already existed in db
						log.Printf("File %+v already exists, adding client to swarm", schemaFile)
					} else {
						log.Printf("Attemping to create chunks")
						for i := 0; i < totalChunks; i++ {
							if i == totalChunks-1 {
								// Create the last chunk without metadata
								t.ChunkStore.CreateChunk(schemaFile, lastChunkSize, i, "any checksum", false)
							}
							// Create a full chunk without metadata
							t.ChunkStore.CreateChunk(schemaFile, maxChunkSize, i, "any checksum", false)
						}
						log.Printf("File: %s, Size: %d, Chunks: %d Max Chunk Size: %d", file.GetFileHash(), file.GetFileSize(), file.GetTotalChunks(), file.GetChunkSize())
					}
					// TODO: add client to swarm of peers
					// err = t.PeerStore.AddPeerToSwarm(clientIP, clientPort, file.GetFileHash())
				}
			default:
				log.Printf("Received unsupported message type from %s", remoteAddr)
			}
		}
	}
}
