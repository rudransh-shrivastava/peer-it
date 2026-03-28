package tracker

import (
	"context"
	"log/slog"
	"net"

	"github.com/rudransh-shrivastava/peer-it/internal/protocol"
	"github.com/rudransh-shrivastava/peer-it/internal/transport"
)

type Server struct {
	config   Config
	listener Listener
	logger   *slog.Logger
	store    *Store
}

func NewServer(cfg Config) (*Server, error) {
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}

	listener := cfg.Listener
	if listener == nil {
		tr, err := transport.NewTransport(cfg.Addr)
		if err != nil {
			return nil, err
		}
		listener = &transportAdapter{tr}
	}

	return &Server{
		config:   cfg,
		listener: listener,
		logger:   logger,
		store:    NewStore(),
	}, nil
}

func (s *Server) Addr() string {
	return s.listener.LocalAddr().String()
}

func (s *Server) Shutdown() error {
	s.logger.Info("Shutting down tracker server")
	return s.listener.Close()
}

func (s *Server) Start(ctx context.Context) error {
	s.logger.Info("Tracker server started", "addr", s.Addr())

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			conn, err := s.listener.Accept(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return ctx.Err()
				}
				s.logger.Error("Failed to accept connection", "error", err)
				continue
			}

			go s.handleConn(ctx, conn)
		}
	}
}

func (s *Server) handleConn(ctx context.Context, conn Conn) {
	remoteAddr := conn.RemoteAddr()
	s.logger.Info("Peer connected", "peer", remoteAddr)
	defer func() {
		_ = conn.Close()
		s.logger.Info("Peer disconnected", "peer", remoteAddr)
	}()

	for {
		select {
		case <-ctx.Done():
			return
		default:
			msg, err := conn.Receive(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				s.logger.Debug("Failed to receive message", "error", err)
				return
			}

			s.handleMessage(ctx, conn, msg)
		}
	}
}

func (s *Server) handleMessage(ctx context.Context, conn Conn, msg protocol.Message) {
	switch msg.Type() {
	case protocol.MsgFileListReq:
		s.logger.Debug("Received FileListReq, sending file list to peer", "peer", conn.RemoteAddr())
		s.handleFileListReqMessage(ctx, conn)
	case protocol.MsgHolePunchReq:
		reqMsg, _ := msg.(*protocol.HolePunchReq)
		s.logger.Debug("Received MsgHolePunchReq, sending request to target", "target_node_id", reqMsg.TargetNodeID)
		// TODO(rudransh-shrivastava): After implementing STUN probes
	case protocol.MsgPeerAnnounce:
		s.logger.Debug("Received PeerAnnounce, adding peer to database", "peer", conn.RemoteAddr())
		announceMsg, _ := msg.(*protocol.PeerAnnounce)
		s.handlePeerAnnounceMessage(conn, *announceMsg)
	case protocol.MsgPeerListReq:
		s.logger.Debug("Received PeerListReq", "peer", conn.RemoteAddr())
		reqMsg, _ := msg.(*protocol.PeerListReq)
		s.handlePeerListReqMessage(ctx, conn, reqMsg.FileHash)
	case protocol.MsgPing:
		s.logger.Debug("Received Ping, sending Pong", "peer", conn.RemoteAddr())
		s.handlePingMessage(ctx, conn)
	default:
		s.logger.Warn("Unhandled message type", "type", msg.Type().String())
	}
}

func (s *Server) handleFileListReqMessage(ctx context.Context, conn Conn) {
	files := s.store.ListFiles()

	res := protocol.FileListRes{Files: files}
	if err := conn.Send(ctx, &res); err != nil {
		s.logger.Error("Failed to send FileListRes", "peer", conn.RemoteAddr(), "error", err)
	}
}

func (s *Server) handlePeerListReqMessage(ctx context.Context, conn Conn, fileHash protocol.FileHash) {
	peers := s.store.GetPeers(fileHash)

	peerInfos := make([]protocol.PeerInfo, 0, len(peers))
	for _, p := range peers {
		peerInfos = append(peerInfos, protocol.PeerInfo{NodeID: p})
	}

	res := protocol.PeerListRes{FileHash: fileHash, Peers: peerInfos}
	if err := conn.Send(ctx, &res); err != nil {
		s.logger.Error("Failed to send PeerListRes", "peer", conn.RemoteAddr(), "error", err)
	}
}

func (s *Server) handlePeerAnnounceMessage(conn Conn, msg protocol.PeerAnnounce) {
	if msg.FileCount != uint16(len(msg.Files)) {
		s.logger.Debug("Received malformed PeerAnnounce, files count does not equal number of files",
			"peer", conn.RemoteAddr(),
		)
		return
	}
	for _, file := range msg.Files {
		hash := generateHash(&file)
		if hash != file.Hash {
			s.logger.Debug("Received malformed PeerAnnounce, invalid file hash", "peer", conn.RemoteAddr())
			return
		}
	}

	peerID := generatePeerID()
	addedFiles := s.store.AddFiles(msg.Files)
	s.logger.Debug("Added files", "peer", conn.RemoteAddr(), "count", addedFiles)
	addedPeerToFiles := s.store.AddPeer(msg.Files, peerID)
	s.logger.Debug("Added peer to files", "peer", conn.RemoteAddr(), "count", addedPeerToFiles)
}

func (s *Server) handlePingMessage(ctx context.Context, conn Conn) {
	if err := conn.Send(ctx, &protocol.Pong{}); err != nil {
		s.logger.Error("Failed to send Pong", "peer", conn.RemoteAddr(), "error", err)
	}
}

type transportAdapter struct {
	tr *transport.Transport
}

func (a *transportAdapter) Accept(ctx context.Context) (Conn, error) {
	return a.tr.Accept(ctx)
}

func (a *transportAdapter) Close() error {
	return a.tr.Close()
}

func (a *transportAdapter) LocalAddr() net.Addr {
	return a.tr.LocalAddr()
}
