package tracker

import (
	"context"
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

	hash1 := protocol.FileHash{0x01}
	announce := &protocol.PeerAnnounce{
		FileCount:  1,
		FileHashes: []protocol.FileHash{hash1},
	}

	if err := client.Send(ctx, announce); err != nil {
		t.Fatalf("Send PeerAnnounce failed: %v", err)
	}

	time.Sleep(50 * time.Millisecond)

	if len(srv.store.peers[hash1]) != 1 {
		t.Errorf("Expected 1 peer in store, got %d", len(srv.store.peers[hash1]))
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
