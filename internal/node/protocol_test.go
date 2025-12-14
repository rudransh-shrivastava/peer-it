package node

import (
	"testing"

	"github.com/rudransh-shrivastava/peer-it/internal/shared/protocol"
)

func TestBuildSignalingOfferMessage(t *testing.T) {
	msg := BuildSignalingOfferMessage("peer123", "v=0...")
	sig := msg.GetSignaling()

	if sig == nil {
		t.Fatal("expected signaling message")
	}
	if sig.TargetPeerId != "peer123" {
		t.Errorf("expected peer123, got %s", sig.TargetPeerId)
	}
	if sig.GetOffer() == nil {
		t.Fatal("expected offer")
	}
	if sig.GetOffer().Sdp != "v=0..." {
		t.Errorf("expected 'v=0...', got %s", sig.GetOffer().Sdp)
	}
}

func TestBuildSignalingAnswerMessage(t *testing.T) {
	msg := BuildSignalingAnswerMessage("peer456", "v=1...")
	sig := msg.GetSignaling()

	if sig == nil {
		t.Fatal("expected signaling message")
	}
	if sig.GetAnswer() == nil {
		t.Fatal("expected answer")
	}
	if sig.GetAnswer().Sdp != "v=1..." {
		t.Errorf("expected 'v=1...', got %s", sig.GetAnswer().Sdp)
	}
}

func TestBuildAnnounceMessage(t *testing.T) {
	files := []*protocol.FileInfo{
		{FileHash: "abc123", FileSize: 1024, ChunkSize: 256, TotalChunks: 4},
	}
	msg := BuildAnnounceMessage(files)

	announce := msg.GetAnnounce()
	if announce == nil {
		t.Fatal("expected announce message")
	}
	if len(announce.Files) != 1 {
		t.Errorf("expected 1 file, got %d", len(announce.Files))
	}
	if announce.Files[0].FileHash != "abc123" {
		t.Errorf("expected abc123, got %s", announce.Files[0].FileHash)
	}
}

func TestBuildPeerListRequestMessage(t *testing.T) {
	msg := BuildPeerListRequestMessage("hash123")
	req := msg.GetPeerListRequest()

	if req == nil {
		t.Fatal("expected peer list request")
	}
	if req.FileHash != "hash123" {
		t.Errorf("expected hash123, got %s", req.FileHash)
	}
}

func TestBuildHeartbeatMessage(t *testing.T) {
	msg := BuildHeartbeatMessage(1234567890)
	hb := msg.GetHeartbeat()

	if hb == nil {
		t.Fatal("expected heartbeat")
	}
	if hb.Timestamp != 1234567890 {
		t.Errorf("expected 1234567890, got %d", hb.Timestamp)
	}
}

func TestBuildLogMessage(t *testing.T) {
	msg := BuildLogMessage("test log")
	log := msg.GetLog()

	if log == nil {
		t.Fatal("expected log message")
	}
	if log.Message != "test log" {
		t.Errorf("expected 'test log', got %s", log.Message)
	}
}

func TestBuildIntroductionMessage(t *testing.T) {
	chunks := []int32{1, 0, 1, 1}
	msg := BuildIntroductionMessage("file123", chunks)
	intro := msg.GetIntroduction()

	if intro == nil {
		t.Fatal("expected introduction")
	}
	if intro.FileHash != "file123" {
		t.Errorf("expected file123, got %s", intro.FileHash)
	}
	if len(intro.ChunksMap) != 4 {
		t.Errorf("expected 4 chunks, got %d", len(intro.ChunksMap))
	}
}

func TestBuildChunkRequestMessage(t *testing.T) {
	msg := BuildChunkRequestMessage("hash", 5)
	req := msg.GetChunkRequest()

	if req == nil {
		t.Fatal("expected chunk request")
	}
	if req.FileHash != "hash" {
		t.Errorf("expected 'hash', got %s", req.FileHash)
	}
	if req.ChunkIndex != 5 {
		t.Errorf("expected 5, got %d", req.ChunkIndex)
	}
}

func TestBuildChunkResponseMessage(t *testing.T) {
	data := []byte("chunk data")
	msg := BuildChunkResponseMessage("hash", 3, data)
	resp := msg.GetChunkResponse()

	if resp == nil {
		t.Fatal("expected chunk response")
	}
	if resp.FileHash != "hash" {
		t.Errorf("expected 'hash', got %s", resp.FileHash)
	}
	if resp.ChunkIndex != 3 {
		t.Errorf("expected 3, got %d", resp.ChunkIndex)
	}
	if string(resp.ChunkData) != "chunk data" {
		t.Errorf("expected 'chunk data', got %s", string(resp.ChunkData))
	}
}
