package node

import (
	"sort"
	"testing"
)

func TestFindEligiblePeers_NoMatch(t *testing.T) {
	peerChunkMap := map[string]map[string][]int32{
		"peer1": {"file123": {0, 0, 0}},
		"peer2": {"file123": {0, 0, 0}},
	}

	result := FindEligiblePeers(peerChunkMap, "file123", 1)
	if len(result) != 0 {
		t.Errorf("expected 0 eligible, got %d", len(result))
	}
}

func TestFindEligiblePeers_OneMatch(t *testing.T) {
	peerChunkMap := map[string]map[string][]int32{
		"peer1": {"file123": {1, 0, 0}},
		"peer2": {"file123": {0, 1, 0}},
	}

	result := FindEligiblePeers(peerChunkMap, "file123", 1)
	if len(result) != 1 {
		t.Errorf("expected 1 eligible, got %d", len(result))
	}
	if result[0] != "peer2" {
		t.Errorf("expected peer2, got %s", result[0])
	}
}

func TestFindEligiblePeers_MultipleMatch(t *testing.T) {
	peerChunkMap := map[string]map[string][]int32{
		"peer1": {"file123": {1, 1, 0}},
		"peer2": {"file123": {0, 1, 1}},
		"peer3": {"file123": {1, 1, 1}},
	}

	result := FindEligiblePeers(peerChunkMap, "file123", 1)
	sort.Strings(result)

	if len(result) != 3 {
		t.Errorf("expected 3 eligible, got %d", len(result))
	}
}

func TestFindEligiblePeers_WrongFile(t *testing.T) {
	peerChunkMap := map[string]map[string][]int32{
		"peer1": {"file123": {1, 1, 1}},
	}

	result := FindEligiblePeers(peerChunkMap, "file999", 0)
	if len(result) != 0 {
		t.Errorf("expected 0 eligible for wrong file, got %d", len(result))
	}
}

func TestFindEligiblePeers_OutOfBounds(t *testing.T) {
	peerChunkMap := map[string]map[string][]int32{
		"peer1": {"file123": {1, 1}},
	}

	result := FindEligiblePeers(peerChunkMap, "file123", 10)
	if len(result) != 0 {
		t.Errorf("expected 0 for out of bounds, got %d", len(result))
	}
}

func TestSelectRandomPeer_Empty(t *testing.T) {
	result := SelectRandomPeer([]string{}, func(n int) int { return 0 })
	if result != "" {
		t.Errorf("expected empty string, got %s", result)
	}
}

func TestSelectRandomPeer_Single(t *testing.T) {
	result := SelectRandomPeer([]string{"peer1"}, func(n int) int { return 0 })
	if result != "peer1" {
		t.Errorf("expected peer1, got %s", result)
	}
}

func TestSelectRandomPeer_Multiple(t *testing.T) {
	peers := []string{"peer1", "peer2", "peer3"}

	result := SelectRandomPeer(peers, func(n int) int { return 1 })
	if result != "peer2" {
		t.Errorf("expected peer2 (index 1), got %s", result)
	}

	result = SelectRandomPeer(peers, func(n int) int { return 2 })
	if result != "peer3" {
		t.Errorf("expected peer3 (index 2), got %s", result)
	}
}
