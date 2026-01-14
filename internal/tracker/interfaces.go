package tracker

import (
	"context"
	"net"

	"github.com/rudransh-shrivastava/peer-it/internal/protocol"
)

type Listener interface {
	Accept(ctx context.Context) (Conn, error)
	Close() error
	LocalAddr() net.Addr
}

type Conn interface {
	Close() error
	Receive(ctx context.Context) (protocol.Message, error)
	RemoteAddr() string
	Send(ctx context.Context, msg protocol.Message) error
}
