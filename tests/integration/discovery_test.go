package integration

import (
	"context"
	"testing"
	"time"

	"github.com/rudransh-shrivastava/peer-it/internal/discovery"
	"github.com/rudransh-shrivastava/peer-it/internal/protocol"
)

func TestDiscoveryLocalhost(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	nodeID1 := [protocol.NodeIDSize]byte{0x01}
	nodeID2 := [protocol.NodeIDSize]byte{0x02}

	discovered := make(chan discovery.PeerInfo, 1)

	d1, err := discovery.New(discovery.Config{
		Interval: 100 * time.Millisecond,
		NodeID:   nodeID1,
		Port:     5001,
	})
	if err != nil {
		t.Fatalf("New d1 failed: %v", err)
	}

	d2, err := discovery.New(discovery.Config{
		Interval: 100 * time.Millisecond,
		NodeID:   nodeID2,
		OnDiscover: func(p discovery.PeerInfo) {
			discovered <- p
		},
		Port: 5002,
	})
	if err != nil {
		t.Fatalf("New d2 failed: %v", err)
	}

	if err := d1.Start(ctx); err != nil {
		t.Fatalf("Start d1 failed: %v", err)
	}
	defer func() { _ = d1.Stop() }()

	if err := d2.Start(ctx); err != nil {
		t.Fatalf("Start d2 failed: %v", err)
	}
	defer func() { _ = d2.Stop() }()

	select {
	case p := <-discovered:
		if p.NodeID != nodeID1 {
			t.Errorf("Expected NodeID1, got different NodeID")
		}
	case <-time.After(2 * time.Second):
		t.Error("Timeout waiting for peer discovery")
	}
}
