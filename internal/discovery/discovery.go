package discovery

import (
	"bytes"
	"context"
	"encoding/gob"
	"log/slog"
	"net"
	"time"

	"github.com/rudransh-shrivastava/peer-it/internal/logger"
	"github.com/rudransh-shrivastava/peer-it/internal/protocol"
	"golang.org/x/net/ipv4"
)

const (
	DefaultInterval      = 5 * time.Second
	DefaultMulticastAddr = "224.0.0.251:9999"
	maxMessageSize       = 1024
)

type PeerInfo struct {
	Addr   net.Addr
	NodeID [protocol.NodeIDSize]byte
}

type Config struct {
	Interval      time.Duration
	Logger        *slog.Logger
	MulticastAddr string
	NodeID        [protocol.NodeIDSize]byte
	OnDiscover    func(PeerInfo)
	Port          uint16
	Socket        Socket
}

type Discovery struct {
	cfg           Config
	done          chan struct{}
	logger        *slog.Logger
	multicastAddr *net.UDPAddr
	socket        Socket
}

func New(cfg Config) (*Discovery, error) {
	if cfg.MulticastAddr == "" {
		cfg.MulticastAddr = DefaultMulticastAddr
	}
	if cfg.Interval == 0 {
		cfg.Interval = DefaultInterval
	}

	log := cfg.Logger
	if log == nil {
		log = logger.NewLogger()
	}

	addr, err := net.ResolveUDPAddr("udp4", cfg.MulticastAddr)
	if err != nil {
		return nil, err
	}

	return &Discovery{
		cfg:           cfg,
		done:          make(chan struct{}),
		logger:        log,
		multicastAddr: addr,
	}, nil
}

func (d *Discovery) Start(ctx context.Context) error {
	if d.cfg.Socket != nil {
		d.socket = d.cfg.Socket
	} else {
		conn, err := net.ListenMulticastUDP("udp4", nil, d.multicastAddr)
		if err != nil {
			return err
		}

		p := ipv4.NewPacketConn(conn)
		if err := p.SetMulticastLoopback(true); err != nil {
			_ = conn.Close()
			return err
		}

		d.socket = conn
	}

	d.logger.Info("Discovery started", "addr", d.multicastAddr.String())

	go d.listenLoop(ctx)
	go d.announceLoop(ctx)

	return nil
}

func (d *Discovery) Stop() error {
	close(d.done)
	if d.socket != nil {
		return d.socket.Close()
	}
	return nil
}

func (d *Discovery) listenLoop(ctx context.Context) {
	buf := make([]byte, maxMessageSize)

	for {
		select {
		case <-ctx.Done():
			return
		case <-d.done:
			return
		default:
			n, addr, err := d.socket.ReadFrom(buf)
			if err != nil {
				select {
				case <-d.done:
					return
				default:
					continue
				}
			}

			var msg protocol.Discovery
			if err := gob.NewDecoder(bytes.NewReader(buf[:n])).Decode(&msg); err != nil {
				d.logger.Debug("Failed to decode discovery message", "error", err)
				continue
			}

			if msg.NodeID == d.cfg.NodeID {
				continue
			}

			d.logger.Debug("Discovered peer", "nodeID", msg.NodeID[:4], "addr", addr)

			if d.cfg.OnDiscover != nil {
				d.cfg.OnDiscover(PeerInfo{
					Addr:   addr,
					NodeID: msg.NodeID,
				})
			}
		}
	}
}

func (d *Discovery) announceLoop(ctx context.Context) {
	ticker := time.NewTicker(d.cfg.Interval)
	defer ticker.Stop()

	d.sendAnnouncement()

	for {
		select {
		case <-ctx.Done():
			return
		case <-d.done:
			return
		case <-ticker.C:
			d.sendAnnouncement()
		}
	}
}

func (d *Discovery) sendAnnouncement() {
	msg := protocol.Discovery{
		NodeID: d.cfg.NodeID,
		Port:   d.cfg.Port,
	}

	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(&msg); err != nil {
		d.logger.Error("Failed to encode discovery message", "error", err)
		return
	}

	if _, err := d.socket.WriteTo(buf.Bytes(), d.multicastAddr); err != nil {
		d.logger.Debug("Failed to send discovery message", "error", err)
	}
}
