package client

import (
	"encoding/binary"
	"io"
	"log"
	"net"
	"time"

	"github.com/rudransh-shrivastava/peer-it/internal/shared/protocol"
	"google.golang.org/protobuf/proto"
)

// This client communicates with the daemon using unix sockets

type Client struct {
	Conn net.Conn
}

func NewClient() (*Client, error) {
	conn, err := net.Dial("unix", "/tmp/pit-daemon.sock")
	if err != nil {
		log.Fatal(err)
		return &Client{}, err
	}
	return &Client{Conn: conn}, nil
}

func (c *Client) AnnounceFile(msg *protocol.AnnounceMessage) {
	netMsg := &protocol.NetworkMessage{
		MessageType: &protocol.NetworkMessage_Announce{
			Announce: msg,
		},
	}
	c.sendMessage(netMsg)
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
	c.Conn.SetReadDeadline(time.Now().Add(time.Second * 10))
	var msgLen uint32
	if err := binary.Read(c.Conn, binary.BigEndian, &msgLen); err != nil {
		if err != io.EOF {
			log.Printf("Error reading message length: %v", err)
		}
	}
	data := make([]byte, msgLen)
	if _, err := io.ReadFull(c.Conn, data); err != nil {
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
	c.sendMessage(netMsg)
}

func (c *Client) sendMessage(msg *protocol.NetworkMessage) error {
	data, err := proto.Marshal(msg)
	if err != nil {
		log.Println("Error marshalling message:", err)
		return err
	}
	msgLen := uint32(len(data))
	if err := binary.Write(c.Conn, binary.BigEndian, msgLen); err != nil {
		log.Printf("Error sending message length: %v", err)
		return err
	}

	if _, err := c.Conn.Write(data); err != nil {
		log.Printf("Error sending message: %v", err)
		return err
	}
	return nil
}
