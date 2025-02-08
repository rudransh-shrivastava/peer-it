package client

import (
	"net"
	"strconv"

	"github.com/rudransh-shrivastava/peer-it/internal/client/db"
	"github.com/rudransh-shrivastava/peer-it/internal/shared/protocol"
	"github.com/rudransh-shrivastava/peer-it/internal/shared/prouter"
	"github.com/rudransh-shrivastava/peer-it/internal/shared/store"
	"github.com/rudransh-shrivastava/peer-it/internal/shared/utils"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/proto"
)

// This client communicates with the daemon using unix sockets
type Client struct {
	DaemonConn net.Conn

	FileStore  *store.FileStore
	ChunkStore *store.ChunkStore

	IPCSocketIndex  string
	Logger          *logrus.Logger
	LogChan         chan *protocol.NetworkMessage_Log
	ProgressBarChan chan int
	TotalChunksChan chan int
}

func NewClient(index string, logger *logrus.Logger) (*Client, error) {
	db, err := db.NewDB(index)
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

	logChan := make(chan *protocol.NetworkMessage_Log, 100)
	progressBarChan := make(chan int)
	totalChunksChan := make(chan int)

	prouter := prouter.NewMessageRouter(conn)
	prouter.AddRoute(logChan, func(msg proto.Message) bool {
		_, ok := msg.(*protocol.NetworkMessage).MessageType.(*protocol.NetworkMessage_Log)
		return ok
	})
	prouter.Start()

	return &Client{
		DaemonConn:      conn,
		FileStore:       fileStore,
		ChunkStore:      chunkStore,
		IPCSocketIndex:  index,
		Logger:          logger,
		LogChan:         logChan,
		ProgressBarChan: progressBarChan,
		TotalChunksChan: totalChunksChan,
	}, nil
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

func (c *Client) SendDownloadSignal(filePath string) error {
	netMsg := &protocol.NetworkMessage{
		MessageType: &protocol.NetworkMessage_SignalDownload{
			SignalDownload: &protocol.SignalDownloadMessage{
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

func (c *Client) ListenForDaemonLogs(done chan struct{}, logger *logrus.Logger) {
	for {
		select {
		case logMsg := <-c.LogChan:
			msg := logMsg.Log.GetMessage()
			if msg == "done" {
				close(done)
				return
			}
			if msg == "progress" {
				c.ProgressBarChan <- 1 // 1 is any number, just to notify progress
				continue
			}
			if msg[0] == '+' {
				totalChunksStr := msg[1:]
				totalChunks, err := strconv.Atoi(totalChunksStr)
				if err != nil {
					logger.Warnf("Error converting string to int: %v", err)
				}
				c.TotalChunksChan <- totalChunks
				continue
			}
			logger.Info(msg)
		}
	}
}
