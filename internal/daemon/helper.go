package daemon

import (
	"fmt"
	"os"
	"time"

	"github.com/rudransh-shrivastava/peer-it/internal/shared/protocol"
	"google.golang.org/protobuf/proto"
)

func (d *Daemon) GetChunkMap(fileHash string) []int32 {
	file, err := d.FileStore.GetFileByHash(fileHash)
	if err != nil {
		d.Logger.Warnf("failed to get file: %v", err)
		return nil
	}
	chunks, err := d.ChunkStore.GetChunks(fileHash)
	if err != nil {
		d.Logger.Warnf("failed to get chunks: %v", err)
		return nil
	}

	chunksMap := make([]int32, file.TotalChunks)
	for _, chunk := range *chunks {
		if chunk.IsAvailable {
			chunksMap[chunk.ChunkIndex] = 1
		}
	}
	return chunksMap
}
func (d *Daemon) sendIntroduction(peerID string, fileHash string) error {
	// Get chunk availability
	chunksMap := d.GetChunkMap(fileHash)

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

func (d *Daemon) readChunkFromFile(fileHash string, chunkIndex int, chunkSize int, maxChunkSize int) ([]byte, error) {
	fileName, err := d.FileStore.GetFileNameByHash(fileHash)
	if err != nil {
		return nil, fmt.Errorf("failed to get file name: %v", err)
	}
	dir := fmt.Sprintf("downloads/daemon-%s/", d.IPCSocketIndex)
	filePath := dir + fileName

	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %v", err)
	}
	defer file.Close()
	offset := int64(chunkIndex * maxChunkSize)
	_, err = file.Seek(offset, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to seek file: %v", err)
	}
	data := make([]byte, chunkSize)
	_, err = file.Read(data)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %v", err)
	}
	return data, nil
}

func (d *Daemon) handleChunkRequest(peerID string, msg *protocol.ChunkRequest) {
	d.Logger.Infof("Handling chunk request from peer %s for file %s, chunk %d", peerID, msg.FileHash, msg.ChunkIndex)

	// Check if we have the chunk
	chunk, err := d.ChunkStore.GetChunk(msg.FileHash, int(msg.ChunkIndex))
	if err != nil {
		d.Logger.Warnf("Failed to get chunk: %v", err)
		return
	}
	if !chunk.IsAvailable {
		d.Logger.Warnf("Chunk not available: %d", msg.ChunkIndex)
		return
	}
	chunkIndex := chunk.ChunkIndex
	chunkSize := chunk.ChunkSize
	// Read chunk data from index to size from the file directly
	chunkData, err := d.readChunkFromFile(msg.FileHash, chunkIndex, chunkSize, maxChunkSize)

	// Send the chunk
	chunkMsg := &protocol.NetworkMessage{
		MessageType: &protocol.NetworkMessage_ChunkResponse{
			ChunkResponse: &protocol.ChunkResponse{
				FileHash:   msg.FileHash,
				ChunkIndex: msg.ChunkIndex,
				ChunkData:  chunkData,
			},
		},
	}

	dc, exists := d.PeerDataChannels[peerID]
	if !exists {
		d.Logger.Warnf("Data channel not found for peer: %s", peerID)
		return
	}

	data, err := proto.Marshal(chunkMsg)
	if err != nil {
		d.Logger.Warnf("Failed to marshal chunk: %v", err)
		return
	}

	err = dc.Send(data)
	if err != nil {
		d.Logger.Warnf("Failed to send chunk: %v", err)
		return
	}
}

func (d *Daemon) handleChunkResponse(peerId string, msg *protocol.ChunkResponse) {
	d.Logger.Infof("Received chunk response from peer %s for file %s, chunk %d", peerId, msg.FileHash, msg.ChunkIndex)

	// Write the chunk to the file
	fileName, err := d.FileStore.GetFileNameByHash(msg.FileHash)
	if err != nil {
		d.Logger.Warnf("Failed to get file name: %v", err)
		return
	}
	dir := fmt.Sprintf("downloads/daemon-%s/", d.IPCSocketIndex)
	filePath := dir + fileName

	file, err := os.OpenFile(filePath, os.O_RDWR, 0644)
	if err != nil {
		d.Logger.Warnf("Failed to open file: %v", err)
		return
	}
	defer file.Close()

	offset := int64(int(msg.ChunkIndex) * maxChunkSize)
	_, err = file.Seek(offset, 0)
	if err != nil {
		d.Logger.Warnf("Failed to seek file: %v", err)
		return
	}

	// before writing verify the chunk data from chunk store
	_, err = file.Write(msg.ChunkData)
	if err != nil {
		d.Logger.Warnf("Failed to write chunk to file: %v", err)
		return
	}

	// set our map to reflect that we have the chunk
	d.mu.Lock()
	d.PeerChunkMap[d.ID][msg.FileHash][msg.ChunkIndex] = 1
	d.mu.Unlock()

	// mark the chunk as available in the chunk store
	err = d.ChunkStore.MarkChunkAvailable(msg.FileHash, int(msg.ChunkIndex))
}

func (d *Daemon) handleIntroduction(peerId string, msg *protocol.IntroductionMessage) {
	d.Logger.Infof("Handling introduction from peer %s for file %s", peerId, msg.FileHash)
	d.Logger.Infof("Received chunks map of length: %d", len(msg.ChunksMap))

	// If its our first time receiving the chunks map
	// we will need to send our chunks map
	_, exists := d.PeerChunkMap[peerId][msg.FileHash]
	if !exists {
		d.Logger.Infof("Sending our chunks map to peer %s for file %s", peerId, msg.FileHash)
		err := d.sendIntroduction(peerId, msg.FileHash)
		if err != nil {
			d.Logger.Warnf("Failed to send introduction: %v", err)
		}

		d.mu.Lock()
		d.PeerChunkMap[peerId] = make(map[string][]int32)
		d.PeerChunkMap[peerId][msg.GetFileHash()] = msg.GetChunksMap()
		d.mu.Unlock()

	} else {
		d.Logger.Infof("Chunks map already exists for peer %s for file %s", peerId, msg.FileHash)

		d.mu.Lock()
		d.PeerChunkMap[peerId][msg.GetFileHash()] = msg.GetChunksMap()
		d.mu.Unlock()
	}

	d.Logger.Infof("Received chunk map from peer %s:%v for file: %s", peerId, msg.ChunksMap, msg.FileHash)

	// Add peer to our peer list
	d.Logger.Infof("Adding peer %s to our swarm for file: %s", peerId, msg.GetFileHash())
	d.addPeerToDownload(peerId, msg.GetFileHash())
}

func (d *Daemon) sendHeartBeatsToTracker() {
	ticker := time.NewTicker(heartbeatInterval * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-d.Ctx.Done():
			d.Logger.Info("Stopping the heart")
			return
		case <-ticker.C:
			d.Logger.Info("Sending a heartbeat")
			hb := &protocol.HeartbeatMessage{
				Timestamp: time.Now().Unix(),
			}

			netMsg := &protocol.NetworkMessage{
				MessageType: &protocol.NetworkMessage_Heartbeat{
					Heartbeat: hb,
				},
			}

			err := d.TrackerRouter.WriteMessage(netMsg)
			if err != nil {
				d.Logger.Warnf("Error sending message to Tracker: %v", err)
			}
		}
	}
}
