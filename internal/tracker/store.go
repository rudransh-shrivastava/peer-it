package tracker

import (
	"slices"
	"sync"

	"github.com/rudransh-shrivastava/peer-it/internal/protocol"
	"github.com/rudransh-shrivastava/peer-it/internal/transport"
)

type file struct {
	metadata *protocol.FileEntry
	peers    []*transport.Peer
}

type Store struct {
	mu    sync.Mutex
	files map[protocol.FileHash]*file
}

func NewStore() *Store {
	return &Store{
		files: make(map[protocol.FileHash]*file),
	}
}

func (s *Store) AddPeer(files *[]protocol.FileEntry, peer *transport.Peer) int {
	s.mu.Lock()
	defer s.mu.Unlock()

	added := 0
	for _, file := range *files {
		if slices.Contains(s.files[file.Hash].peers, peer) {
			continue
		}
		s.files[file.Hash].peers = append(s.files[file.Hash].peers, peer)
		added++
	}
	return added
}

func (s *Store) AddFiles(files *[]protocol.FileEntry) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	added := 0

	for _, f := range *files {
		if _, exists := s.files[f.Hash]; exists {
			continue
		}
		s.files[f.Hash] = &file{
			metadata: &f,
			peers:    []*transport.Peer{},
		}
		added++
	}
	return added
}
