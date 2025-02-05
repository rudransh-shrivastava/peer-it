package cmd

import (
	"crypto/sha256"
	"fmt"

	"github.com/rudransh-shrivastava/peer-it/internal/client/client"
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
		err = client.SendRegisterSignal(filePath)
		if err != nil {
			logger.Fatal(err)
			return
		}
		logger.Info("Successfully registered file with tracker")
	},
}

func generateHash(data []byte) string {
	hash := sha256.New()
	hash.Write(data)
	return fmt.Sprintf("%x", hash.Sum(nil))
}
