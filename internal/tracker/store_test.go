package tracker

import (
	"testing"

	"github.com/rudransh-shrivastava/peer-it/internal/protocol"
	"github.com/rudransh-shrivastava/peer-it/internal/transport"
)

func TestStoreAddFiles(t *testing.T) {
	store := NewStore()

	files := []protocol.FileEntry{
		{Hash: protocol.FileHash{0x01}, Name: "file1.txt", Size: 1024},
		{Hash: protocol.FileHash{0x02}, Name: "file2.txt", Size: 2048},
	}

	added := store.AddFiles(&files)
	if added != 2 {
		t.Errorf("Expected 2 files added, got %d", added)
	}

	if _, exists := store.files[files[0].Hash]; !exists {
		t.Errorf("Expected file1 to exist in store")
	}

	if _, exists := store.files[files[1].Hash]; !exists {
		t.Errorf("Expected file2 to exist in store")
	}
}

func TestStoreAddFilesNoDuplicate(t *testing.T) {
	store := NewStore()

	files := []protocol.FileEntry{
		{Hash: protocol.FileHash{0x01}, Name: "file1.txt", Size: 1024},
	}

	added1 := store.AddFiles(&files)
	added2 := store.AddFiles(&files)

	if added1 != 1 {
		t.Errorf("First add: expected 1, got %d", added1)
	}

	if added2 != 0 {
		t.Errorf("Second add: expected 0 (duplicate), got %d", added2)
	}
}

func TestStoreAddPeer(t *testing.T) {
	store := NewStore()

	hash1 := protocol.FileHash{0x01}
	files := []protocol.FileEntry{
		{Hash: hash1, Name: "file1.txt", Size: 1024},
	}

	// First add the files to the store
	store.AddFiles(&files)

	peer1 := &transport.Peer{}

	added := store.AddPeer(&files, peer1)
	if added != 1 {
		t.Errorf("Expected 1 peer added, got %d", added)
	}

	if len(store.files[hash1].peers) != 1 {
		t.Errorf("Expected 1 peer for hash1, got %d", len(store.files[hash1].peers))
	}
}

func TestStoreAddPeerNoDuplicate(t *testing.T) {
	store := NewStore()

	hash1 := protocol.FileHash{0x01}
	files := []protocol.FileEntry{
		{Hash: hash1, Name: "file1.txt", Size: 1024},
	}

	// First add the files to the store
	store.AddFiles(&files)

	peer1 := &transport.Peer{}

	added1 := store.AddPeer(&files, peer1)
	added2 := store.AddPeer(&files, peer1)

	if added1 != 1 {
		t.Errorf("First add: expected 1, got %d", added1)
	}

	if added2 != 0 {
		t.Errorf("Second add: expected 0 (duplicate), got %d", added2)
	}

	if len(store.files[hash1].peers) != 1 {
		t.Errorf("Expected 1 peer (no duplicate), got %d", len(store.files[hash1].peers))
	}
}

func TestStoreAddPeerMultiplePeers(t *testing.T) {
	store := NewStore()

	hash1 := protocol.FileHash{0x01}
	files := []protocol.FileEntry{
		{Hash: hash1, Name: "file1.txt", Size: 1024},
	}

	// First add the files to the store
	store.AddFiles(&files)

	peer1 := &transport.Peer{}
	peer2 := &transport.Peer{}

	store.AddPeer(&files, peer1)
	store.AddPeer(&files, peer2)

	if len(store.files[hash1].peers) != 2 {
		t.Errorf("Expected 2 different peers, got %d", len(store.files[hash1].peers))
	}
}
