package cmd

import (
	"crypto/sha256"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"github.com/rudransh-shrivastava/peer-it/internal/client/client"
	"github.com/rudransh-shrivastava/peer-it/internal/shared/protocol"
	"github.com/rudransh-shrivastava/peer-it/internal/shared/schema"
	"github.com/rudransh-shrivastava/peer-it/internal/shared/utils"
	"github.com/spf13/cobra"
)

const maxChunkSize = 256 * 1024

// Chunks the file into 256kb chunks, generates checksums for each chunk and stores them in the database
var registerCmd = &cobra.Command{
	Use:   "register path/to/file ipc-socket-index",
	Short: "register a file to the tracker server",
	Long:  `registers a file to the tracker server to make it available for download by other peers.`,
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		filePath := args[0]
		socketIndex := args[1]
		client, err := client.NewClient(socketIndex)
		if err != nil {
			log.Fatal(err)
			return
		}
		file, err := os.Open(filePath)
		if err != nil {
			log.Println(err)
			return
		}
		defer file.Close()

		fileName := strings.Split(file.Name(), "/")[len(strings.Split(file.Name(), "/"))-1]
		fileInfo, err := file.Stat()
		if err != nil {
			log.Fatal(err)
			return
		}
		fileSize := fileInfo.Size()
		fileTotalChunks := int((fileSize + maxChunkSize - 1) / maxChunkSize)
		fileHash := sha256.New()
		if _, err := io.Copy(fileHash, file); err != nil {
			log.Fatal(err)
			return
		}
		fileChecksum := fmt.Sprintf("%x", fileHash.Sum(nil))

		schemaFile := schema.File{
			Size:         fileSize,
			MaxChunkSize: maxChunkSize,
			TotalChunks:  fileTotalChunks,
			Checksum:     fileChecksum,
			CreatedAt:    time.Now().Unix(),
		}
		// log the stats of the file
		log.Printf("file: %s with size %d and checksum %s\n", fileName, fileSize, fileChecksum)
		created, err := client.FileStore.CreateFile(&schemaFile)
		if !created {
			log.Println("file already exists")
			return
		}
		if err != nil {
			log.Fatal(err)
			return
		}

		log.Printf("Attempting to create chunks with metadata...")

		buffer := make([]byte, maxChunkSize)
		chunkIndex := 0
		file.Seek(0, 0)
		for {
			n, err := file.Read(buffer)
			if err != nil && err != io.EOF {
				log.Println(err)
				return
			}
			if n == 0 {
				break
			}

			checksum := generateChecksum(buffer[:n])
			err = client.ChunkStore.CreateChunk(&schemaFile, n, chunkIndex, checksum, false)
			if err != nil {
				log.Fatal(err)
				return
			}
			log.Printf("chunk %d: %s with size %d\n", chunkIndex, checksum, n)
			chunkIndex++
		}

		// copy the file to downlaoads/complete
		createdFile, err := os.Create("downloads/complete/" + fileName)
		if err != nil {
			log.Fatal(err)
			return
		}
		defer createdFile.Close()

		_, err = io.Copy(createdFile, file)
		if err != nil {
			log.Fatal(err)
			return
		}

		// send announce message to tracker to tell it to add the file
		log.Printf("Preparing to send Announce message to Tracker with newly created file")

		fileInfoMsgs := make([]*protocol.FileInfo, 0)
		fileInfoMsgs = append(fileInfoMsgs, &protocol.FileInfo{
			FileSize:    schemaFile.Size,
			ChunkSize:   int32(schemaFile.MaxChunkSize),
			FileHash:    schemaFile.Checksum,
			TotalChunks: int32(schemaFile.TotalChunks),
		})

		announceMsg := &protocol.AnnounceMessage{
			Files: fileInfoMsgs,
		}
		err = utils.SendAnnounceMsg(client.DaemonConn, announceMsg)
		if err != nil {
			log.Fatal(err)
			return
		}
		log.Printf("Registered %s successfully with the Tracker \n", filePath)
	},
}

func generateChecksum(data []byte) string {
	hash := sha256.New()
	hash.Write(data)
	return fmt.Sprintf("%x", hash.Sum(nil))
}
