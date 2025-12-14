// Package webrtc implements WebRTC transport.
package webrtc

import (
	"context"
	"fmt"
	"sync"

	"github.com/pion/webrtc/v3"
	"github.com/rudransh-shrivastava/peer-it/internal/transport"
)

type webrtcTransport struct {
	config      webrtc.Configuration
	signaler    transport.Signaler
	connections map[string]*connection
	incoming    chan transport.Conn
	mu          sync.RWMutex
}

// New creates a WebRTC transport.
func New(signaler transport.Signaler, stunServers []string) transport.Transport {
	iceServers := make([]webrtc.ICEServer, 0, len(stunServers))
	for _, server := range stunServers {
		iceServers = append(iceServers, webrtc.ICEServer{URLs: []string{server}})
	}

	return &webrtcTransport{
		config: webrtc.Configuration{
			ICEServers:         iceServers,
			ICETransportPolicy: webrtc.ICETransportPolicyAll,
		},
		signaler:    signaler,
		connections: make(map[string]*connection),
		incoming:    make(chan transport.Conn, 16),
	}
}

func (t *webrtcTransport) Connect(ctx context.Context, peerID string, _ transport.ConnectionMetadata) (transport.Conn, error) {
	pc, err := webrtc.NewPeerConnection(t.config)
	if err != nil {
		return nil, fmt.Errorf("failed to create peer connection: %w", err)
	}

	conn := newConnection(peerID, pc, t.signaler, true)

	t.mu.Lock()
	t.connections[peerID] = conn
	t.mu.Unlock()

	if err := conn.createDataChannel(); err != nil {
		return nil, err
	}

	offer, err := pc.CreateOffer(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create offer: %w", err)
	}

	if err := pc.SetLocalDescription(offer); err != nil {
		return nil, fmt.Errorf("failed to set local description: %w", err)
	}

	if err := t.signaler.SendSignal(ctx, peerID, []byte(offer.SDP)); err != nil {
		return nil, fmt.Errorf("failed to send offer: %w", err)
	}

	return conn, nil
}

func (t *webrtcTransport) Accept() <-chan transport.Conn {
	return t.incoming
}

func (t *webrtcTransport) HandleSignal(signal transport.Signal) error {
	t.mu.RLock()
	conn, exists := t.connections[signal.PeerID]
	t.mu.RUnlock()

	if !exists {
		pc, err := webrtc.NewPeerConnection(t.config)
		if err != nil {
			return fmt.Errorf("failed to create peer connection: %w", err)
		}

		conn = newConnection(signal.PeerID, pc, t.signaler, false)
		conn.onOpen = func() {
			t.incoming <- conn
		}

		t.mu.Lock()
		t.connections[signal.PeerID] = conn
		t.mu.Unlock()
	}

	return conn.handleSignal(signal.Payload)
}

func (t *webrtcTransport) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	for _, conn := range t.connections {
		_ = conn.Close()
	}
	t.connections = make(map[string]*connection)
	close(t.incoming)
	return nil
}
