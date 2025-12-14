package node

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"os"

	"github.com/rudransh-shrivastava/peer-it/internal/parser"
	"github.com/rudransh-shrivastava/peer-it/internal/shared/protocol"
	"github.com/rudransh-shrivastava/peer-it/internal/shared/utils"
)

func (n *Node) handleRegisterSignal(ctx context.Context, cliAddr string, registerSignal *protocol.NetworkMessage_SignalRegister) {
	filePath := registerSignal.SignalRegister.GetFilePath()
	n.logger.Debugf("Received a new Register Signal from %s: %v", cliAddr, registerSignal)

	file, err := os.Open(filePath)
	if err != nil {
		n.logger.Warnf("Error opening file: %v", err)
		return
	}
	defer func() { _ = file.Close() }()

	fileName := ExtractFileName(file.Name())
	fileInfo, err := file.Stat()
	if err != nil {
		n.logger.Warnf("Error getting file info: %v", err)
		return
	}
	fileSize := fileInfo.Size()
	fileTotalChunks := CalculateTotalChunks(fileSize, maxChunkSize)

	fileHash, err := HashFile(file)
	if err != nil {
		n.logger.Warnf("Error hashing file: %v", err)
		return
	}

	n.logger.Debugf("file: %s with size %d and hash %s", fileName, fileSize, fileHash)

	dbFile, created, err := n.files.CreateFile(ctx,
		fileName,
		fileSize,
		maxChunkSize,
		fileTotalChunks,
		fileHash,
	)
	if err != nil && err != sql.ErrNoRows {
		n.logger.Warnf("Error creating file: %v", err)
		return
	}
	if !created {
		n.logger.Info("file already registered with tracker")
		return
	}

	n.logger.Info("Attempting to create chunks with metadata...")

	parserFile := &parser.ParserFile{
		FileName:     fileName,
		FileHash:     fileHash,
		FileSize:     fmt.Sprintf("%d", fileSize),
		MaxChunkSize: fmt.Sprintf("%d", maxChunkSize),
		TotalChunks:  fmt.Sprintf("%d", fileTotalChunks),
		Chunks:       make([]parser.ParserChunk, 0),
	}

	buffer := make([]byte, maxChunkSize)
	chunkIndex := 0
	if _, err := file.Seek(0, 0); err != nil {
		n.logger.Warnf("Error seeking to start of file: %v", err)
		return
	}

	for {
		chunkSize, readErr := file.Read(buffer)
		if readErr != nil && readErr != io.EOF {
			n.logger.Warnf("Error reading buffer: %v", readErr)
			return
		}
		if chunkSize == 0 {
			break
		}

		chunkHash := utils.GenerateHash(buffer[:chunkSize])
		if _, chunkErr := n.chunks.CreateChunk(ctx, dbFile.ID, chunkIndex, chunkSize, chunkHash, true); chunkErr != nil {
			n.logger.Warnf("Error creating chunk: %v", chunkErr)
			return
		}
		n.logger.Debugf("chunk %d: %s with size %d", chunkIndex, chunkHash, chunkSize)
		parserFile.Chunks = append(parserFile.Chunks, parser.ParserChunk{
			ChunkIndex: fmt.Sprintf("%d", chunkIndex),
			ChunkHash:  chunkHash,
			ChunkSize:  fmt.Sprintf("%d", chunkSize),
		})
		chunkIndex++
	}

	downloadDirPath := fmt.Sprintf("downloads/daemon-%s/", n.ipcSocketIndex)
	if err := os.MkdirAll(downloadDirPath, os.ModePerm); err != nil {
		n.logger.Warnf("Error creating directory: %v", err)
		return
	}

	createdFile, err := os.Create(downloadDirPath + fileName)
	if err != nil {
		n.logger.Warnf("Error creating file: %v", err)
		return
	}
	defer func() { _ = createdFile.Close() }()

	if _, err := file.Seek(0, 0); err != nil {
		n.logger.Warnf("Error seeking to start of file: %v", err)
		return
	}
	if _, err = io.Copy(createdFile, file); err != nil {
		n.logger.Warnf("Error copying file: %v", err)
		return
	}

	if err := parser.GenerateP2PFile(parserFile, downloadDirPath+fileName+".p2p"); err != nil {
		n.logger.Warnf("Error generating .p2p file: %v", err)
		return
	}

	fileInfoMsgs := []*protocol.FileInfo{
		{
			FileSize:    dbFile.Size,
			ChunkSize:   int32(dbFile.MaxChunkSize),
			FileHash:    dbFile.Hash,
			TotalChunks: int32(dbFile.TotalChunks),
		},
	}
	n.logger.Infof("Preparing to send Announce message to Tracker with newly created file: %v", fileInfoMsgs)

	if err := n.trackerRouter.WriteMessage(BuildAnnounceMessage(fileInfoMsgs)); err != nil {
		n.logger.Warnf("Error sending announce message: %v", err)
		return
	}
	n.logger.Info("Successfully registered a new file with tracker")
}
