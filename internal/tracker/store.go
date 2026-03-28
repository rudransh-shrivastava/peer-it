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
	pendingCalls map[callKey]struct{}
}

func NewStore() *Store {
	return &Store{
		files:        make(map[protocol.FileHash]*file),
		peers:        make(map[protocol.NodeID]Conn),
		pendingCalls: make(map[callKey]struct{}),
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

// Must be called twice with switched IDs to allow bi-directional communication.
func (s *Store) AddPendingCall(sourceID, targetID protocol.NodeID) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.pendingCalls[callKey{sourceID, targetID}] = struct{}{}
}

// Returns true if the call was pending (and consumes the token)
func (s *Store) ConsumePendingCall(sourceID, targetID protocol.NodeID) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := callKey{sourceID, targetID}
	_, ok := s.pendingCalls[key]
	if ok {
		delete(s.pendingCalls, key)
	}
	return ok
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

func (s *Store) ListFiles() []protocol.FileEntry {
	s.mu.Lock()
	defer s.mu.Unlock()

	files := []protocol.FileEntry{}
	for _, file := range s.files {
		files = append(files, *file.metadata)
	}
	return files
}
