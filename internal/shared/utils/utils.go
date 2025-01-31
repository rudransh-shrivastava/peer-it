package utils

import (
	"encoding/binary"
	"io"
	"log"
	"net"

	"github.com/rudransh-shrivastava/peer-it/internal/shared/protocol"
	"google.golang.org/protobuf/proto"
)

func SendAnnounceMsg(conn net.Conn, msg *protocol.AnnounceMessage) error {
	netMsg := &protocol.NetworkMessage{
		MessageType: &protocol.NetworkMessage_Announce{
			Announce: msg,
		},
	}
	err := SendNetMsg(conn, netMsg)
	if err != nil {
		log.Fatal(err)
		return err
	}
	return nil
}

func SendPeerListRequestMsg(conn net.Conn, msg *protocol.PeerListRequest) error {
	log.Printf("Requesting peer list for hash: %s", msg.GetFileHash())

	netMsg := &protocol.NetworkMessage{
		MessageType: &protocol.NetworkMessage_PeerListRequest{
			PeerListRequest: msg,
		},
	}
	err := SendNetMsg(conn, netMsg)
	if err != nil {
		log.Fatal(err)
		return err
	}
	return nil
}

func SendIntroductionMsg(conn net.Conn, msg *protocol.IntroductionMessage) error {
	netMsg := &protocol.NetworkMessage{
		MessageType: &protocol.NetworkMessage_Introduction{
			Introduction: msg,
		},
	}
	err := SendNetMsg(conn, netMsg)
	if err != nil {
		log.Fatal(err)
		return err
	}
	return nil
}

// SendNetMsg takes in a connection and a network message
// and sends the network message over the connection
func SendNetMsg(conn net.Conn, msg *protocol.NetworkMessage) error {
	data, err := proto.Marshal(msg)
	if err != nil {
		log.Printf("Error marshalling message: %v", err)
		return err
	}
	msgLen := uint32(len(data))
	if err := binary.Write(conn, binary.BigEndian, msgLen); err != nil {
		log.Printf("Error sending message length: %v", err)
		return err
	}

	if _, err := conn.Write(data); err != nil {
		log.Printf("Error sending message: %v", err)
		return err
	}
	return nil
}

// ReceiveNetMsg reads a network message from a connection
// and returns the network message
// It is a blocking call
func ReceiveNetMsg(conn net.Conn) *protocol.NetworkMessage {
	var msgLen uint32
	if err := binary.Read(conn, binary.BigEndian, &msgLen); err != nil {
		if err != io.EOF {
			log.Printf("Error reading message length: %v", err)
		}
	}
	data := make([]byte, msgLen)
	if _, err := io.ReadFull(conn, data); err != nil {
		log.Printf("Error reading message body: %v", err)
	}

	var netMsg protocol.NetworkMessage
	if err := proto.Unmarshal(data, &netMsg); err != nil {
		log.Printf("Error unmarshaling message: %v", err)
	}
	return &netMsg
}
