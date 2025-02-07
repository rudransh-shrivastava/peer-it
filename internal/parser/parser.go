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
0			fileName:string
1			fileHash:string
2			fileSize:int
3			maxChunkSize:int
4			totalChunks:int
5			0:chunkHash(string)|chunkSize(int)
6			1:chunkHash(string)|chunkSize(int)
n+5			n:chunkHash(string)|chunkSize(int)

each line represents a key value pair where the key is separated from the value by a colon
except for the chunk lines, where the key is the index of the chunk
and the value is the chunkHash and chunkSize separated by a pipe character
each line is separated by a newline character
*/

type ParserFile struct {
	FileName     string
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
	fileName := parserFile.FileName
	fileHash := parserFile.FileHash
	fileSize := parserFile.FileSize
	maxChunkSize := parserFile.MaxChunkSize
	totalChunks := parserFile.TotalChunks

	p2pFile.WriteString("fileName:" + fileName + "\n")
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
			parserFile.FileName = strings.Split(line, ":")[1]
			logger.Infof("Filename: %s", parserFile.FileName)
		}
		if scanIndex == 1 {
			parserFile.FileHash = strings.Split(line, ":")[1]
			logger.Infof("FileHash: %s", parserFile.FileHash)
		}
		if scanIndex == 2 {
			parserFile.FileSize = strings.Split(line, ":")[1]
			logger.Infof("FileSize: %s", parserFile.FileSize)
		}
		if scanIndex == 3 {
			parserFile.MaxChunkSize = strings.Split(line, ":")[1]
			logger.Infof("MaxChunkSize: %s", parserFile.MaxChunkSize)
		}
		if scanIndex == 4 {
			parserFile.TotalChunks = strings.Split(line, ":")[1]
			logger.Infof("TotalChunks: %s", parserFile.TotalChunks)
		}

		if scanIndex > 4 {
			chunkLine := scanner.Text()

			logger.Infof("Parsing >4 %s", chunkLine)
			chunkIndex := strings.Split(chunkLine, ":")[0]
			chunkData := strings.Split(chunkLine, ":")[1]
			chunkHash := strings.Split(chunkData, "|")[0]
			chunkSize := strings.Split(chunkData, "|")[1]
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
