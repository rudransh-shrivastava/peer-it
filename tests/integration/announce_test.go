package integration

import (
	"crypto/sha256"
	"testing"

	"github.com/rudransh-shrivastava/peer-it/internal/protocol"
)

func TestPeerTrackerAnnounce(t *testing.T) {
	net := NewTestNetwork(t)
	defer net.Close()

	ctx := net.Context()
	client := net.NewClient()

	hash1 := sha256.Sum256([]byte("File Name 1"))
	hash2 := sha256.Sum256([]byte("File Name 2"))
	announceMsg := &protocol.PeerAnnounce{
		FileCount:  2,
		FileHashes: []protocol.FileHash{hash1, hash2},
	}

	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Connect failed: %v", err)
	}

	if err := client.SendToTracker(ctx, announceMsg); err != nil {
		t.Fatalf("SendToTracker failed: %v", err)
	}
}
