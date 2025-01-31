package client

import (
	"log"
	"net"

	"github.com/rudransh-shrivastava/peer-it/internal/client/db"
	"github.com/rudransh-shrivastava/peer-it/internal/shared/protocol"
	"github.com/rudransh-shrivastava/peer-it/internal/shared/store"
	"github.com/rudransh-shrivastava/peer-it/internal/shared/utils"
)

// This client communicates with the daemon using unix sockets
type Client struct {
	DaemonConn net.Conn

	FileStore  *store.FileStore
	ChunkStore *store.ChunkStore

	IPCSocketIndex string
}

func NewClient(index string) (*Client, error) {
	db, err := db.NewDB()
	if err != nil {
		log.Fatal(err)
		return &Client{}, err
	}

	fileStore := store.NewFileStore(db)
	chunkStore := store.NewChunkStore(db)
	socketUrl := "/tmp/pit-daemon-" + index + ".sock"
	conn, err := net.Dial("unix", socketUrl)
	if err != nil {
		log.Fatal(err)
		return &Client{}, err
	}

	return &Client{
		DaemonConn:     conn,
		FileStore:      fileStore,
		ChunkStore:     chunkStore,
		IPCSocketIndex: index,
	}, nil
}

func (c *Client) WaitForPeerList() *protocol.PeerListResponse {
	netMsg := utils.ReceiveNetMsg(c.DaemonConn)
	switch msg := netMsg.MessageType.(type) {
	case *protocol.NetworkMessage_PeerListResponse:
		log.Printf("Received peer list response from daemon %+v", msg)
		peerListResponse := netMsg.GetPeerListResponse()
		return peerListResponse
	default:
		log.Printf("Unexpected message type: %T", msg)
	}
	return nil
}
