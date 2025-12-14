package transport

import (
	"context"
	"io"
)

type Transport interface {
	Connect(ctx context.Context, peerID string, metadata ConnectionMetadata) (Conn, error)
	Accept() <-chan Conn
	Close() error
}

type Conn interface {
	PeerID() string
	Send(data []byte) error
	Recv() <-chan []byte
	Close() error
}

type ConnectionMetadata struct {
	FileHash string
}

type Signaler interface {
	SendSignal(ctx context.Context, peerID string, signal []byte) error
	RecvSignal() <-chan Signal
	io.Closer
}

type Signal struct {
	PeerID  string
	Payload []byte
}
