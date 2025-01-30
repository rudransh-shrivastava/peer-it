package client

import (
	"encoding/binary"
	"io"
	"log"
	"net"
	"time"

	"github.com/rudransh-shrivastava/peer-it/internal/shared/protocol"
	"github.com/rudransh-shrivastava/peer-it/internal/shared/utils"
	"google.golang.org/protobuf/proto"
)

// This client communicates with the daemon using unix sockets
type Client struct {
	DaemonConn net.Conn
}

func NewClient() (*Client, error) {
	conn, err := net.Dial("unix", "/tmp/pit-daemon.sock")
	if err != nil {
		log.Fatal(err)
		return &Client{}, err
	}
	return &Client{DaemonConn: conn}, nil
}

func (c *Client) AnnounceFile(msg *protocol.AnnounceMessage) {
	netMsg := &protocol.NetworkMessage{
		MessageType: &protocol.NetworkMessage_Announce{
			Announce: msg,
		},
	}
	err := utils.SendNetMsg(c.DaemonConn, netMsg)
	if err != nil {
		log.Fatal(err)
	}
}

func (c *Client) WaitForPeerList() *protocol.PeerListResponse {
	netMsg := c.receiveMsg()
	switch msg := netMsg.MessageType.(type) {
	case *protocol.NetworkMessage_PeerListResponse:
		log.Printf("Received peer list response from daemon %+v", msg)
	default:
		log.Printf("Unexpected message type: %T", msg)
	}
	peerListResponse := netMsg.GetPeerListResponse()
	return peerListResponse
}

func (c *Client) receiveMsg() *protocol.NetworkMessage {
	// set deadline
	c.DaemonConn.SetReadDeadline(time.Now().Add(time.Second * 10))
	var msgLen uint32
	if err := binary.Read(c.DaemonConn, binary.BigEndian, &msgLen); err != nil {
		if err != io.EOF {
			log.Printf("Error reading message length: %v", err)
		}
	}
	data := make([]byte, msgLen)
	if _, err := io.ReadFull(c.DaemonConn, data); err != nil {
		log.Printf("Error reading message body: %v", err)
	}

	var netMsg protocol.NetworkMessage
	if err := proto.Unmarshal(data, &netMsg); err != nil {
		log.Printf("Error unmarshaling message: %v", err)
	}
	return &netMsg
}

func (c *Client) RequestPeerList(msg *protocol.PeerListRequest) {
	log.Printf("Requesting peer list for hash: %s", msg.GetFileHash())

	netMsg := &protocol.NetworkMessage{
		MessageType: &protocol.NetworkMessage_PeerListRequest{
			PeerListRequest: msg,
		},
	}
	err := utils.SendNetMsg(c.DaemonConn, netMsg)
	if err != nil {
		log.Fatal(err)
	}
}
