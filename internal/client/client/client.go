package client

import (
	"net"

	"github.com/rudransh-shrivastava/peer-it/internal/client/db"
	"github.com/rudransh-shrivastava/peer-it/internal/shared/protocol"
	"github.com/rudransh-shrivastava/peer-it/internal/shared/store"
	"github.com/rudransh-shrivastava/peer-it/internal/shared/utils"
	"github.com/rudransh-shrivastava/peer-it/internal/shared/utils/logger"
	"github.com/sirupsen/logrus"
)

// This client communicates with the daemon using unix sockets
type Client struct {
	DaemonConn net.Conn

	FileStore  *store.FileStore
	ChunkStore *store.ChunkStore

	IPCSocketIndex string
	Logger         *logrus.Logger
}

func NewClient(index string) (*Client, error) {
	db, err := db.NewDB()
	logger := logger.NewLogger()
	if err != nil {
		logger.Fatal(err)
		return &Client{}, err
	}

	fileStore := store.NewFileStore(db)
	chunkStore := store.NewChunkStore(db)
	socketUrl := "/tmp/pit-daemon-" + index + ".sock"
	conn, err := net.Dial("unix", socketUrl)
	if err != nil {
		logger.Fatal(err)
		return &Client{}, err
	}

	return &Client{
		DaemonConn:     conn,
		FileStore:      fileStore,
		ChunkStore:     chunkStore,
		IPCSocketIndex: index,
		Logger:         logger,
	}, nil
}

func (c *Client) WaitForPeerList() *protocol.PeerListResponse {
	netMsg, err := utils.UnsafeReceiveNetMsg(c.DaemonConn)
	if err != nil {
		c.Logger.Fatal(err)
	}
	switch msg := netMsg.MessageType.(type) {
	case *protocol.NetworkMessage_PeerListResponse:
		c.Logger.Debugf("Received peer list response from daemon %+v", msg)
		peerListResponse := netMsg.GetPeerListResponse()
		return peerListResponse
	default:
		c.Logger.Warnf("Unexpected message type: %T", msg)
	}
	return nil
}

func (c *Client) SendRegisterSignal(filePath string) error {
	netMsg := &protocol.NetworkMessage{
		MessageType: &protocol.NetworkMessage_SignalRegister{
			SignalRegister: &protocol.SignalRegisterMessage{
				FilePath: filePath,
			},
		},
	}
	err := utils.SendNetMsg(c.DaemonConn, netMsg)
	if err != nil {
		c.Logger.Warnf("Error sending message to Daemon: %v", err)
		return err
	}
	return nil
}

func (c *Client) SendDownloadSignal(fileHash string) error {
	netMsg := &protocol.NetworkMessage{
		MessageType: &protocol.NetworkMessage_SignalDownload{
			SignalDownload: &protocol.SignalDownloadMessage{
				FileHash: fileHash,
			},
		},
	}
	err := utils.SendNetMsg(c.DaemonConn, netMsg)
	if err != nil {
		c.Logger.Warnf("Error sending message to Daemon: %v", err)
		return err
	}
	return nil
}
