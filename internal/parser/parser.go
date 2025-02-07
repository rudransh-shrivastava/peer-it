package parser

import (
	"bufio"
	"os"
	"strings"

	"github.com/rudransh-shrivastava/peer-it/internal/shared/utils/logger"
)

/*
	Format of the .p2p file
lineNo	 	string
0			fileHash:string
1			fileSize:int
2			maxChunkSize:int
3			totalChunks:int
4			0:chunkHash(string)|chunkSize(int)
5			1:chunkHash(string)|chunkSize(int)
n+4			n:chunkHash(string)|chunkSize(int)

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

func ParseP2PFile(path string) (*ParserFile, error) {
	logger := logger.NewLogger()
	p2pFile, err := os.Open(path)
	if err != nil {
		logger.Warnf("Error opening .p2p file: %v", err)
		return nil, err
	}
	defer p2pFile.Close()

	scanner := bufio.NewScanner(p2pFile)
	parserFile := &ParserFile{}
	parserFile.Chunks = make([]ParserChunk, 0)
	scanIndex := 0
	for scanner.Scan() {
		line := scanner.Text()
		if scanIndex == 0 {
			parserFile.FileHash = strings.Split(line, ":")[1]
		}
		if scanIndex == 1 {
			parserFile.FileSize = strings.Split(line, ":")[1]
		}
		if scanIndex == 2 {
			parserFile.MaxChunkSize = strings.Split(line, ":")[1]
		}
		if scanIndex == 3 {
			parserFile.TotalChunks = strings.Split(line, ":")[1]
		}

		if scanIndex > 3 {
			scanner.Scan()
			chunkLine := scanner.Text()

			chunkIndex := strings.Split(chunkLine, ":")[0]
			chunkData := strings.Split(strings.Split(chunkLine, ":")[1], "|")
			chunkHash := chunkData[0]
			chunkSize := chunkData[1]
			parserFile.Chunks = append(parserFile.Chunks, ParserChunk{
				ChunkIndex: chunkIndex,
				ChunkHash:  chunkHash,
				ChunkSize:  chunkSize,
			})
		}
		scanIndex++
	}

	return parserFile, nil
}
