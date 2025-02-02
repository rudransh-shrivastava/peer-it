package utils

import (
	"encoding/binary"
	"io"
	"net"

	"github.com/rudransh-shrivastava/peer-it/internal/shared/protocol"
	"google.golang.org/protobuf/proto"
)

// TODO: make generic
func SendRegisterMsg(conn net.Conn, msg *protocol.RegisterMessage) error {
	netMsg := &protocol.NetworkMessage{
		MessageType: &protocol.NetworkMessage_Register{
			Register: msg,
		},
	}
	err := SendNetMsg(conn, netMsg)
	if err != nil {
		return err
	}
	return nil
}

func SendAnnounceMsg(conn net.Conn, msg *protocol.AnnounceMessage) error {
	netMsg := &protocol.NetworkMessage{
		MessageType: &protocol.NetworkMessage_Announce{
			Announce: msg,
		},
	}
	err := SendNetMsg(conn, netMsg)
	if err != nil {
		return err
	}
	return nil
}

func SendPeerListRequestMsg(conn net.Conn, msg *protocol.PeerListRequest) error {

	netMsg := &protocol.NetworkMessage{
		MessageType: &protocol.NetworkMessage_PeerListRequest{
			PeerListRequest: msg,
		},
	}
	err := SendNetMsg(conn, netMsg)
	if err != nil {
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
		return err
	}
	return nil
}

// SendNetMsg takes in a connection and a network message
// and sends the network message over the connection
func SendNetMsg(conn net.Conn, msg *protocol.NetworkMessage) error {
	data, err := proto.Marshal(msg)
	if err != nil {
		return err
	}
	msgLen := uint32(len(data))
	if err := binary.Write(conn, binary.BigEndian, msgLen); err != nil {
		return err
	}

	if _, err := conn.Write(data); err != nil {
		return err
	}
	return nil
}

// ReceiveNetMsg reads a network message from a connection
// and returns the network message
// It is a blocking call
func ReceiveNetMsg(conn net.Conn) (*protocol.NetworkMessage, error) {
	var msgLen uint32
	if err := binary.Read(conn, binary.BigEndian, &msgLen); err != nil {
		if err != io.EOF {
			return nil, err
		}
	}
	data := make([]byte, msgLen)
	if _, err := io.ReadFull(conn, data); err != nil {
		return nil, err
	}

	var netMsg protocol.NetworkMessage
	if err := proto.Unmarshal(data, &netMsg); err != nil {
		return nil, err
	}
	return &netMsg, nil
}
