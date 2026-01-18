package discovery

import (
	"errors"
	"net"
	"sync"
)

var errSocketClosed = errors.New("socket closed")

type mockAddr struct {
	addr string
}

func (a mockAddr) Network() string { return "udp" }
func (a mockAddr) String() string  { return a.addr }

type mockSocket struct {
	mu        sync.Mutex
	closed    bool
	closeCh   chan struct{}
	localAddr net.Addr
	readIdx   int
	recvAddrs []net.Addr
	recvData  [][]byte
	sentData  [][]byte
}

func newMockSocket() *mockSocket {
	return &mockSocket{
		closeCh:   make(chan struct{}),
		localAddr: mockAddr{addr: "127.0.0.1:9999"},
	}
}

func (s *mockSocket) Close() error {
	s.mu.Lock()
	if !s.closed {
		s.closed = true
		close(s.closeCh)
	}
	s.mu.Unlock()
	return nil
}

func (s *mockSocket) LocalAddr() net.Addr {
	return s.localAddr
}

func (s *mockSocket) ReadFrom(b []byte) (int, net.Addr, error) {
	s.mu.Lock()

	if s.closed {
		s.mu.Unlock()
		return 0, nil, errSocketClosed
	}

	if s.readIdx < len(s.recvData) {
		data := s.recvData[s.readIdx]
		addr := s.recvAddrs[s.readIdx]
		s.readIdx++
		s.mu.Unlock()

		n := copy(b, data)
		return n, addr, nil
	}

	s.mu.Unlock()

	<-s.closeCh
	return 0, nil, errSocketClosed
}

func (s *mockSocket) WriteTo(b []byte, addr net.Addr) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return 0, errSocketClosed
	}

	data := make([]byte, len(b))
	copy(data, b)
	s.sentData = append(s.sentData, data)
	return len(b), nil
}

func (s *mockSocket) queueMessage(data []byte, from string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.recvData = append(s.recvData, data)
	s.recvAddrs = append(s.recvAddrs, mockAddr{addr: from})
}
