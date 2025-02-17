package cmd

import (
	"fmt"
	"strings"

	"github.com/rudransh-shrivastava/peer-it/internal/client/client"
	"github.com/rudransh-shrivastava/peer-it/internal/shared/utils/logger"
	"github.com/spf13/cobra"
)

// Chunks the file into 32KB chunks, generates hashes for each chunk and stores them in the database
var registerCmd = &cobra.Command{
	Use:   "register path/to/file ipc-socket-index",
	Short: "register a file to the tracker server and generates a .p2p file",
	Long: `registers a file to the tracker server to make it available for download by other peers.
			It also generates a .p2p file which contains the metadata of the file`,
	Args: cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		filePath := args[0]
		socketIndex := args[1]
		logger := logger.NewLogger()
		client, err := client.NewClient(socketIndex, logger)
		if err != nil {
			logger.Fatal(err)
			return
		}
		err = client.SendRegisterSignal(filePath)
		if err != nil {
			logger.Fatal(err)
			return
		}
		fileName := strings.Split(filePath, "/")[len(strings.Split(filePath, "/"))-1]
		logger.Info("Successfully registered file with tracker")
		fullFilePath := fmt.Sprintf("downloads/daemon-%s/%s.pit", socketIndex, fileName)
		logger.Infof("Created a .p2p file: %s", fullFilePath)
	},
}
