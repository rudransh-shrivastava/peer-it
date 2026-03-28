package tracker

import (
	"context"
	"testing"
	"time"

	"github.com/rudransh-shrivastava/peer-it/internal/protocol"
)

// Test that when a source peer requests a call to a target peer, the server
// forwards a CallReq to the target and later delivers STUN candidates from
// the target back to the source via the pending call channel.
func TestSTUNCandidatesSignaling(t *testing.T) {
	srv, _ := setupMockServerConn(t)
	defer func() { _ = srv.Shutdown() }()

	// Prepare two mock connections and assign them stable NodeIDs in the store.
	sourceConn := newMockConn("source-peer")
	targetConn := newMockConn("target-peer")

	sourceID := protocol.NodeID{1}
	targetID := protocol.NodeID{2}

	// Register connections in the store so GetPeerID/GetPeer succeed.
	srv.store.mu.Lock()
	srv.store.peers[sourceID] = sourceConn
	srv.store.peers[targetID] = targetConn
	srv.store.mu.Unlock()

	// Call from source to target — run in goroutine because it will block
	// waiting for STUN candidates to be provided on the pending channel.
	ctx := context.Background()
	go srv.handleMessage(ctx, sourceConn, &protocol.CallReq{TargetNodeID: targetID})

	// Wait for the pending channel to be created by the server.
	var ch chan protocol.STUNCandidates
	var ok bool
	deadline := time.After(200 * time.Millisecond)
	for {
		select {
		case <-deadline:
			t.Fatal("timed out waiting for pending call channel")
		default:
			ch, ok = srv.store.GetPendingCallCh(sourceID, targetID)
			if ok && ch != nil {
				goto GOTCH
			}
			time.Sleep(5 * time.Millisecond)
		}
	}
GOTCH:

	// Ensure the server forwarded a CallReq to the target connection.
	if len(targetConn.sentMsgs) == 0 {
		t.Fatalf("expected CallReq sent to target, none found")
	}
	if callReq, ok := targetConn.sentMsgs[0].(*protocol.CallReq); !ok {
		t.Fatalf("expected *protocol.CallReq sent to target, got %T", targetConn.sentMsgs[0])
	} else if callReq.TargetNodeID != sourceID {
		t.Fatalf("expected CallReq.TargetNodeID %v, got %v", sourceID, callReq.TargetNodeID)
	}

	// Simulate the target replying with STUN candidates via the pending channel.
	// Target should send STUN candidates intended for the source peer
	candidates := protocol.STUNCandidates{
		Candidates:   []protocol.STUNCandidate{{IP: "198.51.100.2", Port: 54321}},
		TargetNodeID: sourceID,
	}

	select {
	case ch <- candidates:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("timed out sending STUNCandidates into pending channel")
	}

	// Give server a brief moment to forward candidates to the source.
	time.Sleep(10 * time.Millisecond)

	if len(sourceConn.sentMsgs) == 0 {
		t.Fatalf("expected STUNCandidates sent to source, none found")
	}
	// expect the server to forward a value of STUNCandidates
	resVal, ok := sourceConn.sentMsgs[0].(protocol.STUNCandidates)
	if !ok {
		t.Fatalf("expected protocol.STUNCandidates (value) sent to source, got %T", sourceConn.sentMsgs[0])
	}
	res := resVal
	if len(res.Candidates) != 1 {
		t.Fatalf("expected 1 candidate forwarded to source, got %d", len(res.Candidates))
	}
	if res.TargetNodeID != sourceID {
		t.Fatalf("expected STUNCandidates.TargetNodeID %v, got %v", sourceID, res.TargetNodeID)
	}
}
