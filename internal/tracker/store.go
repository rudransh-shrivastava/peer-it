package tracker

import (
	"slices"
	"sync"

	"github.com/rudransh-shrivastava/peer-it/internal/protocol"
)

type file struct {
	metadata *protocol.FileEntry
	peers    []Conn
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

func (s *Store) AddPeer(files []protocol.FileEntry, peer Conn) int {
	s.mu.Lock()
	defer s.mu.Unlock()

	added := 0
	for _, file := range files {
		if slices.Contains(s.files[file.Hash].peers, peer) {
			continue
		}
		s.files[file.Hash].peers = append(s.files[file.Hash].peers, peer)
		added++
	}
	return added
}

func (s *Store) AddFiles(files []protocol.FileEntry) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	added := 0

	for _, f := range files {
		if _, exists := s.files[f.Hash]; exists {
			continue
		}
		s.files[f.Hash] = &file{
			metadata: &f,
			peers:    []Conn{},
		}
		added++
	}
	return added
}

func (s *Store) ListFiles() []protocol.FileEntry {
	s.mu.Lock()
	defer s.mu.Unlock()

	files := []protocol.FileEntry{}
	for _, file := range s.files {
		files = append(files, *file.metadata)
	}
	return files
}

func (s *Store) GetPeers(hash protocol.FileHash) []Conn {
	s.mu.Lock()
	defer s.mu.Unlock()

	f, exists := s.files[hash]
	if !exists {
		return nil
	}

	peers := make([]Conn, len(f.peers))
	copy(peers, f.peers)
	return peers
}
