package daemon

import (
	"fmt"

	"github.com/rudransh-shrivastava/peer-it/internal/shared/protocol"
	"google.golang.org/protobuf/proto"
)

func (d *Daemon) sendIntroduction(peerID string, fileHash string) error {
	// Get chunk availability
	chunks, err := d.ChunkStore.GetChunks(fileHash)
	if err != nil {
		return fmt.Errorf("failed to get chunks: %v", err)
	}

	chunksMap := make([]int32, len(*chunks))
	for _, chunk := range *chunks {
		chunksMap[chunk.Index] = 1
	}

	introMsg := &protocol.NetworkMessage{
		MessageType: &protocol.NetworkMessage_Introduction{
			Introduction: &protocol.IntroductionMessage{
				FileHash:  fileHash,
				ChunksMap: chunksMap,
			},
		},
	}

	dc, exists := d.PeerDataChannels[peerID]
	if !exists {
		return fmt.Errorf("data channel not found for peer: %s", peerID)
	}

	data, err := proto.Marshal(introMsg)
	if err != nil {
		return fmt.Errorf("failed to marshal introduction: %v", err)
	}

	err = dc.Send(data)
	if err != nil {
		return fmt.Errorf("failed to send introduction: %v", err)
	}

	return nil
}

func (d *Daemon) handleIntroduction(peerID string, msg *protocol.IntroductionMessage) {
	d.Logger.Infof("Handling introduction from peer %s for file %s", peerID, msg.FileHash)
	d.Logger.Infof("Received chunks map of length: %d", len(msg.ChunksMap))

	// Store peer's chunk availability
	d.updatePeerChunkMap(peerID, msg.FileHash, msg.ChunksMap)
	d.Logger.Infof("Received chunk map from peer %s:%v for file: %s", peerID, msg.ChunksMap, msg.FileHash)
	// Request missing chunks
}

func (d *Daemon) updatePeerChunkMap(peerID string, fileHash string, chunksMap []int32) {
	d.mu.Lock()
	d.PeerChunkMap[peerID] = make(map[string][]int32)
	d.PeerChunkMap[peerID][fileHash] = chunksMap
	d.mu.Unlock()
}
