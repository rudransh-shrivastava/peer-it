package node

import (
	"context"
	"fmt"
	"io"
	"os"

	internaldb "github.com/rudransh-shrivastava/peer-it/internal/db"
	"github.com/rudransh-shrivastava/peer-it/internal/shared/protocol"
	"google.golang.org/protobuf/proto"
)

func BuildChunkMap(totalChunks int, chunks []internaldb.Chunk) []int32 {
	result := make([]int32, totalChunks)
	for _, c := range chunks {
		if c.IsAvailable == 1 && c.ChunkIndex < int64(totalChunks) {
			result[c.ChunkIndex] = 1
		}
	}
	return result
}

func FindMissingChunkIndex(chunks []internaldb.Chunk) int32 {
	for i, chunk := range chunks {
		if chunk.IsAvailable != 1 {
			return int32(i)
		}
	}
	return -1
}

func IsDownloadComplete(chunks []internaldb.Chunk) bool {
	for _, chunk := range chunks {
		if chunk.IsAvailable != 1 {
			return false
		}
	}
	return true
}

func ReadChunkData(r io.ReaderAt, chunkIndex, chunkSize, maxChunkSize int) ([]byte, error) {
	offset := int64(chunkIndex * maxChunkSize)
	data := make([]byte, chunkSize)
	_, err := r.ReadAt(data, offset)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func WriteChunkData(w io.WriterAt, chunkIndex, maxChunkSize int, data []byte) error {
	offset := int64(chunkIndex * maxChunkSize)
	_, err := w.WriteAt(data, offset)
	return err
}

func (n *Node) GetChunkMap(fileHash string) []int32 {
	ctx := context.Background()
	file, err := n.files.GetFileByHash(ctx, fileHash)
	if err != nil {
		n.logger.Warnf("failed to get file: %v", err)
		return nil
	}
	chunks, err := n.chunks.GetChunks(ctx, fileHash)
	if err != nil {
		n.logger.Warnf("failed to get chunks: %v", err)
		return nil
	}
	return BuildChunkMap(int(file.TotalChunks), chunks)
}

func (n *Node) readChunkFromFile(fileHash string, chunkIndex int, chunkSize int, maxChunkSz int) ([]byte, error) {
	ctx := context.Background()
	fileName, err := n.files.GetFileNameByHash(ctx, fileHash)
	if err != nil {
		return nil, fmt.Errorf("failed to get file name: %v", err)
	}
	filePath := BuildDownloadPath(n.ipcSocketIndex, fileName)

	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %v", err)
	}
	defer func() { _ = file.Close() }()

	return ReadChunkData(file, chunkIndex, chunkSize, maxChunkSz)
}

func (n *Node) handleChunkRequest(peerID string, msg *protocol.ChunkRequest) {
	ctx := context.Background()
	n.logger.Infof("Handling chunk request from peer %s for file %s, chunk %d", peerID, msg.FileHash, msg.ChunkIndex)

	chunk, err := n.chunks.GetChunk(ctx, msg.FileHash, int(msg.ChunkIndex))
	if err != nil {
		n.logger.Warnf("Failed to get chunk: %v", err)
		return
	}
	if chunk.IsAvailable != 1 {
		n.logger.Warnf("Chunk not available: %d", msg.ChunkIndex)
		return
	}

	chunkData, err := n.readChunkFromFile(msg.FileHash, int(chunk.ChunkIndex), int(chunk.ChunkSize), maxChunkSize)
	if err != nil {
		n.logger.Warnf("Failed to read chunk from file: %v", err)
		return
	}

	dc, exists := n.peerDataChannels[peerID]
	if !exists {
		n.logger.Warnf("Data channel not found for peer: %s", peerID)
		return
	}

	data, err := proto.Marshal(BuildChunkResponseMessage(msg.FileHash, msg.ChunkIndex, chunkData))
	if err != nil {
		n.logger.Warnf("Failed to marshal chunk: %v", err)
		return
	}

	if err := dc.Send(data); err != nil {
		n.logger.Warnf("Failed to send chunk: %v", err)
	}
}

func (n *Node) handleChunkResponse(peerID string, msg *protocol.ChunkResponse) {
	ctx := context.Background()
	n.messageCLI(fmt.Sprintf("Got chunk %d from peer %s", msg.ChunkIndex, peerID))
	n.messageCLI("progress")
	n.logger.Infof("Received chunk response from peer %s for file %s, chunk %d", peerID, msg.FileHash, msg.ChunkIndex)

	fileName, err := n.files.GetFileNameByHash(ctx, msg.FileHash)
	if err != nil {
		n.logger.Warnf("Failed to get file name: %v", err)
		return
	}
	filePath := BuildDownloadPath(n.ipcSocketIndex, fileName)

	file, err := os.OpenFile(filePath, os.O_RDWR, 0644)
	if err != nil {
		n.logger.Warnf("Failed to open file: %v", err)
		return
	}
	defer func() { _ = file.Close() }()

	if err := WriteChunkData(file, int(msg.ChunkIndex), maxChunkSize, msg.ChunkData); err != nil {
		n.logger.Warnf("Failed to write chunk to file: %v", err)
		return
	}

	n.mu.Lock()
	n.peerChunkMap[n.id][msg.FileHash][msg.ChunkIndex] = 1
	n.mu.Unlock()

	if err := n.chunks.MarkChunkAvailable(ctx, msg.FileHash, int(msg.ChunkIndex)); err != nil {
		n.logger.Warnf("Failed to mark chunk as available: %v", err)
	}
}
