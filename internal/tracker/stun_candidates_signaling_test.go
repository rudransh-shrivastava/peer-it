package tracker

import (
	"context"
	"testing"

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

	// Call from source to target; server should create pending tokens and
	// forward a CallReq to the target.
	ctx := context.Background()
	srv.handleMessage(ctx, sourceConn, &protocol.CallReq{TargetNodeID: targetID})

	// Ensure the server forwarded a CallReq to the target connection.
	if len(targetConn.sentMsgs) == 0 {
		t.Fatalf("expected CallReq sent to target, none found")
	}
	if callReq, ok := targetConn.sentMsgs[0].(*protocol.CallReq); !ok {
		t.Fatalf("expected *protocol.CallReq sent to target, got %T", targetConn.sentMsgs[0])
	} else if callReq.TargetNodeID != sourceID {
		t.Fatalf("expected CallReq.TargetNodeID %v, got %v", sourceID, callReq.TargetNodeID)
	}

	// The server should have created pending call tokens for both directions.
	if ok := srv.store.ConsumePendingCall(sourceID, targetID); !ok {
		t.Fatalf("expected pending call for (%v,%v) to exist", sourceID, targetID)
	}
	if ok := srv.store.ConsumePendingCall(targetID, sourceID); !ok {
		t.Fatalf("expected pending call for (%v,%v) to exist", targetID, sourceID)
	}
}
