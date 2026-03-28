package tracker

import (
	"slices"
	"sync"

	"github.com/rudransh-shrivastava/peer-it/internal/protocol"
)

type file struct {
	metadata *protocol.FileEntry
	peers    []protocol.NodeID
}

type Store struct {
	mu    sync.Mutex
	files map[protocol.FileHash]*file
	peers map[protocol.NodeID][]Conn
}

func NewStore() *Store {
	return &Store{
		files: make(map[protocol.FileHash]*file),
		peers: make(map[protocol.NodeID][]Conn),
	}
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
			peers:    []protocol.NodeID{},
		}
		added++
	}
	return added
}

func (s *Store) AddPeer(files []protocol.FileEntry, peerID protocol.NodeID, conn Conn) int {
	s.mu.Lock()
	defer s.mu.Unlock()

	added := 0
	for _, file := range files {
		if slices.Contains(s.files[file.Hash].peers, peerID) {
			continue
		}
		s.files[file.Hash].peers = append(s.files[file.Hash].peers, peerID)
		if slices.Contains(s.peers[peerID], conn) {
			continue
		}
		s.peers[peerID] = append(s.peers[peerID], conn)
		added++
	}
	return added
}

func (s *Store) GetPeers(hash protocol.FileHash) []protocol.NodeID {
	s.mu.Lock()
	defer s.mu.Unlock()

	f, exists := s.files[hash]
	if !exists {
		return nil
	}

	peers := make([]protocol.NodeID, len(f.peers))
	copy(peers, f.peers)
	return peers
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
