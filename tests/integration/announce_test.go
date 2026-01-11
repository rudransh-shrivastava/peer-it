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
		FileCount: 2,
		Files: []protocol.FileEntry{
			{Hash: hash1, Name: "File Name 1", Size: 1024},
			{Hash: hash2, Name: "File Name 2", Size: 2048},
		},
	}

	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Connect failed: %v", err)
	}

	if err := client.SendToTracker(ctx, announceMsg); err != nil {
		t.Fatalf("SendToTracker failed: %v", err)
	}
}
