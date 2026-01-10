package tracker

import (
	"context"
	"log/slog"

	"github.com/rudransh-shrivastava/peer-it/internal/protocol"
	"github.com/rudransh-shrivastava/peer-it/internal/transport"
)

type Server struct {
	config    Config
	logger    *slog.Logger
	transport *transport.Transport
}

func NewServer(cfg Config) (*Server, error) {
	tr, err := transport.NewTransport(cfg.Addr)
	if err != nil {
		return nil, err
	}

	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
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
	s.logger.Info("Tracker server started", "addr", s.Addr())

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
				s.logger.Error("Failed to accept connection", "error", err)
				continue
			}

			go s.handlePeer(ctx, peer)
		}
	}
}

func (s *Server) handlePeer(ctx context.Context, peer *transport.Peer) {
	remoteAddr := peer.RemoteAddr()
	s.logger.Info("Peer connected", "peer", remoteAddr)
	defer func() {
		_ = peer.Close()
		s.logger.Info("Peer disconnected", "peer", remoteAddr)
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
				s.logger.Debug("Failed to receive message", "error", err)
				return
			}

			s.handleMessage(ctx, peer, msg)
		}
	}
}

func (s *Server) handleMessage(ctx context.Context, peer *transport.Peer, msg protocol.Message) {
	switch msg.Type() {
	case protocol.MsgPing:
		s.logger.Debug("Received Ping, sending Pong", "peer", peer.RemoteAddr())
		if err := peer.Send(ctx, &protocol.Pong{}); err != nil {
			s.logger.Error("Failed to send Pong", "error", err)
		}
	default:
		s.logger.Warn("Unhandled message type", "type", msg.Type().String())
	}
}
