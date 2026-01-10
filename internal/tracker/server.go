package tracker

import (
	"context"

	"github.com/rudransh-shrivastava/peer-it/internal/protocol"
	"github.com/rudransh-shrivastava/peer-it/internal/transport"
	"github.com/sirupsen/logrus"
)

type Server struct {
	config    Config
	logger    *logrus.Logger
	transport *transport.Transport
}

func NewServer(cfg Config) (*Server, error) {
	tr, err := transport.NewTransport(cfg.Addr)
	if err != nil {
		return nil, err
	}

	logger := cfg.Logger
	if logger == nil {
		logger = logrus.New()
	}

	return &Server{
		config:    cfg,
		logger:    logger,
		transport: tr,
	}, nil
}

func (s *Server) Addr() string {
	return s.transport.LocalAddr().String()
}

func (s *Server) Shutdown() error {
	s.logger.Info("Shutting down tracker server")
	return s.transport.Close()
}

func (s *Server) Start(ctx context.Context) error {
	s.logger.WithField("addr", s.Addr()).Info("Tracker server started")

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			peer, err := s.transport.Accept(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return ctx.Err()
				}
				s.logger.WithError(err).Error("Failed to accept connection")
				continue
			}

			go s.handlePeer(ctx, peer)
		}
	}
}

func (s *Server) handlePeer(ctx context.Context, peer *transport.Peer) {
	remoteAddr := peer.RemoteAddr()
	s.logger.WithField("peer", remoteAddr).Info("Peer connected")
	defer func() {
		_ = peer.Close()
		s.logger.WithField("peer", remoteAddr).Info("Peer disconnected")
	}()

	for {
		select {
		case <-ctx.Done():
			return
		default:
			msg, err := peer.Receive(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				s.logger.WithError(err).Debug("Failed to receive message")
				return
			}

			s.handleMessage(ctx, peer, msg)
		}
	}
}

func (s *Server) handleMessage(ctx context.Context, peer *transport.Peer, msg protocol.Message) {
	switch msg.Type() {
	case protocol.MsgPing:
		s.logger.WithField("peer", peer.RemoteAddr()).Debug("Received Ping, sending Pong")
		if err := peer.Send(ctx, &protocol.Pong{}); err != nil {
			s.logger.WithError(err).Error("Failed to send Pong")
		}
	default:
		s.logger.WithField("type", msg.Type().String()).Warn("Unhandled message type")
	}
}
