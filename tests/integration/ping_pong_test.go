package integration

import (
	"testing"

	"github.com/rudransh-shrivastava/peer-it/internal/protocol"
)

func TestPeerTrackerPingPong(t *testing.T) {
	net := NewNetwork(t)
	defer net.Close()

	client := net.NewClient()
	ctx := net.Context()

	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Connect failed: %v", err)
	}

	if err := client.SendToTracker(ctx, &protocol.Ping{}); err != nil {
		t.Fatalf("SendToTracker failed: %v", err)
	}

	msg, err := client.ReceiveFromTracker(ctx)
	if err != nil {
		t.Fatalf("ReceiveFromTracker failed: %v", err)
	}

	if _, ok := msg.(*protocol.Pong); !ok {
		t.Errorf("Expected Pong, got %T", msg)
	}
}
