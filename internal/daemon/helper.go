package daemon

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/rudransh-shrivastava/peer-it/internal/shared/protocol"
	"google.golang.org/protobuf/proto"
)

func (d *Daemon) GetChunkMap(fileHash string) []int32 {
	ctx := context.Background()
	file, err := d.FileStore.GetFileByHash(ctx, fileHash)
	if err != nil {
		d.Logger.Warnf("failed to get file: %v", err)
		return nil
	}
	chunks, err := d.ChunkStore.GetChunks(ctx, fileHash)
	if err != nil {
		d.Logger.Warnf("failed to get chunks: %v", err)
		return nil
	}

	chunksMap := make([]int32, file.TotalChunks)
	for _, chunk := range chunks {
		if chunk.IsAvailable == 1 {
			chunksMap[chunk.ChunkIndex] = 1
		}
	}
	return chunksMap
}

func (d *Daemon) sendIntroduction(peerID string, fileHash string) error {
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

func (d *Daemon) readChunkFromFile(fileHash string, chunkIndex int, chunkSize int, maxChunkSz int) ([]byte, error) {
	ctx := context.Background()
	fileName, err := d.FileStore.GetFileNameByHash(ctx, fileHash)
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
	offset := int64(chunkIndex * maxChunkSz)
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
	ctx := context.Background()
	d.Logger.Infof("Handling chunk request from peer %s for file %s, chunk %d", peerID, msg.FileHash, msg.ChunkIndex)

	chunk, err := d.ChunkStore.GetChunk(ctx, msg.FileHash, int(msg.ChunkIndex))
	if err != nil {
		d.Logger.Warnf("Failed to get chunk: %v", err)
		return
	}
	if chunk.IsAvailable != 1 {
		d.Logger.Warnf("Chunk not available: %d", msg.ChunkIndex)
		return
	}

	chunkData, err := d.readChunkFromFile(msg.FileHash, int(chunk.ChunkIndex), int(chunk.ChunkSize), maxChunkSize)
	if err != nil {
		d.Logger.Warnf("Failed to read chunk from file: %v", err)
		return
	}

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

func (d *Daemon) handleChunkResponse(peerID string, msg *protocol.ChunkResponse) {
	ctx := context.Background()
	logMsg := fmt.Sprintf("Got chunk %d from peer %s", msg.ChunkIndex, peerID)
	d.messageCLI(logMsg)
	d.messageCLI("progress")
	d.Logger.Infof("Received chunk response from peer %s for file %s, chunk %d", peerID, msg.FileHash, msg.ChunkIndex)

	fileName, err := d.FileStore.GetFileNameByHash(ctx, msg.FileHash)
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

	_, err = file.Write(msg.ChunkData)
	if err != nil {
		d.Logger.Warnf("Failed to write chunk to file: %v", err)
		return
	}

	d.mu.Lock()
	d.PeerChunkMap[d.ID][msg.FileHash][msg.ChunkIndex] = 1
	d.mu.Unlock()

	if err := d.ChunkStore.MarkChunkAvailable(ctx, msg.FileHash, int(msg.ChunkIndex)); err != nil {
		d.Logger.Warnf("Failed to mark chunk as available: %v", err)
	}
}

func (d *Daemon) handleIntroduction(peerID string, msg *protocol.IntroductionMessage) {
	d.Logger.Infof("Handling introduction from peer %s for file %s", peerID, msg.FileHash)
	d.Logger.Infof("Received chunks map of length: %d", len(msg.ChunksMap))
	logMsg := fmt.Sprintf("Connected to peer %s", peerID)
	d.messageCLI(logMsg)

	_, exists := d.PeerChunkMap[peerID][msg.FileHash]
	if !exists {
		d.Logger.Infof("Sending our chunks map to peer %s for file %s", peerID, msg.FileHash)
		err := d.sendIntroduction(peerID, msg.FileHash)
		if err != nil {
			d.Logger.Warnf("Failed to send introduction: %v", err)
		}

		d.mu.Lock()
		d.PeerChunkMap[peerID] = make(map[string][]int32)
		d.PeerChunkMap[peerID][msg.GetFileHash()] = msg.GetChunksMap()
		d.mu.Unlock()
	} else {
		d.Logger.Infof("Chunks map already exists for peer %s for file %s", peerID, msg.FileHash)

		d.mu.Lock()
		d.PeerChunkMap[peerID][msg.GetFileHash()] = msg.GetChunksMap()
		d.mu.Unlock()
	}

	d.Logger.Infof("Received chunk map from peer %s:%v for file: %s", peerID, msg.ChunksMap, msg.FileHash)

	d.Logger.Infof("Adding peer %s to our swarm for file: %s", peerID, msg.GetFileHash())
	d.addPeerToDownload(peerID, msg.GetFileHash())
}

func (d *Daemon) messageCLI(msg string) {
	d.Logger.Debugf("Sending LOG: %s to CLI", msg)
	cliRouter := d.CLIRouter
	if cliRouter == nil {
		d.Logger.Warnf("CLI router not found")
		return
	}

	netMsg := &protocol.NetworkMessage{
		MessageType: &protocol.NetworkMessage_Log{
			Log: &protocol.LogMessage{
				Message: msg,
			},
		},
	}
	if err := cliRouter.WriteMessage(netMsg); err != nil {
		d.Logger.Warnf("Failed to send message to CLI: %v", err)
	}
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
