package integration

import (
	"context"
	"testing"
	"time"

	"github.com/rudransh-shrivastava/peer-it/internal/logger"
	"github.com/rudransh-shrivastava/peer-it/internal/peer"
	"github.com/rudransh-shrivastava/peer-it/internal/tracker"
)

type Network struct {
	tracker *tracker.Server
	clients []*peer.Client
	cancel  context.CancelFunc
	ctx     context.Context
	t       *testing.T
}

func NewNetwork(t *testing.T) *Network {
	t.Helper()

	log := logger.NewLogger()

	srv, err := tracker.NewServer(tracker.Config{
		Addr:   ":0",
		Logger: log,
	})
	if err != nil {
		t.Fatalf("Failed to create tracker: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

	go func() {
		_ = srv.Start(ctx)
	}()

	time.Sleep(50 * time.Millisecond)

	return &Network{
		tracker: srv,
		cancel:  cancel,
		ctx:     ctx,
		t:       t,
	}
}

func (n *Network) NewClient() *peer.Client {
	n.t.Helper()

	log := logger.NewLogger()

	client, err := peer.NewClient(peer.Config{
		Addr:        ":0",
		Logger:      log,
		TrackerAddr: n.tracker.Addr(),
	})
	if err != nil {
		n.t.Fatalf("Failed to create client: %v", err)
	}

	n.clients = append(n.clients, client)
	return client
}

func (n *Network) Context() context.Context {
	return n.ctx
}

func (n *Network) Close() {
	n.cancel()
	for _, c := range n.clients {
		_ = c.Shutdown()
	}
	_ = n.tracker.Shutdown()
}
