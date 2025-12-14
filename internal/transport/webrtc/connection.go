package webrtc

import (
	"context"
	"fmt"
	"sync"

	"github.com/pion/webrtc/v3"
	"github.com/rudransh-shrivastava/peer-it/internal/transport"
)

type connection struct {
	peerID      string
	pc          *webrtc.PeerConnection
	dc          *webrtc.DataChannel
	signaler    transport.Signaler
	recvChan    chan []byte
	isInitiator bool
	onOpen      func()
	mu          sync.Mutex
}

func newConnection(peerID string, pc *webrtc.PeerConnection, signaler transport.Signaler, isInitiator bool) *connection {
	conn := &connection{
		peerID:      peerID,
		pc:          pc,
		signaler:    signaler,
		recvChan:    make(chan []byte, 256),
		isInitiator: isInitiator,
	}

	pc.OnConnectionStateChange(func(s webrtc.PeerConnectionState) {
		if s == webrtc.PeerConnectionStateFailed || s == webrtc.PeerConnectionStateClosed {
			close(conn.recvChan)
		}
	})

	if !isInitiator {
		pc.OnDataChannel(func(dc *webrtc.DataChannel) {
			conn.setupDataChannel(dc)
		})
	}

	return conn
}

func (c *connection) createDataChannel() error {
	ordered := true
	dc, err := c.pc.CreateDataChannel("data", &webrtc.DataChannelInit{
		Ordered: &ordered,
	})
	if err != nil {
		return fmt.Errorf("failed to create data channel: %w", err)
	}
	c.setupDataChannel(dc)
	return nil
}

func (c *connection) setupDataChannel(dc *webrtc.DataChannel) {
	c.mu.Lock()
	c.dc = dc
	c.mu.Unlock()

	dc.OnOpen(func() {
		if c.onOpen != nil {
			c.onOpen()
		}
	})

	dc.OnMessage(func(msg webrtc.DataChannelMessage) {
		select {
		case c.recvChan <- msg.Data:
		default:
		}
	})

	dc.OnClose(func() {
		close(c.recvChan)
	})
}

func (c *connection) handleSignal(payload []byte) error {
	sdp := string(payload)

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.pc.RemoteDescription() == nil {
		desc := webrtc.SessionDescription{SDP: sdp}
		if c.isInitiator {
			desc.Type = webrtc.SDPTypeAnswer
		} else {
			desc.Type = webrtc.SDPTypeOffer
		}

		if err := c.pc.SetRemoteDescription(desc); err != nil {
			return fmt.Errorf("failed to set remote description: %w", err)
		}

		if !c.isInitiator {
			answer, err := c.pc.CreateAnswer(nil)
			if err != nil {
				return fmt.Errorf("failed to create answer: %w", err)
			}
			if err := c.pc.SetLocalDescription(answer); err != nil {
				return fmt.Errorf("failed to set local description: %w", err)
			}
			if err := c.signaler.SendSignal(context.Background(), c.peerID, []byte(answer.SDP)); err != nil {
				return fmt.Errorf("failed to send answer: %w", err)
			}
		}
	}

	return nil
}

func (c *connection) PeerID() string {
	return c.peerID
}

func (c *connection) Send(data []byte) error {
	c.mu.Lock()
	dc := c.dc
	c.mu.Unlock()

	if dc == nil {
		return fmt.Errorf("data channel not ready")
	}
	return dc.Send(data)
}

func (c *connection) Recv() <-chan []byte {
	return c.recvChan
}

func (c *connection) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.dc != nil {
		_ = c.dc.Close()
	}
	return c.pc.Close()
}
