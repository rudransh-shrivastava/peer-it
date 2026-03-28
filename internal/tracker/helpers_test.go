package tracker

import (
	"crypto/sha256"
	"testing"

	"github.com/rudransh-shrivastava/peer-it/internal/protocol"
)

func TestGenerateHash(t *testing.T) {
	file := &protocol.FileEntry{Name: "test.txt", Size: 1024}
	hash := generateHash(file)

	expected := sha256.Sum256([]byte("test.txt1024"))
	if hash != expected {
		t.Errorf("Hash mismatch")
	}
}

func TestGeneratePeerID(t *testing.T) {
	// use the shared mockConn from mocks_test.go
	id := generatePeerID()

	if len(id) != protocol.NodeIDSize {
		t.Fatalf("unexpected NodeID size: got %d, want %d", len(id), protocol.NodeIDSize)
	}

	// Generate another ID and ensure they are different (extremely likely).
	id2 := generatePeerID()
	if id == id2 {
		t.Error("generatePeerID returned the same ID twice")
	}
}

func TestGenerateHashDifferentInputs(t *testing.T) {
	file1 := &protocol.FileEntry{Name: "file1.txt", Size: 1024}
	file2 := &protocol.FileEntry{Name: "file2.txt", Size: 1024}

	hash1 := generateHash(file1)
	hash2 := generateHash(file2)

	if hash1 == hash2 {
		t.Error("Different files should have different hashes")
	}
}
