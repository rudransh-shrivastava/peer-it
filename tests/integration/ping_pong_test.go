package integration

import (
	"context"
	"testing"
	"time"

	"github.com/rudransh-shrivastava/peer-it/internal/logger"
	"github.com/rudransh-shrivastava/peer-it/internal/peer"
	"github.com/rudransh-shrivastava/peer-it/internal/tracker"
)

func TestPeerTrackerPingPong(t *testing.T) {
	log := logger.NewLogger()

	// Start tracker
	srv, err := tracker.NewServer(tracker.Config{
		Addr:   ":0",
		Logger: log,
	})
	if err != nil {
		t.Fatalf("Failed to create tracker: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	serverErr := make(chan error, 1)
	go func() {
		serverErr <- srv.Start(ctx)
	}()

	// Give server time to start
	time.Sleep(50 * time.Millisecond)

	// Create peer client
	client, err := peer.NewClient(peer.Config{
		Addr:        ":0",
		Logger:      log,
		TrackerAddr: srv.Addr(),
	})
	if err != nil {
		t.Fatalf("Failed to create peer: %v", err)
	}
	defer func() { _ = client.Shutdown() }()

	// Connect to tracker
	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	// Send ping, expect pong
	if err := client.Ping(ctx); err != nil {
		t.Fatalf("Ping failed: %v", err)
	}

	// Shutdown
	cancel()
	_ = srv.Shutdown()

	// Check server exited cleanly
	select {
	case err := <-serverErr:
		if err != nil && err != context.Canceled {
			t.Errorf("Server error: %v", err)
		}
	case <-time.After(time.Second):
		t.Error("Server did not shutdown in time")
	}
}
