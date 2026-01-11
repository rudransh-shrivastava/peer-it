package tracker

import (
	"context"
	"crypto/sha256"
	"fmt"
	"testing"
	"time"

	"github.com/rudransh-shrivastava/peer-it/internal/logger"
	"github.com/rudransh-shrivastava/peer-it/internal/protocol"
	"github.com/rudransh-shrivastava/peer-it/internal/transport"
)

func TestNewServer(t *testing.T) {
	srv, err := NewServer(Config{
		Addr:   ":0",
		Logger: logger.NewLogger(),
	})
	if err != nil {
		t.Fatalf("NewServer failed: %v", err)
	}
	defer func() { _ = srv.Shutdown() }()

	if srv.Addr() == "" {
		t.Error("Expected non-empty address")
	}
}

func TestServerAddr(t *testing.T) {
	srv, err := NewServer(Config{Addr: ":0"})
	if err != nil {
		t.Fatalf("NewServer failed: %v", err)
	}
	defer func() { _ = srv.Shutdown() }()

	addr := srv.Addr()
	if addr == "" {
		t.Error("Expected non-empty address")
	}
}

func TestServerHandlePing(t *testing.T) {
	srv, client := setupServerClient(t)
	defer func() { _ = srv.Shutdown() }()
	defer func() { _ = client.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Send(ctx, &protocol.Ping{}); err != nil {
		t.Fatalf("Send Ping failed: %v", err)
	}

	msg, err := client.Receive(ctx)
	if err != nil {
		t.Fatalf("Receive failed: %v", err)
	}

	if _, ok := msg.(*protocol.Pong); !ok {
		t.Errorf("Expected Pong, got %T", msg)
	}
}

func TestServerHandlePeerAnnounce(t *testing.T) {
	srv, client := setupServerClient(t)
	defer func() { _ = srv.Shutdown() }()
	defer func() { _ = client.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	fileName := "file1.txt"
	fileSize := uint64(1024)
	hash1 := sha256.Sum256([]byte(fmt.Sprintf("%s%d", fileName, fileSize)))
	announce := &protocol.PeerAnnounce{
		FileCount: 1,
		Files: []protocol.FileEntry{
			{Hash: hash1, Name: fileName, Size: fileSize},
		},
	}

	if err := client.Send(ctx, announce); err != nil {
		t.Fatalf("Send PeerAnnounce failed: %v", err)
	}

	time.Sleep(50 * time.Millisecond)

	if srv.store.files[hash1] == nil || len(srv.store.files[hash1].peers) != 1 {
		t.Errorf("Expected 1 peer in store for hash1")
	}
}

func TestGenerateHash(t *testing.T) {
	file := &protocol.FileEntry{
		Name: "example.txt",
		Size: 2048,
	}

	expectedHash := sha256.Sum256([]byte(fmt.Sprintf("%s%d", file.Name, file.Size)))
	actualHash := generateHash(file)

	if actualHash != expectedHash {
		t.Errorf("Hash mismatch: expected %x, got %x", expectedHash, actualHash)
	}
}

func TestGenerateHashDifferentInputs(t *testing.T) {
	file1 := &protocol.FileEntry{Name: "file.txt", Size: 100}
	file2 := &protocol.FileEntry{Name: "file.txt", Size: 200}
	file3 := &protocol.FileEntry{Name: "other.txt", Size: 100}

	hash1 := generateHash(file1)
	hash2 := generateHash(file2)
	hash3 := generateHash(file3)

	if hash1 == hash2 {
		t.Error("Different sizes should produce different hashes")
	}

	if hash1 == hash3 {
		t.Error("Different names should produce different hashes")
	}
}

func TestServerHandlePeerAnnounceMismatchedFileCount(t *testing.T) {
	srv, client := setupServerClient(t)
	defer func() { _ = srv.Shutdown() }()
	defer func() { _ = client.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	fileName := "file1.txt"
	fileSize := uint64(1024)
	hash1 := sha256.Sum256([]byte(fmt.Sprintf("%s%d", fileName, fileSize)))

	// FileCount is 2 but only 1 file provided
	announce := &protocol.PeerAnnounce{
		FileCount: 2,
		Files: []protocol.FileEntry{
			{Hash: hash1, Name: fileName, Size: fileSize},
		},
	}

	if err := client.Send(ctx, announce); err != nil {
		t.Fatalf("Send PeerAnnounce failed: %v", err)
	}

	time.Sleep(50 * time.Millisecond)

	// Should NOT be added to store due to validation failure
	if srv.store.files[hash1] != nil {
		t.Errorf("Expected file to NOT be in store due to mismatched FileCount")
	}
}

func TestServerHandlePeerAnnounceInvalidHash(t *testing.T) {
	srv, client := setupServerClient(t)
	defer func() { _ = srv.Shutdown() }()
	defer func() { _ = client.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Use an invalid hash that doesn't match name+size
	invalidHash := protocol.FileHash{0x01, 0x02, 0x03}
	announce := &protocol.PeerAnnounce{
		FileCount: 1,
		Files: []protocol.FileEntry{
			{Hash: invalidHash, Name: "file1.txt", Size: 1024},
		},
	}

	if err := client.Send(ctx, announce); err != nil {
		t.Fatalf("Send PeerAnnounce failed: %v", err)
	}

	time.Sleep(50 * time.Millisecond)

	// Should NOT be added to store due to invalid hash
	if srv.store.files[invalidHash] != nil {
		t.Errorf("Expected file to NOT be in store due to invalid hash")
	}
}

func setupServerClient(t *testing.T) (*Server, *transport.Peer) {
	t.Helper()

	srv, err := NewServer(Config{
		Addr:   ":0",
		Logger: logger.NewLogger(),
	})
	if err != nil {
		t.Fatalf("NewServer failed: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)

	go func() {
		_ = srv.Start(ctx)
	}()

	time.Sleep(50 * time.Millisecond)

	tr, err := transport.NewTransport(":0")
	if err != nil {
		t.Fatalf("NewTransport failed: %v", err)
	}

	client, err := tr.Dial(ctx, srv.Addr())
	if err != nil {
		t.Fatalf("Dial failed: %v", err)
	}

	t.Cleanup(func() {
		cancel()
		_ = tr.Close()
	})

	return srv, client
}
