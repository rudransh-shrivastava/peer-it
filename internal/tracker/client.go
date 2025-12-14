package tracker

import (
	"context"
)

// Client communicates with a tracker server.
type Client interface {
	Connect(ctx context.Context) error
	Announce(ctx context.Context, files []FileInfo) error
	GetPeers(ctx context.Context, fileHash string) ([]PeerInfo, error)
	OnSignal(handler func(signal SignalMessage))
	SendSignal(ctx context.Context, targetPeerID string, payload []byte) error
	ID() string
	Close() error
}

// FileInfo describes a file for tracker announcement.
type FileInfo struct {
	Hash        string
	Name        string
	Size        int64
	ChunkSize   int
	TotalChunks int
}

// PeerInfo identifies a peer.
type PeerInfo struct {
	ID string
}

// SignalMessage is relayed signaling data from another peer.
type SignalMessage struct {
	SourcePeerID string
	Payload      []byte
}
