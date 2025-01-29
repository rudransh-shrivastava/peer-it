package client

import (
	"encoding/binary"
	"log"
	"net"

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
