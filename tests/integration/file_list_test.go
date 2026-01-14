package integration

import (
	"crypto/sha256"
	"fmt"
	"testing"

	"github.com/rudransh-shrivastava/peer-it/internal/protocol"
)

func TestFileListReqEmpty(t *testing.T) {
	net := NewTestNetwork(t)
	defer net.Close()

	ctx := net.Context()
	client := net.NewClient()

	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Connect failed: %v", err)
	}

	if err := client.SendToTracker(ctx, &protocol.FileListReq{}); err != nil {
		t.Fatalf("SendToTracker failed: %v", err)
	}

	msg, err := client.ReceiveFromTracker(ctx)
	if err != nil {
		t.Fatalf("ReceiveFromTracker failed: %v", err)
	}

	res, ok := msg.(*protocol.FileListRes)
	if !ok {
		t.Fatalf("Expected FileListRes, got %T", msg)
	}

	if len(res.Files) != 0 {
		t.Errorf("Expected 0 files, got %d", len(res.Files))
	}
}

func TestFileListReqAfterAnnounce(t *testing.T) {
	net := NewTestNetwork(t)
	defer net.Close()

	ctx := net.Context()
	client := net.NewClient()

	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Connect failed: %v", err)
	}

	// Announce files first
	fileName := "testfile.txt"
	fileSize := uint64(2048)
	hash := sha256.Sum256([]byte(fmt.Sprintf("%s%d", fileName, fileSize)))

	announce := &protocol.PeerAnnounce{
		FileCount: 1,
		Files: []protocol.FileEntry{
			{Hash: hash, Name: fileName, Size: fileSize},
		},
	}

	if err := client.SendToTracker(ctx, announce); err != nil {
		t.Fatalf("SendToTracker announce failed: %v", err)
	}

	// Request file list
	if err := client.SendToTracker(ctx, &protocol.FileListReq{}); err != nil {
		t.Fatalf("SendToTracker FileListReq failed: %v", err)
	}

	msg, err := client.ReceiveFromTracker(ctx)
	if err != nil {
		t.Fatalf("ReceiveFromTracker failed: %v", err)
	}

	res, ok := msg.(*protocol.FileListRes)
	if !ok {
		t.Fatalf("Expected FileListRes, got %T", msg)
	}

	if len(res.Files) != 1 {
		t.Errorf("Expected 1 file, got %d", len(res.Files))
	}

	if res.Files[0].Name != fileName {
		t.Errorf("Expected file name %s, got %s", fileName, res.Files[0].Name)
	}
}

func TestFileListReqMultiplePeersAnnounce(t *testing.T) {
	net := NewTestNetwork(t)
	defer net.Close()

	ctx := net.Context()

	// Peer 1 announces file1
	client1 := net.NewClient()
	if err := client1.Connect(ctx); err != nil {
		t.Fatalf("Client1 Connect failed: %v", err)
	}

	file1Name := "file1.txt"
	file1Size := uint64(1024)
	hash1 := sha256.Sum256([]byte(fmt.Sprintf("%s%d", file1Name, file1Size)))

	announce1 := &protocol.PeerAnnounce{
		FileCount: 1,
		Files:     []protocol.FileEntry{{Hash: hash1, Name: file1Name, Size: file1Size}},
	}
	if err := client1.SendToTracker(ctx, announce1); err != nil {
		t.Fatalf("Client1 SendToTracker failed: %v", err)
	}

	// Peer 2 announces file2
	client2 := net.NewClient()
	if err := client2.Connect(ctx); err != nil {
		t.Fatalf("Client2 Connect failed: %v", err)
	}

	file2Name := "file2.txt"
	file2Size := uint64(2048)
	hash2 := sha256.Sum256([]byte(fmt.Sprintf("%s%d", file2Name, file2Size)))

	announce2 := &protocol.PeerAnnounce{
		FileCount: 1,
		Files:     []protocol.FileEntry{{Hash: hash2, Name: file2Name, Size: file2Size}},
	}
	if err := client2.SendToTracker(ctx, announce2); err != nil {
		t.Fatalf("Client2 SendToTracker failed: %v", err)
	}

	// Peer 3 fetches the file list
	client3 := net.NewClient()
	if err := client3.Connect(ctx); err != nil {
		t.Fatalf("Client3 Connect failed: %v", err)
	}

	if err := client3.SendToTracker(ctx, &protocol.FileListReq{}); err != nil {
		t.Fatalf("Client3 SendToTracker FileListReq failed: %v", err)
	}

	msg, err := client3.ReceiveFromTracker(ctx)
	if err != nil {
		t.Fatalf("Client3 ReceiveFromTracker failed: %v", err)
	}

	res, ok := msg.(*protocol.FileListRes)
	if !ok {
		t.Fatalf("Expected FileListRes, got %T", msg)
	}

	if len(res.Files) != 2 {
		t.Errorf("Expected 2 files from both peers, got %d", len(res.Files))
	}
}
