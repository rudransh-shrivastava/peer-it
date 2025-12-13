package parser

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/rudransh-shrivastava/peer-it/internal/shared/utils/logger"
)

/*
Format of the .p2p file

	lineNo		string
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
	ChunkHash  string
	ChunkIndex string
	ChunkSize  string
}

func GenerateP2PFile(parserFile *ParserFile, path string) error {
	p2pFile, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("creating .p2p file: %w", err)
	}
	defer p2pFile.Close()

	lines := []string{
		"fileName:" + parserFile.FileName,
		"fileHash:" + parserFile.FileHash,
		"fileSize:" + parserFile.FileSize,
		"maxChunkSize:" + parserFile.MaxChunkSize,
		"totalChunks:" + parserFile.TotalChunks,
	}

	for _, line := range lines {
		if _, err := p2pFile.WriteString(line + "\n"); err != nil {
			return fmt.Errorf("writing to .p2p file: %w", err)
		}
	}

	for _, chunk := range parserFile.Chunks {
		line := chunk.ChunkIndex + ":" + chunk.ChunkHash + "|" + chunk.ChunkSize + "\n"
		if _, err := p2pFile.WriteString(line); err != nil {
			return fmt.Errorf("writing chunk to .p2p file: %w", err)
		}
	}

	return nil
}

func ParseP2PFile(path string) (*ParserFile, error) {
	log := logger.NewLogger()
	p2pFile, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening .p2p file: %w", err)
	}
	defer p2pFile.Close()

	scanner := bufio.NewScanner(p2pFile)
	parserFile := &ParserFile{
		Chunks: make([]ParserChunk, 0),
	}

	scanIndex := 0
	for scanner.Scan() {
		line := scanner.Text()

		switch scanIndex {
		case 0:
			parserFile.FileName = strings.Split(line, ":")[1]
			log.Infof("Filename: %s", parserFile.FileName)
		case 1:
			parserFile.FileHash = strings.Split(line, ":")[1]
			log.Infof("FileHash: %s", parserFile.FileHash)
		case 2:
			parserFile.FileSize = strings.Split(line, ":")[1]
			log.Infof("FileSize: %s", parserFile.FileSize)
		case 3:
			parserFile.MaxChunkSize = strings.Split(line, ":")[1]
			log.Infof("MaxChunkSize: %s", parserFile.MaxChunkSize)
		case 4:
			parserFile.TotalChunks = strings.Split(line, ":")[1]
			log.Infof("TotalChunks: %s", parserFile.TotalChunks)
		default:
			log.Infof("Parsing chunk line: %s", line)
			chunkIndex := strings.Split(line, ":")[0]
			chunkData := strings.Split(line, ":")[1]
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
