package tracker

import (
	"net"
	"time"

	"github.com/rudransh-shrivastava/peer-it/internal/shared/protocol"
	"github.com/rudransh-shrivastava/peer-it/internal/shared/schema"
	"github.com/rudransh-shrivastava/peer-it/internal/shared/store"
	"github.com/rudransh-shrivastava/peer-it/internal/shared/utils"
	"github.com/sirupsen/logrus"
)

const (
	ClientTimeout = 10
)

type Tracker struct {
	PeerStore  *store.PeerStore
	FileStore  *store.FileStore
	ChunkStore *store.ChunkStore

	Logger *logrus.Logger
}

func NewTracker(peerStore *store.PeerStore, fileStore *store.FileStore, chunkStore *store.ChunkStore, logger *logrus.Logger) *Tracker {
	return &Tracker{
		PeerStore:  peerStore,
		FileStore:  fileStore,
		ChunkStore: chunkStore,
		Logger:     logger,
	}
}

func (t *Tracker) Start() {
	listen, err := net.Listen("tcp", ":42069")
	if err != nil {
		t.Logger.Fatalf("Error starting TCP server: %+v", err)
		return
	}
	defer listen.Close()

	t.Logger.Infof("Server listening on port 42069")

	for {
		conn, err := listen.Accept()
		if err != nil {
			t.Logger.Warnf("Error accepting connection: %+v", err)
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
		t.Logger.Fatal(err)
	}
}

func (t *Tracker) ListenCLIConnMsgs(cliConn net.Conn) {
	defer cliConn.Close()

	remoteAddr := cliConn.RemoteAddr().String()
	clientIP, clientPort, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		t.Logger.Fatal(err)
		return
	}

	t.Logger.Infof("New client connected: %s\n", remoteAddr)
	err = t.PeerStore.CreatePeer(clientIP, clientPort)
	if err != nil {
		t.Logger.Fatalf("Error creating client: %v", err)
		return
	}

	// Start a timer to track client timeout
	timeout := time.NewTicker(ClientTimeout * time.Second)
	defer timeout.Stop()

	for {
		select {
		// If the client times out, delete the client from the db
		case <-timeout.C:
			t.Logger.Infof("Client %s timed out, deleting \n", remoteAddr)
			err := t.PeerStore.DeletePeer(clientIP, clientPort)
			if err != nil {
				t.Logger.Fatalf("Error deleting client: %v", err)
			}
			return
		default:
			netMsg, err := utils.ReceiveNetMsg(cliConn)
			if err != nil {
				t.Logger.Fatal(err)
			}

			switch msg := netMsg.MessageType.(type) {
			// Reset the timer if the message is a heartbeat
			case *protocol.NetworkMessage_Register:
				// Save the public listener port of the client in DB
				t.Logger.Debugf("Received register message from %s: %v", remoteAddr, msg.Register)
				err := t.PeerStore.RegisterPeerPublicListenPort(clientIP, clientPort, msg.Register.GetListenPort(), msg.Register.GetPublicIpAddress())
				if err != nil {
					t.Logger.Fatalf("Error registering client: %v", err)
				}
				t.Logger.Debugf("Client %s registered with public listener port %s", remoteAddr, msg.Register.GetListenPort())
			case *protocol.NetworkMessage_Heartbeat:
				t.Logger.Debugf("Received heartbeat from %s: %v", remoteAddr, msg.Heartbeat)
				timeout.Reset(ClientTimeout * time.Second)
			case *protocol.NetworkMessage_Goodbye:
				// Remove the client from db
				t.PeerStore.DeletePeer(clientIP, clientPort)
				t.Logger.Infof("Client %s disconnected", remoteAddr)
				return
			case *protocol.NetworkMessage_Announce:
				t.Logger.Debugf("Received Announce message from %s: %v", remoteAddr, msg.Announce)
				files := msg.Announce.GetFiles()
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
						t.Logger.Fatalf("Error creating file: %v", err)
					}
					if !created {
						// File already existed in db
						t.Logger.Debugf("File %+v already exists, adding client to swarm", schemaFile)
					} else {
						t.Logger.Debugf("File: %s, Size: %d, Chunks: %d Max Chunk Size: %d", file.GetFileHash(), file.GetFileSize(), file.GetTotalChunks(), file.GetChunkSize())
					}
					// Add client to swarm of peers
					err = t.PeerStore.AddPeerToSwarm(clientIP, clientPort, file.GetFileHash())
				}
			case *protocol.NetworkMessage_PeerListRequest:
				t.Logger.Debugf("Received peer list request from %s: %v", remoteAddr, msg.PeerListRequest)
				fileHash := msg.PeerListRequest.GetFileHash()

				dbPeers, err := t.PeerStore.GetPeersByFileHash(fileHash)
				if err != nil {
					t.Logger.Fatalf("Error getting peers: %v", err)
				}

				peers := make([]*protocol.PeerInfo, 0)
				for _, peer := range dbPeers {
					if peer.IPAddress == clientIP && peer.Port == clientPort {
						t.Logger.Debugf("Skipping %s : %s", clientIP, clientPort)
						t.Logger.Debugf("Skipped: %+v", peer)
						continue
					}

					publicIPAddr, publicListenPort, err := t.PeerStore.FindPublicListenPort(peer.IPAddress, peer.Port)
					if err != nil {
						t.Logger.Fatalf("Error finding public listen port: %v", err)
					}
					t.Logger.Debugf("Got public IP:PORT for %+v from db %s:%s", peer, publicIPAddr, publicListenPort)
					peers = append(peers, &protocol.PeerInfo{
						IpAddress: publicIPAddr,
						Port:      publicListenPort,
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
				t.Logger.Debugf("Sending back the peers list to %s : %+v", remoteAddr, peerListResponse)
				err = utils.SendNetMsg(cliConn, netMsg)
				if err != nil {
					t.Logger.Fatalf("Error sending peer list response: %v", err)
				}

			default:
				t.Logger.Debugf("Received unsupported message type %+v from %s", netMsg.MessageType, remoteAddr)
			}
		}
	}
}
