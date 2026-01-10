package peer

import (
	"context"
	"errors"
	"log/slog"

	"github.com/rudransh-shrivastava/peer-it/internal/protocol"
	"github.com/rudransh-shrivastava/peer-it/internal/transport"
)

var ErrNotConnected = errors.New("not connected to tracker")

type Client struct {
	config      Config
	logger      *slog.Logger
	trackerPeer *transport.Peer
	transport   *transport.Transport
}

func NewClient(cfg Config) (*Client, error) {
	tr, err := transport.NewTransport(cfg.Addr)
	if err != nil {
		return nil, err
	}

	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}

	return &Client{
		config:    cfg,
		logger:    logger,
		transport: tr,
	}, nil
}

func (c *Client) Addr() string {
	return c.transport.LocalAddr().String()
}

func (c *Client) Connect(ctx context.Context) error {
	c.logger.Info("Connecting to tracker", "tracker", c.config.TrackerAddr)

	peer, err := c.transport.Dial(ctx, c.config.TrackerAddr)
	if err != nil {
		c.logger.Error("Failed to connect to tracker", "error", err)
		return err
	}

	c.trackerPeer = peer
	c.logger.Info("Connected to tracker", "tracker", c.config.TrackerAddr)
	return nil
}

func (c *Client) Ping(ctx context.Context) error {
	if c.trackerPeer == nil {
		return ErrNotConnected
	}

	c.logger.Debug("Sending Ping to tracker")
	if err := c.trackerPeer.Send(ctx, &protocol.Ping{}); err != nil {
		c.logger.Error("Failed to send Ping", "error", err)
		return err
	}

	msg, err := c.trackerPeer.Receive(ctx)
	if err != nil {
		c.logger.Error("Failed to receive response", "error", err)
		return err
	}

	if _, ok := msg.(*protocol.Pong); !ok {
		c.logger.Error("Expected Pong, got different message", "type", msg.Type().String())
		return errors.New("expected Pong response")
	}

	c.logger.Info("Received Pong from tracker")
	return nil
}

func (c *Client) Shutdown() error {
	c.logger.Info("Shutting down peer client")

	if c.trackerPeer != nil {
		_ = c.trackerPeer.Close()
	}

	return c.transport.Close()
}
