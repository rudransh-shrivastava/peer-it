package transport

import (
	"context"
	"io"
	"sync"

	"github.com/quic-go/quic-go"
	"github.com/rudransh-shrivastava/peer-it/internal/protocol"
)

type Peer struct {
	codec         *protocol.Codec
	conn          *quic.Conn
	controlStream *quic.Stream
	mu            sync.Mutex
}

func NewPeer(conn *quic.Conn) *Peer {
	return &Peer{
		codec: protocol.NewCodec(),
		conn:  conn,
	}
}

func (p *Peer) AcceptDataStream(ctx context.Context) (*quic.Stream, error) {
	return p.conn.AcceptStream(ctx)
}

func (p *Peer) Close() error {
	if p.controlStream != nil {
		_ = p.controlStream.Close()
	}
	return p.conn.CloseWithError(0, "")
}

func (p *Peer) OpenDataStream(ctx context.Context) (*quic.Stream, error) {
	return p.conn.OpenStreamSync(ctx)
}

func (p *Peer) Receive(ctx context.Context) (protocol.Message, error) {
	stream, err := p.acceptControlStream(ctx)
	if err != nil {
		return nil, err
	}

	return p.codec.Decode(stream)
}

func (p *Peer) ReceiveFromStream(stream io.Reader) (protocol.Message, error) {
	return p.codec.Decode(stream)
}

func (p *Peer) RemoteAddr() string {
	return p.conn.RemoteAddr().String()
}

func (p *Peer) Send(ctx context.Context, msg protocol.Message) error {
	stream, err := p.getControlStream(ctx)
	if err != nil {
		return err
	}

	return p.codec.Encode(stream, msg)
}

func (p *Peer) SendOnStream(stream io.Writer, msg protocol.Message) error {
	return p.codec.Encode(stream, msg)
}

func (p *Peer) acceptControlStream(ctx context.Context) (*quic.Stream, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.controlStream != nil {
		return p.controlStream, nil
	}

	stream, err := p.conn.AcceptStream(ctx)
	if err != nil {
		return nil, err
	}
	p.controlStream = stream
	return stream, nil
}

func (p *Peer) getControlStream(ctx context.Context) (*quic.Stream, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.controlStream != nil {
		return p.controlStream, nil
	}

	stream, err := p.conn.OpenStreamSync(ctx)
	if err != nil {
		return nil, err
	}
	p.controlStream = stream
	return stream, nil
}
