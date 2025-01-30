package tracker

import (
	"log"
	"net"
	"strconv"
	"time"

	"github.com/rudransh-shrivastava/peer-it/internal/shared/protocol"
	"github.com/rudransh-shrivastava/peer-it/internal/shared/schema"
	"github.com/rudransh-shrivastava/peer-it/internal/shared/store"
	"github.com/rudransh-shrivastava/peer-it/internal/shared/utils"
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
		// Listen for incoming msgs
		go t.ListenCLIConnMsgs(conn)
	}
}

func (t *Tracker) Stop() {
	// cleanup peers
	err := t.PeerStore.DropAllPeers()
	if err != nil {
		log.Println(err)
	}
}

func (t *Tracker) ListenCLIConnMsgs(cliConn net.Conn) {
	defer cliConn.Close()

	remoteAddr := cliConn.RemoteAddr().String()
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
			netMsg := utils.ReceiveNetMsg(cliConn)

			switch msg := netMsg.MessageType.(type) {
			// Reset the timer if the message is a heartbeat
			case *protocol.NetworkMessage_Heartbeat:
				log.Printf("Received heartbeat from %s: %v", remoteAddr, msg.Heartbeat)
				timeout.Reset(ClientTimeout * time.Second)
			case *protocol.NetworkMessage_Goodbye:
				// Remove the client from db
				t.PeerStore.DeletePeer(clientIP, clientPort)
				log.Printf("Client %s disconnected", remoteAddr)
				return
			case *protocol.NetworkMessage_Announce:
				log.Printf("Received announce from %s: %v", remoteAddr, msg.Announce)
				files := msg.Announce.GetFiles()
				log.Printf("Announce Files: %+v", files)
				for _, file := range files {
					totalChunks := int(file.GetTotalChunks())
					maxChunkSize := int(file.GetChunkSize())
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
						log.Printf("File: %s, Size: %d, Chunks: %d Max Chunk Size: %d", file.GetFileHash(), file.GetFileSize(), file.GetTotalChunks(), file.GetChunkSize())
					}
					// Add client to swarm of peers
					err = t.PeerStore.AddPeerToSwarm(clientIP, clientPort, file.GetFileHash())
				}
			case *protocol.NetworkMessage_PeerListRequest:
				log.Printf("Received peer list request from %s: %v", remoteAddr, msg.PeerListRequest)
				fileHash := msg.PeerListRequest.GetFileHash()

				dbPeers, err := t.PeerStore.GetPeersByFileHash(fileHash)
				if err != nil {
					log.Printf("Error getting peers: %v", err)
				}

				peers := make([]*protocol.PeerInfo, 0)
				for _, peer := range dbPeers {
					port, err := strconv.Atoi(peer.Port)
					int32port := int32(port)
					if err != nil {
						log.Printf("Error converting port to int: %v", err)
						continue
					}
					peers = append(peers, &protocol.PeerInfo{
						IpAddress: peer.IPAddress,
						Port:      int32port,
					})
				}
				peerListResponse := &protocol.PeerListResponse{
					FileHash:    fileHash,
					TotalChunks: 0,
					ChunkSize:   0,
					Peers:       peers,
				}
				netMsg := &protocol.NetworkMessage{
					MessageType: &protocol.NetworkMessage_PeerListResponse{
						PeerListResponse: peerListResponse,
					},
				}
				log.Printf("Sending back the peers list to %s : %+v", remoteAddr, peerListResponse)
				err = utils.SendNetMsg(cliConn, netMsg)
				if err != nil {
					log.Printf("Error sending peer list response: %v", err)
				}

			default:
				log.Printf("Received unsupported message type %+v from %s", netMsg.MessageType, remoteAddr)
			}
		}
	}
}
