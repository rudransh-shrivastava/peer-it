package store

import (
	"context"

	"github.com/rudransh-shrivastava/peer-it/internal/db"
)

// FileRepository defines file metadata operations.
type FileRepository interface {
	CreateFile(ctx context.Context, name string, size int64, maxChunkSize, totalChunks int, hash string) (db.File, bool, error)
	GetFiles(ctx context.Context) ([]db.File, error)
	GetFileByHash(ctx context.Context, hash string) (db.File, error)
	GetFileNameByHash(ctx context.Context, hash string) (string, error)
}

// ChunkRepository defines chunk metadata operations.
type ChunkRepository interface {
	CreateChunk(ctx context.Context, fileID int64, index, size int, hash string, isAvailable bool) (db.Chunk, error)
	GetChunk(ctx context.Context, fileHash string, chunkIndex int) (db.Chunk, error)
	GetChunks(ctx context.Context, fileHash string) ([]db.Chunk, error)
	MarkChunkAvailable(ctx context.Context, fileHash string, chunkIndex int) error
}

// PeerRepository defines peer storage operations.
type PeerRepository interface {
	CreatePeer(ctx context.Context, ip, port string) (string, error)
	DeletePeer(ctx context.Context, ip, port string) error
	AddPeerToSwarm(ctx context.Context, ip, port, fileHash string) error
	DropAllPeers(ctx context.Context) error
	GetPeersByFileHash(ctx context.Context, fileHash string) ([]db.Peer, error)
}
