package node

import (
	"fmt"

	"github.com/rudransh-shrivastava/peer-it/internal/shared/protocol"
	"google.golang.org/protobuf/proto"
)

func FindEligiblePeers(peerChunkMap map[string]map[string][]int32, fileHash string, chunkIndex int32) []string {
	var eligible []string
	for peerID, peerFiles := range peerChunkMap {
		if chunks, exists := peerFiles[fileHash]; exists {
			if chunkIndex < int32(len(chunks)) && chunks[chunkIndex] == 1 {
				eligible = append(eligible, peerID)
			}
		}
	}
	return eligible
}

func SelectRandomPeer(peers []string, randFn func(n int) int) string {
	if len(peers) == 0 {
		return ""
	}
	return peers[randFn(len(peers))]
}

func (n *Node) sendIntroduction(peerID string, fileHash string) error {
	chunksMap := n.GetChunkMap(fileHash)

	dc, exists := n.peerDataChannels[peerID]
	if !exists {
		return fmt.Errorf("data channel not found for peer: %s", peerID)
	}

	data, err := proto.Marshal(BuildIntroductionMessage(fileHash, chunksMap))
	if err != nil {
		return fmt.Errorf("failed to marshal introduction: %v", err)
	}

	if err := dc.Send(data); err != nil {
		return fmt.Errorf("failed to send introduction: %v", err)
	}

	return nil
}

func (n *Node) handleIntroduction(peerID string, msg *protocol.IntroductionMessage) {
	n.logger.Infof("Handling introduction from peer %s for file %s", peerID, msg.FileHash)
	n.logger.Infof("Received chunks map of length: %d", len(msg.ChunksMap))
	n.messageCLI(fmt.Sprintf("Connected to peer %s", peerID))

	_, exists := n.peerChunkMap[peerID][msg.FileHash]
	if !exists {
		n.logger.Infof("Sending our chunks map to peer %s for file %s", peerID, msg.FileHash)
		if err := n.sendIntroduction(peerID, msg.FileHash); err != nil {
			n.logger.Warnf("Failed to send introduction: %v", err)
		}

		n.mu.Lock()
		n.peerChunkMap[peerID] = make(map[string][]int32)
		n.peerChunkMap[peerID][msg.GetFileHash()] = msg.GetChunksMap()
		n.mu.Unlock()
	} else {
		n.logger.Infof("Chunks map already exists for peer %s for file %s", peerID, msg.FileHash)

		n.mu.Lock()
		n.peerChunkMap[peerID][msg.GetFileHash()] = msg.GetChunksMap()
		n.mu.Unlock()
	}

	n.logger.Infof("Received chunk map from peer %s:%v for file: %s", peerID, msg.ChunksMap, msg.FileHash)

	n.logger.Infof("Adding peer %s to our swarm for file: %s", peerID, msg.GetFileHash())
	n.addPeerToDownload(peerID, msg.GetFileHash())
}
