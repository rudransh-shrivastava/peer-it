package cmd

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/rudransh-shrivastava/peer-it/internal/client/client"
	"github.com/rudransh-shrivastava/peer-it/internal/shared/protocol"
	"github.com/rudransh-shrivastava/peer-it/internal/shared/schema"
	"github.com/rudransh-shrivastava/peer-it/internal/shared/utils"
	"github.com/rudransh-shrivastava/peer-it/internal/shared/utils/logger"
	"github.com/spf13/cobra"
)

const maxChunkSize = 256 * 1024

// Chunks the file into 256kb chunks, generates hashes for each chunk and stores them in the database
var registerCmd = &cobra.Command{
	Use:   "register path/to/file ipc-socket-index",
	Short: "register a file to the tracker server",
	Long:  `registers a file to the tracker server to make it available for download by other peers.`,
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		filePath := args[0]
		socketIndex := args[1]
		logger := logger.NewLogger()
		client, err := client.NewClient(socketIndex)
		if err != nil {
			logger.Fatal(err)
			return
		}
		file, err := os.Open(filePath)
		if err != nil {
			logger.Fatal(err)
			return
		}
		defer file.Close()

		fileName := strings.Split(file.Name(), "/")[len(strings.Split(file.Name(), "/"))-1]
		fileInfo, err := file.Stat()
		if err != nil {
			logger.Fatal(err)
			return
		}
		fileSize := fileInfo.Size()
		fileTotalChunks := int((fileSize + maxChunkSize - 1) / maxChunkSize)
		hash := sha256.New()
		if _, err := io.Copy(hash, file); err != nil {
			logger.Fatal(err)
			return
		}
		fileHash := fmt.Sprintf("%x", hash.Sum(nil))

		schemaFile := schema.File{
			Size:         fileSize,
			MaxChunkSize: maxChunkSize,
			TotalChunks:  fileTotalChunks,
			Hash:         fileHash,
			CreatedAt:    time.Now().Unix(),
		}
		// log the stats of the file
		logger.Debugf("file: %s with size %d and hash %s\n", fileName, fileSize, fileHash)
		created, err := client.FileStore.CreateFile(&schemaFile)
		if !created {
			logger.Info("file already registered with tracker")
			return
		}
		if err != nil {
			logger.Fatal(err)
			return
		}

		logger.Info("Attempting to create chunks with metadata...")

		buffer := make([]byte, maxChunkSize)
		chunkIndex := 0
		file.Seek(0, 0)
		for {
			n, err := file.Read(buffer)
			if err != nil && err != io.EOF {
				logger.Fatal(err)
				return
			}
			if n == 0 {
				break
			}

			hash := generateHash(buffer[:n])
			err = client.ChunkStore.CreateChunk(&schemaFile, n, chunkIndex, hash, false)
			if err != nil {
				logger.Fatal(err)
				return
			}
			logger.Debugf("chunk %d: %s with size %d\n", chunkIndex, hash, n)
			chunkIndex++
		}

		// copy the file to downlaoads/complete
		downloadDirPath := fmt.Sprintf("downloads/daemon-%s/", socketIndex)
		err = os.MkdirAll(downloadDirPath, os.ModePerm)
		if err != nil {
			logger.Fatal(err)
			return
		}

		createdFile, err := os.Create(downloadDirPath + fileName)
		if err != nil {
			logger.Fatal(err)
			return
		}
		defer createdFile.Close()

		file.Seek(0, 0)
		_, err = io.Copy(createdFile, file)
		if err != nil {
			logger.Fatal(err)
			return
		}

		// send announce message to tracker to tell it to add the file
		logger.Infof("Preparing to send Announce message to Tracker with newly created file")

		fileInfoMsgs := make([]*protocol.FileInfo, 0)
		fileInfoMsgs = append(fileInfoMsgs, &protocol.FileInfo{
			FileSize:    schemaFile.Size,
			ChunkSize:   int32(schemaFile.MaxChunkSize),
			FileHash:    schemaFile.Hash,
			TotalChunks: int32(schemaFile.TotalChunks),
		})

		announceMsg := &protocol.AnnounceMessage{
			Files: fileInfoMsgs,
		}
		err = utils.SendAnnounceMsg(client.DaemonConn, announceMsg)
		if err != nil {
			logger.Fatal(err)
			return
		}
		logger.Infof("Registered %s successfully with the Tracker \n", filePath)
	},
}

func generateHash(data []byte) string {
	hash := sha256.New()
	hash.Write(data)
	return fmt.Sprintf("%x", hash.Sum(nil))
}
