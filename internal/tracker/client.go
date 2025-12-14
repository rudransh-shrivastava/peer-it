package tracker

import (
	"context"
)

type Client interface {
	Connect(ctx context.Context) error
	Announce(ctx context.Context, files []FileInfo) error
	GetPeers(ctx context.Context, fileHash string) ([]PeerInfo, error)
	OnSignal(handler func(signal SignalMessage))
	SendSignal(ctx context.Context, targetPeerID string, payload []byte) error
	ID() string
	Close() error
}

type FileInfo struct {
	Hash        string
	Name        string
	Size        int64
	ChunkSize   int
	TotalChunks int
}

type PeerInfo struct {
	ID string
}

type SignalMessage struct {
	SourcePeerID string
	Payload      []byte
}
