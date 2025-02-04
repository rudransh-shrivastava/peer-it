// prouter is a simple message router
// it routes Protobuf messages to channels
package prouter

import (
	"encoding/binary"
	"io"
	"log"
	"net"
	"reflect"

	"github.com/rudransh-shrivastava/peer-it/internal/shared/protocol"
	"google.golang.org/protobuf/proto"
)

// MessageRouter is a simple message router that routes
// Protobuf messages to channels based on a match function
type MessageRouter struct {
	Conn   net.Conn
	done   chan struct{}
	Routes map[interface{}]func(proto.Message) bool
}

func NewMessageRouter(conn net.Conn) *MessageRouter {
	return &MessageRouter{
		Conn:   conn,
		done:   make(chan struct{}),
		Routes: make(map[interface{}]func(proto.Message) bool),
	}
}

func (r *MessageRouter) AddRoute(ch interface{}, matchFn func(proto.Message) bool) {
	if reflect.TypeOf(ch).Kind() != reflect.Chan {
		log.Fatal("AddRoute: argument must be a channel")
	}
	r.Routes[ch] = matchFn
}

func (r *MessageRouter) Start() {
	go r.listen()
}

func (r *MessageRouter) stop() {
	close(r.done)
	r.Conn.Close()
}

func (r *MessageRouter) listen() {
	defer r.stop()

	for {
		select {
		case <-r.done:
			return
		default:
			var length uint32
			if err := binary.Read(r.Conn, binary.BigEndian, &length); err != nil {
				if err != io.EOF {
					log.Printf("Error reading length: %v", err)
				}
				return
			}

			msgBytes := make([]byte, length)
			if _, err := io.ReadFull(r.Conn, msgBytes); err != nil {
				log.Printf("Error reading message: %v", err)
				return
			}

			msg := &protocol.NetworkMessage{}
			if err := proto.Unmarshal(msgBytes, msg); err != nil {
				log.Printf("Protobuf unmarshal error: %v", err)
				continue
			}

			r.routeMessage(msg)
		}
	}
}

func (r *MessageRouter) WriteMessage(msg *protocol.NetworkMessage) error {
	data, err := proto.Marshal(msg)
	if err != nil {
		return err
	}

	msgLen := uint32(len(data))
	if err := binary.Write(r.Conn, binary.BigEndian, msgLen); err != nil {
		return err
	}

	if _, err := r.Conn.Write(data); err != nil {
		return err
	}

	return nil
}

func (r *MessageRouter) routeMessage(msg proto.Message) {
	networkMsg, ok := msg.(*protocol.NetworkMessage)
	if !ok {
		log.Println("Received invalid message type")
		return
	}

	concreteMsg := reflect.ValueOf(networkMsg.MessageType)
	if concreteMsg.IsNil() {
		return
	}

	for ch, matchFn := range r.Routes {
		if !matchFn(msg) {
			continue
		}

		chVal := reflect.ValueOf(ch)
		if !concreteMsg.Type().AssignableTo(chVal.Type().Elem()) {
			// Skip if message type doesn't match channel type
			continue
		}

		// no drop
		chVal.Send(concreteMsg)
		// chVal.Send(reflect.ValueOf(networkMsg.MessageType))
	}
}
