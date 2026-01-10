package tracker

import (
	"testing"

	"github.com/rudransh-shrivastava/peer-it/internal/protocol"
	"github.com/rudransh-shrivastava/peer-it/internal/transport"
)

func TestStoreAddPeer(t *testing.T) {
	store := NewStore()

	hash1 := protocol.FileHash{0x01}
	hash2 := protocol.FileHash{0x02}
	hashes := []protocol.FileHash{hash1, hash2}

	var peer1 *transport.Peer

	added := store.AddPeer(hashes, peer1)
	if added != 2 {
		t.Errorf("Expected 2 files added, got %d", added)
	}

	if len(store.peers[hash1]) != 1 {
		t.Errorf("Expected 1 peer for hash1, got %d", len(store.peers[hash1]))
	}

	if len(store.peers[hash2]) != 1 {
		t.Errorf("Expected 1 peer for hash2, got %d", len(store.peers[hash2]))
	}
}

func TestStoreAddPeerNoDuplicate(t *testing.T) {
	store := NewStore()

	hash1 := protocol.FileHash{0x01}
	hashes := []protocol.FileHash{hash1}

	var peer1 *transport.Peer

	added1 := store.AddPeer(hashes, peer1)
	added2 := store.AddPeer(hashes, peer1)

	if added1 != 1 {
		t.Errorf("First add: expected 1, got %d", added1)
	}

	if added2 != 0 {
		t.Errorf("Second add: expected 0 (duplicate), got %d", added2)
	}

	if len(store.peers[hash1]) != 1 {
		t.Errorf("Expected 1 peer (no duplicate), got %d", len(store.peers[hash1]))
	}
}

func TestStoreAddPeerMultiplePeers(t *testing.T) {
	store := NewStore()

	hash1 := protocol.FileHash{0x01}
	hashes := []protocol.FileHash{hash1}

	peer1 := &transport.Peer{}
	peer2 := &transport.Peer{}

	store.AddPeer(hashes, peer1)
	store.AddPeer(hashes, peer2)

	if len(store.peers[hash1]) != 2 {
		t.Errorf("Expected 2 different peers, got %d", len(store.peers[hash1]))
	}
}
