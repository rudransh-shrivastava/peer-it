package transport

import (
	"context"
	"crypto/tls"
	"net"

	"github.com/quic-go/quic-go"
)

type Transport struct {
	conn      *net.UDPConn
	listener  *quic.Listener
	quicTr    *quic.Transport
	tlsConfig *tls.Config
}

func NewTransport(addr string) (*Transport, error) {
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return nil, err
	}

	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return nil, err
	}

	tlsConfig, err := DefaultTLSConfig()
	if err != nil {
		_ = conn.Close()
		return nil, err
	}

	return &Transport{
		conn:      conn,
		quicTr:    &quic.Transport{Conn: conn},
		tlsConfig: tlsConfig,
	}, nil
}

func (t *Transport) Accept(ctx context.Context) (*Peer, error) {
	if t.listener == nil {
		if _, err := t.Listen(ctx); err != nil {
			return nil, err
		}
	}

	conn, err := t.listener.Accept(ctx)
	if err != nil {
		return nil, err
	}

	return NewPeer(conn), nil
}

func (t *Transport) Close() error {
	if t.listener != nil {
		_ = t.listener.Close()
	}
	return t.quicTr.Close()
}

func (t *Transport) Dial(ctx context.Context, addr string) (*Peer, error) {
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return nil, err
	}

	conn, err := t.quicTr.Dial(ctx, udpAddr, t.tlsConfig, DefaultQUICConfig())
	if err != nil {
		return nil, err
	}

	return NewPeer(conn), nil
}

func (t *Transport) Listen(ctx context.Context) (*quic.Listener, error) {
	listener, err := t.quicTr.Listen(t.tlsConfig, DefaultQUICConfig())
	if err != nil {
		return nil, err
	}
	t.listener = listener
	return listener, nil
}

func (t *Transport) LocalAddr() net.Addr {
	return t.conn.LocalAddr()
}

func (t *Transport) ReadNonQUIC(ctx context.Context, buf []byte) (int, net.Addr, error) {
	return t.quicTr.ReadNonQUICPacket(ctx, buf)
}

func (t *Transport) WriteNonQUIC(addr net.Addr, data []byte) (int, error) {
	return t.conn.WriteTo(data, addr)
}
