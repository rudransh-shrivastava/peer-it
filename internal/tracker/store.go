package tracker

import (
	"slices"
	"sync"

	"github.com/rudransh-shrivastava/peer-it/internal/protocol"
)

type callKey struct {
	source protocol.NodeID
	target protocol.NodeID
}

type file struct {
	metadata *protocol.FileEntry
	peers    []protocol.NodeID
}

type Store struct {
	mu           sync.Mutex
	files        map[protocol.FileHash]*file
	peers        map[protocol.NodeID]Conn
	pendingCalls map[callKey]chan protocol.STUNCandidates
}

func NewStore() *Store {
	return &Store{
		files:        make(map[protocol.FileHash]*file),
		peers:        make(map[protocol.NodeID]Conn),
		pendingCalls: make(map[callKey]chan protocol.STUNCandidates),
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

	if _, ok := s.peers[peerID]; !ok {
		s.peers[peerID] = conn
	}

	added := 0
	for _, file := range files {
		if slices.Contains(s.files[file.Hash].peers, peerID) {
			continue
		}
		s.files[file.Hash].peers = append(s.files[file.Hash].peers, peerID)
		added++
	}
	return added
}

func (s *Store) AddPendingCallCh(sourceID, targetID protocol.NodeID, ch chan protocol.STUNCandidates) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.pendingCalls[callKey{sourceID, targetID}] = ch
}

func (s *Store) GetPeer(peerID protocol.NodeID) (Conn, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if conn, ok := s.peers[peerID]; ok {
		return conn, true
	}
	return nil, false
}

func (s *Store) GetPeerID(conn Conn) (protocol.NodeID, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for id, c := range s.peers {
		if c == conn {
			return id, true
		}
	}
	return protocol.NodeID{}, false
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

func (s *Store) GetPendingCallCh(sourceID, targetID protocol.NodeID) (chan protocol.STUNCandidates, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ch, ok := s.pendingCalls[callKey{sourceID, targetID}]
	return ch, ok
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

func (s *Store) RemovePendingCallCh(sourceID, targetID protocol.NodeID) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.pendingCalls, callKey{sourceID, targetID})
}
