package client

import (
	"net"
	"strconv"

	"github.com/rudransh-shrivastava/peer-it/internal/shared/protocol"
	"github.com/rudransh-shrivastava/peer-it/internal/shared/prouter"
	"github.com/rudransh-shrivastava/peer-it/internal/shared/utils"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/proto"
)

type Client struct {
	DaemonConn      net.Conn
	IPCSocketIndex  string
	Logger          *logrus.Logger
	LogChan         chan *protocol.NetworkMessage_Log
	ProgressBarChan chan int
	TotalChunksChan chan int
}

func NewClient(index string, logger *logrus.Logger) (*Client, error) {
	addr := "127.0.0.1:707" + index
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		logger.Fatal(err)
		return &Client{}, err
	}

	logChan := make(chan *protocol.NetworkMessage_Log, 100)
	progressBarChan := make(chan int)
	totalChunksChan := make(chan int)

	router := prouter.NewMessageRouter(conn)
	router.AddRoute(logChan, func(msg proto.Message) bool {
		_, ok := msg.(*protocol.NetworkMessage).MessageType.(*protocol.NetworkMessage_Log)
		return ok
	})
	router.Start()

	return &Client{
		DaemonConn:      conn,
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

func (c *Client) ListenForDaemonLogs(done chan struct{}) {
	for logMsg := range c.LogChan {
		msg := logMsg.Log.GetMessage()
		if msg == "done" {
			close(done)
			return
		}
		if msg == "progress" {
			c.ProgressBarChan <- 1
			continue
		}
		if msg[0] == '+' {
			totalChunksStr := msg[1:]
			totalChunks, err := strconv.Atoi(totalChunksStr)
			if err != nil {
				c.Logger.Warnf("Error converting string to int: %v", err)
			}
			c.TotalChunksChan <- totalChunks
			continue
		}
		c.Logger.Info(msg)
	}
}
