package client

import (
	"log"
	"net"

	"github.com/rudransh-shrivastava/peer-it/internal/shared/protocol"
	"github.com/rudransh-shrivastava/peer-it/internal/shared/utils"
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
	netMsg := utils.ReceiveNetMsg(c.DaemonConn)
	switch msg := netMsg.MessageType.(type) {
	case *protocol.NetworkMessage_PeerListResponse:
		log.Printf("Received peer list response from daemon %+v", msg)
	default:
		log.Printf("Unexpected message type: %T", msg)
	}
	peerListResponse := netMsg.GetPeerListResponse()
	return peerListResponse
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
