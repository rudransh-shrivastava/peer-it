package peer

import (
	"context"
	"log/slog"

	"github.com/rudransh-shrivastava/peer-it/internal/protocol"
	"github.com/rudransh-shrivastava/peer-it/internal/transport"
)

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

func (c *Client) SendToTracker(ctx context.Context, msg protocol.Message) error {
	c.logger.Debug("Sending to tracker", "type", msg.Type().String())
	return c.trackerPeer.Send(ctx, msg)
}

func (c *Client) ReceiveFromTracker(ctx context.Context) (protocol.Message, error) {
	msg, err := c.trackerPeer.Receive(ctx)
	if err != nil {
		return nil, err
	}
	c.logger.Debug("Received from tracker", "type", msg.Type().String())
	return msg, nil
}

func (c *Client) Shutdown() error {
	c.logger.Info("Shutting down peer client")

	if c.trackerPeer != nil {
		_ = c.trackerPeer.Close()
	}

	return c.transport.Close()
}
