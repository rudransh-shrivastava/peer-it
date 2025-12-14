package store_test

import (
	"context"
	"testing"

	"github.com/rudransh-shrivastava/peer-it/internal/db"
	"github.com/rudransh-shrivastava/peer-it/internal/store"
)

func setupTestDB(t *testing.T) (*store.FileStore, *store.ChunkStore, *store.PeerStore) {
	t.Helper()
	sqlDB, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	return store.NewFileStore(sqlDB), store.NewChunkStore(sqlDB), store.NewPeerStore(sqlDB)
}

func TestFileStore_CreateFile(t *testing.T) {
	fs, _, _ := setupTestDB(t)
	ctx := context.Background()

	file, created, err := fs.CreateFile(ctx, "test.txt", 1024, 256, 4, "abc123")
	if err != nil {
		t.Fatalf("CreateFile failed: %v", err)
	}
	if !created {
		t.Error("expected file to be created")
	}
	if file.Name != "test.txt" {
		t.Errorf("expected name 'test.txt', got %q", file.Name)
	}
	if file.Hash != "abc123" {
		t.Errorf("expected hash 'abc123', got %q", file.Hash)
	}
}

func TestFileStore_CreateFile_Duplicate(t *testing.T) {
	fs, _, _ := setupTestDB(t)
	ctx := context.Background()

	_, _, _ = fs.CreateFile(ctx, "test.txt", 1024, 256, 4, "abc123")

	file, created, err := fs.CreateFile(ctx, "test.txt", 1024, 256, 4, "abc123")
	if err != nil {
		t.Fatalf("second CreateFile failed: %v", err)
	}
	if created {
		t.Error("expected file NOT to be created (duplicate)")
	}
	if file.Hash != "abc123" {
		t.Errorf("expected existing file returned, got hash %q", file.Hash)
	}
}

func TestFileStore_GetFileByHash(t *testing.T) {
	fs, _, _ := setupTestDB(t)
	ctx := context.Background()

	_, _, _ = fs.CreateFile(ctx, "test.txt", 1024, 256, 4, "abc123")

	file, err := fs.GetFileByHash(ctx, "abc123")
	if err != nil {
		t.Fatalf("GetFileByHash failed: %v", err)
	}
	if file.Name != "test.txt" {
		t.Errorf("expected name 'test.txt', got %q", file.Name)
	}
}

func TestFileStore_GetFileByHash_NotFound(t *testing.T) {
	fs, _, _ := setupTestDB(t)
	ctx := context.Background()

	_, err := fs.GetFileByHash(ctx, "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent hash")
	}
}

func TestFileStore_GetFiles(t *testing.T) {
	fs, _, _ := setupTestDB(t)
	ctx := context.Background()

	_, _, _ = fs.CreateFile(ctx, "file1.txt", 1024, 256, 4, "hash1")
	_, _, _ = fs.CreateFile(ctx, "file2.txt", 2048, 256, 8, "hash2")

	files, err := fs.GetFiles(ctx)
	if err != nil {
		t.Fatalf("GetFiles failed: %v", err)
	}
	if len(files) != 2 {
		t.Errorf("expected 2 files, got %d", len(files))
	}
}

func TestChunkStore_CreateChunk(t *testing.T) {
	fs, cs, _ := setupTestDB(t)
	ctx := context.Background()

	file, _, _ := fs.CreateFile(ctx, "test.txt", 1024, 256, 4, "abc123")

	chunk, err := cs.CreateChunk(ctx, file.ID, 0, 256, "chunkhash0", false)
	if err != nil {
		t.Fatalf("CreateChunk failed: %v", err)
	}
	if chunk.ChunkIndex != 0 {
		t.Errorf("expected chunk index 0, got %d", chunk.ChunkIndex)
	}
	if chunk.IsAvailable != 0 {
		t.Errorf("expected chunk not available, got %d", chunk.IsAvailable)
	}
}

func TestChunkStore_GetChunks(t *testing.T) {
	fs, cs, _ := setupTestDB(t)
	ctx := context.Background()

	file, _, _ := fs.CreateFile(ctx, "test.txt", 1024, 256, 4, "abc123")

	for i := 0; i < 4; i++ {
		_, err := cs.CreateChunk(ctx, file.ID, i, 256, "chunkhash", i == 0)
		if err != nil {
			t.Fatalf("CreateChunk %d failed: %v", i, err)
		}
	}

	chunks, err := cs.GetChunks(ctx, "abc123")
	if err != nil {
		t.Fatalf("GetChunks failed: %v", err)
	}
	if len(chunks) != 4 {
		t.Errorf("expected 4 chunks, got %d", len(chunks))
	}
}

func TestChunkStore_MarkChunkAvailable(t *testing.T) {
	fs, cs, _ := setupTestDB(t)
	ctx := context.Background()

	file, _, _ := fs.CreateFile(ctx, "test.txt", 1024, 256, 4, "abc123")
	_, _ = cs.CreateChunk(ctx, file.ID, 0, 256, "chunkhash0", false)

	err := cs.MarkChunkAvailable(ctx, "abc123", 0)
	if err != nil {
		t.Fatalf("MarkChunkAvailable failed: %v", err)
	}

	chunk, err := cs.GetChunk(ctx, "abc123", 0)
	if err != nil {
		t.Fatalf("GetChunk failed: %v", err)
	}
	if chunk.IsAvailable != 1 {
		t.Errorf("expected chunk to be available, got %d", chunk.IsAvailable)
	}
}

func TestChunkStore_GetChunk(t *testing.T) {
	fs, cs, _ := setupTestDB(t)
	ctx := context.Background()

	file, _, _ := fs.CreateFile(ctx, "test.txt", 1024, 256, 4, "abc123")
	_, _ = cs.CreateChunk(ctx, file.ID, 2, 256, "chunkhash2", true)

	chunk, err := cs.GetChunk(ctx, "abc123", 2)
	if err != nil {
		t.Fatalf("GetChunk failed: %v", err)
	}
	if chunk.ChunkIndex != 2 {
		t.Errorf("expected chunk index 2, got %d", chunk.ChunkIndex)
	}
	if chunk.IsAvailable != 1 {
		t.Errorf("expected chunk available, got %d", chunk.IsAvailable)
	}
}

func TestPeerStore_CreatePeer(t *testing.T) {
	_, _, ps := setupTestDB(t)
	ctx := context.Background()

	id, err := ps.CreatePeer(ctx, "192.168.1.1", "8080")
	if err != nil {
		t.Fatalf("CreatePeer failed: %v", err)
	}
	if id == "" {
		t.Error("expected non-empty peer ID")
	}
}

func TestPeerStore_DeletePeer(t *testing.T) {
	_, _, ps := setupTestDB(t)
	ctx := context.Background()

	_, _ = ps.CreatePeer(ctx, "192.168.1.1", "8080")

	err := ps.DeletePeer(ctx, "192.168.1.1", "8080")
	if err != nil {
		t.Fatalf("DeletePeer failed: %v", err)
	}
}

func TestPeerStore_AddPeerToSwarm(t *testing.T) {
	fs, _, ps := setupTestDB(t)
	ctx := context.Background()

	_, _, _ = fs.CreateFile(ctx, "test.txt", 1024, 256, 4, "abc123")
	_, _ = ps.CreatePeer(ctx, "192.168.1.1", "8080")

	err := ps.AddPeerToSwarm(ctx, "192.168.1.1", "8080", "abc123")
	if err != nil {
		t.Fatalf("AddPeerToSwarm failed: %v", err)
	}
}

func TestPeerStore_GetPeersByFileHash(t *testing.T) {
	fs, _, ps := setupTestDB(t)
	ctx := context.Background()

	_, _, _ = fs.CreateFile(ctx, "test.txt", 1024, 256, 4, "abc123")
	_, _ = ps.CreatePeer(ctx, "192.168.1.1", "8080")
	_, _ = ps.CreatePeer(ctx, "192.168.1.2", "8081")
	_ = ps.AddPeerToSwarm(ctx, "192.168.1.1", "8080", "abc123")
	_ = ps.AddPeerToSwarm(ctx, "192.168.1.2", "8081", "abc123")

	peers, err := ps.GetPeersByFileHash(ctx, "abc123")
	if err != nil {
		t.Fatalf("GetPeersByFileHash failed: %v", err)
	}
	if len(peers) != 2 {
		t.Errorf("expected 2 peers, got %d", len(peers))
	}
}

func TestPeerStore_DropAllPeers(t *testing.T) {
	fs, _, ps := setupTestDB(t)
	ctx := context.Background()

	_, _, _ = fs.CreateFile(ctx, "test.txt", 1024, 256, 4, "abc123")
	_, _ = ps.CreatePeer(ctx, "192.168.1.1", "8080")
	_ = ps.AddPeerToSwarm(ctx, "192.168.1.1", "8080", "abc123")

	err := ps.DropAllPeers(ctx)
	if err != nil {
		t.Fatalf("DropAllPeers failed: %v", err)
	}

	peers, _ := ps.GetPeersByFileHash(ctx, "abc123")
	if len(peers) != 0 {
		t.Errorf("expected 0 peers after drop, got %d", len(peers))
	}
}
