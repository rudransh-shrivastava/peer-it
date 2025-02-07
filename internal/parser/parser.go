package parser

import (
	"os"

	"github.com/rudransh-shrivastava/peer-it/internal/shared/utils/logger"
)

/*
	Format of the .p2p file

fileHash:string
fileSize:int
maxChunkSize:int
totalChunks:int
0:chunkHash(string)|chunkSize(int)
1:chunkHash(string)|chunkSize(int)
n:chunkHash(string)|chunkSize(int)

each line represents a key value pair where the key is separated from the value by a colon
except for the chunk lines, where the key is the index of the chunk
and the value is the chunkHash and chunkSize separated by a pipe character
each line is separated by a newline character
*/

type ParserFile struct {
	FileHash     string
	FileSize     string
	MaxChunkSize string
	TotalChunks  string
	Chunks       []ParserChunk
}
type ParserChunk struct {
	ChunkIndex string
	ChunkHash  string
	ChunkSize  string
}

func GenerateP2PFile(parserFile *ParserFile, path string) error {
	logger := logger.NewLogger()
	p2pFile, err := os.Create(path)
	if err != nil {
		logger.Warnf("Error creating .p2p file: %v", err)
		return err
	}
	defer p2pFile.Close()
	fileHash := parserFile.FileHash
	fileSize := parserFile.FileSize
	maxChunkSize := parserFile.MaxChunkSize
	totalChunks := parserFile.TotalChunks

	p2pFile.WriteString("fileHash:" + fileHash + "\n")
	p2pFile.WriteString("fileSize:" + fileSize + "\n")
	p2pFile.WriteString("maxChunkSize:" + maxChunkSize + "\n")
	p2pFile.WriteString("totalChunks:" + totalChunks + "\n")

	for _, chunk := range parserFile.Chunks {
		p2pFile.WriteString(chunk.ChunkIndex + ":" + chunk.ChunkHash + "|" + chunk.ChunkSize + "\n")
	}

	return nil
}
