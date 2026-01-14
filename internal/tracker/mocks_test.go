package tracker

import (
	"context"
	"errors"
	"net"

	"github.com/rudransh-shrivastava/peer-it/internal/protocol"
)

type mockAddr struct {
	addr string
}

func (a mockAddr) Network() string { return "mock" }
func (a mockAddr) String() string  { return a.addr }

type mockListener struct {
	acceptCh chan Conn
	closeCh  chan struct{}
	closed   bool
}

func newMockListener() *mockListener {
	return &mockListener{
		acceptCh: make(chan Conn, 10),
		closeCh:  make(chan struct{}),
	}
}

func (l *mockListener) Accept(ctx context.Context) (Conn, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-l.closeCh:
		return nil, errors.New("listener closed")
	case conn := <-l.acceptCh:
		return conn, nil
	}
}

func (l *mockListener) Close() error {
	if !l.closed {
		l.closed = true
		close(l.closeCh)
	}
	return nil
}

func (l *mockListener) LocalAddr() net.Addr {
	return mockAddr{addr: "mock://test:0"}
}

type mockConn struct {
	addr     string
	recvCh   chan protocol.Message
	recvErr  error
	sentMsgs []protocol.Message
	sendErr  error
	closed   bool
}

func newMockConn(addr string) *mockConn {
	return &mockConn{
		addr:   addr,
		recvCh: make(chan protocol.Message, 10),
	}
}

func (c *mockConn) Close() error {
	c.closed = true
	return nil
}

func (c *mockConn) Receive(ctx context.Context) (protocol.Message, error) {
	if c.recvErr != nil {
		return nil, c.recvErr
	}
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case msg := <-c.recvCh:
		return msg, nil
	}
}

func (c *mockConn) RemoteAddr() string {
	return c.addr
}

func (c *mockConn) Send(ctx context.Context, msg protocol.Message) error {
	if c.sendErr != nil {
		return c.sendErr
	}
	c.sentMsgs = append(c.sentMsgs, msg)
	return nil
}
