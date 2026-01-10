package tracker

import (
	"slices"
	"sync"

	"github.com/rudransh-shrivastava/peer-it/internal/protocol"
	"github.com/rudransh-shrivastava/peer-it/internal/transport"
)

type Store struct {
	mu    sync.Mutex
	peers map[protocol.FileHash][]*transport.Peer
}

func NewStore() *Store {
	return &Store{
		peers: make(map[protocol.FileHash][]*transport.Peer),
	}
}

func (s *Store) AddPeer(hashes []protocol.FileHash, peer *transport.Peer) int {
	s.mu.Lock()
	defer s.mu.Unlock()

	added := 0
	for _, hash := range hashes {
		if slices.Contains(s.peers[hash], peer) {
			continue
		}
		s.peers[hash] = append(s.peers[hash], peer)
		added++
	}
	return added
}
