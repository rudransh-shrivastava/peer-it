// Package transport defines interfaces for peer-to-peer connections.
package transport

import (
	"context"
	"io"
)

// Transport handles peer connection establishment.
type Transport interface {
	Connect(ctx context.Context, peerID string, metadata ConnectionMetadata) (Conn, error)
	Accept() <-chan Conn
	Close() error
}

// Conn represents a bidirectional connection to a peer.
type Conn interface {
	PeerID() string
	Send(data []byte) error
	Recv() <-chan []byte
	Close() error
}

// ConnectionMetadata contains context for connection establishment.
type ConnectionMetadata struct {
	FileHash string
}

// Signaler handles out-of-band signaling for connection setup.
type Signaler interface {
	SendSignal(ctx context.Context, peerID string, signal []byte) error
	RecvSignal() <-chan Signal
	io.Closer
}

// Signal is an incoming signaling message from a peer.
type Signal struct {
	PeerID  string
	Payload []byte
}
