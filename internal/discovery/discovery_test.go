package discovery

import (
	"bytes"
	"context"
	"encoding/gob"
	"testing"
	"time"

	"github.com/rudransh-shrivastava/peer-it/internal/protocol"
)

func TestNewDiscovery(t *testing.T) {
	d, err := New(Config{
		NodeID: [protocol.NodeIDSize]byte{0x01},
		Port:   5000,
	})
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	if d.cfg.MulticastAddr != DefaultMulticastAddr {
		t.Errorf("Expected default multicast addr, got %s", d.cfg.MulticastAddr)
	}

	if d.cfg.Interval != DefaultInterval {
		t.Errorf("Expected default interval, got %v", d.cfg.Interval)
	}
}

func TestDiscoveryStartStop(t *testing.T) {
	socket := newMockSocket()

	d, err := New(Config{
		NodeID:   [protocol.NodeIDSize]byte{0x01},
		Port:     5000,
		Socket:   socket,
		Interval: 100 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	if err := d.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	time.Sleep(150 * time.Millisecond)

	if err := d.Stop(); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}

	if !socket.closed {
		t.Error("Expected socket to be closed")
	}

	if len(socket.sentData) == 0 {
		t.Error("Expected at least one announcement to be sent")
	}
}

func TestDiscoverySendsAnnouncement(t *testing.T) {
	socket := newMockSocket()
	nodeID := [protocol.NodeIDSize]byte{0x01, 0x02, 0x03}

	d, err := New(Config{
		NodeID:   nodeID,
		Port:     6000,
		Socket:   socket,
		Interval: 50 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	_ = d.Start(ctx)
	time.Sleep(100 * time.Millisecond)
	_ = d.Stop()

	if len(socket.sentData) == 0 {
		t.Fatal("No announcements sent")
	}

	var msg protocol.Discovery
	if err := gob.NewDecoder(bytes.NewReader(socket.sentData[0])).Decode(&msg); err != nil {
		t.Fatalf("Failed to decode announcement: %v", err)
	}

	if msg.NodeID != nodeID {
		t.Error("NodeID mismatch")
	}

	if msg.Port != 6000 {
		t.Errorf("Expected port 6000, got %d", msg.Port)
	}
}

func TestDiscoveryIgnoresSelf(t *testing.T) {
	socket := newMockSocket()
	nodeID := [protocol.NodeIDSize]byte{0x01, 0x02, 0x03}

	var buf bytes.Buffer
	_ = gob.NewEncoder(&buf).Encode(&protocol.Discovery{NodeID: nodeID, Port: 5000})
	socket.queueMessage(buf.Bytes(), "192.168.1.100:9999")

	discovered := make(chan PeerInfo, 1)
	d, _ := New(Config{
		NodeID:   nodeID,
		Port:     5000,
		Socket:   socket,
		Interval: time.Hour,
		OnDiscover: func(p PeerInfo) {
			discovered <- p
		},
	})

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_ = d.Start(ctx)

	select {
	case <-discovered:
		t.Error("Should not discover self")
	case <-time.After(50 * time.Millisecond):
	}

	_ = d.Stop()
}

func TestDiscoveryReceivesPeer(t *testing.T) {
	socket := newMockSocket()
	myNodeID := [protocol.NodeIDSize]byte{0x01}
	peerNodeID := [protocol.NodeIDSize]byte{0x02}

	var buf bytes.Buffer
	_ = gob.NewEncoder(&buf).Encode(&protocol.Discovery{NodeID: peerNodeID, Port: 7000})
	socket.queueMessage(buf.Bytes(), "192.168.1.50:9999")

	discovered := make(chan PeerInfo, 1)
	d, _ := New(Config{
		NodeID:   myNodeID,
		Port:     5000,
		Socket:   socket,
		Interval: time.Hour,
		OnDiscover: func(p PeerInfo) {
			discovered <- p
		},
	})

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	_ = d.Start(ctx)

	select {
	case p := <-discovered:
		if p.NodeID != peerNodeID {
			t.Error("NodeID mismatch")
		}
		if p.Addr.String() != "192.168.1.50:9999" {
			t.Errorf("Addr mismatch: %s", p.Addr.String())
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Timeout waiting for peer discovery")
	}

	_ = d.Stop()
}
