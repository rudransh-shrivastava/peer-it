package discovery

import "net"

type Socket interface {
	Close() error
	LocalAddr() net.Addr
	ReadFrom(b []byte) (int, net.Addr, error)
	WriteTo(b []byte, addr net.Addr) (int, error)
}
