package peer

import (
	"context"
	"errors"

	"github.com/rudransh-shrivastava/peer-it/internal/protocol"
	"github.com/rudransh-shrivastava/peer-it/internal/transport"
	"github.com/sirupsen/logrus"
)

var ErrNotConnected = errors.New("not connected to tracker")

type Client struct {
	config      Config
	logger      *logrus.Logger
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
		logger = logrus.New()
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
	c.logger.WithField("tracker", c.config.TrackerAddr).Info("Connecting to tracker")

	peer, err := c.transport.Dial(ctx, c.config.TrackerAddr)
	if err != nil {
		c.logger.WithError(err).Error("Failed to connect to tracker")
		return err
	}

	c.trackerPeer = peer
	c.logger.WithField("tracker", c.config.TrackerAddr).Info("Connected to tracker")
	return nil
}

func (c *Client) Ping(ctx context.Context) error {
	if c.trackerPeer == nil {
		return ErrNotConnected
	}

	c.logger.Debug("Sending Ping to tracker")
	if err := c.trackerPeer.Send(ctx, &protocol.Ping{}); err != nil {
		c.logger.WithError(err).Error("Failed to send Ping")
		return err
	}

	msg, err := c.trackerPeer.Receive(ctx)
	if err != nil {
		c.logger.WithError(err).Error("Failed to receive response")
		return err
	}

	if _, ok := msg.(*protocol.Pong); !ok {
		c.logger.WithField("type", msg.Type().String()).Error("Expected Pong, got different message")
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
