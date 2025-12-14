// Package store provides database access for files, chunks, and peers.
package store

import (
	"context"
	"database/sql"
	"strconv"
	"time"

	"github.com/rudransh-shrivastava/peer-it/internal/db"
)

type FileStore struct {
	queries *db.Queries
}

func NewFileStore(sqlDB *sql.DB) *FileStore {
	return &FileStore{queries: db.New(sqlDB)}
}

func (fs *FileStore) CreateFile(ctx context.Context, name string, size int64, maxChunkSize, totalChunks int, hash string) (db.File, bool, error) {
	existing, err := fs.queries.GetFileByHash(ctx, hash)
	if err == nil {
		return existing, false, nil
	}
	if err != sql.ErrNoRows {
		return db.File{}, false, err
	}

	file, err := fs.queries.CreateFile(ctx, db.CreateFileParams{
		Name:         name,
		Size:         size,
		MaxChunkSize: int64(maxChunkSize),
		TotalChunks:  int64(totalChunks),
		Hash:         hash,
		CreatedAt:    time.Now().Unix(),
	})
	if err != nil {
		return db.File{}, false, err
	}

	return file, true, nil
}

func (fs *FileStore) GetFiles(ctx context.Context) ([]db.File, error) {
	return fs.queries.GetAllFiles(ctx)
}

func (fs *FileStore) GetFileByHash(ctx context.Context, hash string) (db.File, error) {
	return fs.queries.GetFileByHash(ctx, hash)
}

func (fs *FileStore) GetFileNameByHash(ctx context.Context, hash string) (string, error) {
	return fs.queries.GetFileNameByHash(ctx, hash)
}

type ChunkStore struct {
	queries *db.Queries
}

func NewChunkStore(sqlDB *sql.DB) *ChunkStore {
	return &ChunkStore{queries: db.New(sqlDB)}
}

func (cs *ChunkStore) CreateChunk(ctx context.Context, fileID int64, index, size int, hash string, isAvailable bool) (db.Chunk, error) {
	available := int64(0)
	if isAvailable {
		available = 1
	}

	return cs.queries.CreateChunk(ctx, db.CreateChunkParams{
		FileID:      fileID,
		ChunkIndex:  int64(index),
		ChunkSize:   int64(size),
		ChunkHash:   hash,
		IsAvailable: available,
	})
}

func (cs *ChunkStore) GetChunk(ctx context.Context, fileHash string, chunkIndex int) (db.Chunk, error) {
	return cs.queries.GetChunk(ctx, db.GetChunkParams{
		Hash:       fileHash,
		ChunkIndex: int64(chunkIndex),
	})
}

func (cs *ChunkStore) GetChunks(ctx context.Context, fileHash string) ([]db.Chunk, error) {
	return cs.queries.GetChunksByFileHash(ctx, fileHash)
}

func (cs *ChunkStore) MarkChunkAvailable(ctx context.Context, fileHash string, chunkIndex int) error {
	return cs.queries.MarkChunkAvailable(ctx, db.MarkChunkAvailableParams{
		Hash:       fileHash,
		ChunkIndex: int64(chunkIndex),
	})
}

type PeerStore struct {
	queries *db.Queries
}

func NewPeerStore(sqlDB *sql.DB) *PeerStore {
	return &PeerStore{queries: db.New(sqlDB)}
}

func (ps *PeerStore) CreatePeer(ctx context.Context, ip, port string) (string, error) {
	peer, err := ps.queries.CreatePeer(ctx, db.CreatePeerParams{
		IpAddress: ip,
		Port:      port,
	})
	if err != nil {
		return "", err
	}
	return strconv.FormatInt(peer.ID, 10), nil
}

func (ps *PeerStore) DeletePeer(ctx context.Context, ip, port string) error {
	return ps.queries.DeletePeerByIPPort(ctx, db.DeletePeerByIPPortParams{
		IpAddress: ip,
		Port:      port,
	})
}

func (ps *PeerStore) AddPeerToSwarm(ctx context.Context, ip, port, fileHash string) error {
	return ps.queries.AddPeerToSwarm(ctx, db.AddPeerToSwarmParams{
		IpAddress: ip,
		Port:      port,
		Hash:      fileHash,
	})
}

func (ps *PeerStore) DropAllPeers(ctx context.Context) error {
	if err := ps.queries.DeleteAllSwarms(ctx); err != nil {
		return err
	}
	return ps.queries.DeleteAllPeers(ctx)
}

func (ps *PeerStore) GetPeersByFileHash(ctx context.Context, fileHash string) ([]db.Peer, error) {
	return ps.queries.GetPeersByFileHash(ctx, fileHash)
}

var (
	_ FileRepository  = (*FileStore)(nil)
	_ ChunkRepository = (*ChunkStore)(nil)
	_ PeerRepository  = (*PeerStore)(nil)
)
