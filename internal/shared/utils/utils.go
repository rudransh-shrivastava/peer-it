package utils

import (
	"encoding/binary"
	"io"
	"log"
	"net"

	"github.com/rudransh-shrivastava/peer-it/internal/shared/protocol"
	"google.golang.org/protobuf/proto"
)

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
