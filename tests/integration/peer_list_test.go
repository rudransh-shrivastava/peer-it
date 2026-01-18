package integration

import (
	"crypto/sha256"
	"fmt"
	"testing"

	"github.com/rudransh-shrivastava/peer-it/internal/protocol"
)

func TestPeerListReqEmpty(t *testing.T) {
	net := NewTestNetwork(t)
	defer net.Close()

	ctx := net.Context()
	client := net.NewClient()

	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Connect failed: %v", err)
	}

	hash := protocol.FileHash{0x01, 0x02, 0x03}
	if err := client.SendToTracker(ctx, &protocol.PeerListReq{FileHash: hash}); err != nil {
		t.Fatalf("SendToTracker failed: %v", err)
	}

	msg, err := client.ReceiveFromTracker(ctx)
	if err != nil {
		t.Fatalf("ReceiveFromTracker failed: %v", err)
	}

	res, ok := msg.(*protocol.PeerListRes)
	if !ok {
		t.Fatalf("Expected PeerListRes, got %T", msg)
	}

	if len(res.Peers) != 0 {
		t.Errorf("Expected 0 peers, got %d", len(res.Peers))
	}
}

func TestPeerListReqAfterAnnounce(t *testing.T) {
	net := NewTestNetwork(t)
	defer net.Close()

	ctx := net.Context()

	fileName := "shared.txt"
	fileSize := uint64(1024)
	hash := sha256.Sum256([]byte(fmt.Sprintf("%s%d", fileName, fileSize)))

	client1 := net.NewClient()
	if err := client1.Connect(ctx); err != nil {
		t.Fatalf("Client1 Connect failed: %v", err)
	}

	announce := &protocol.PeerAnnounce{
		FileCount: 1,
		Files:     []protocol.FileEntry{{Hash: hash, Name: fileName, Size: fileSize}},
	}
	if err := client1.SendToTracker(ctx, announce); err != nil {
		t.Fatalf("Client1 SendToTracker failed: %v", err)
	}

	client2 := net.NewClient()
	if err := client2.Connect(ctx); err != nil {
		t.Fatalf("Client2 Connect failed: %v", err)
	}

	if err := client2.SendToTracker(ctx, &protocol.PeerListReq{FileHash: hash}); err != nil {
		t.Fatalf("Client2 SendToTracker failed: %v", err)
	}

	msg, err := client2.ReceiveFromTracker(ctx)
	if err != nil {
		t.Fatalf("Client2 ReceiveFromTracker failed: %v", err)
	}

	res, ok := msg.(*protocol.PeerListRes)
	if !ok {
		t.Fatalf("Expected PeerListRes, got %T", msg)
	}

	if len(res.Peers) != 1 {
		t.Errorf("Expected 1 peer, got %d", len(res.Peers))
	}
}
