package cmd

import (
	"github.com/rudransh-shrivastava/peer-it/internal/client/client"
	"github.com/rudransh-shrivastava/peer-it/internal/shared/utils/logger"
	"github.com/spf13/cobra"
)

var downloadCmd = &cobra.Command{
	Use:   "download file-path ipc-socket-index",
	Short: "download a file",
	Long:  `download a file from the swarm, requires a .pit file's path`,
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		filePath := args[0]
		ipcSocketIndex := args[1]
		logger := logger.NewLogger()
		client, err := client.NewClient(ipcSocketIndex)
		if err != nil {
			logger.Fatal(err)
			return
		}
		// Send the download signal to the daemon
		err = client.SendDownloadSignal(filePath)
		if err != nil {
			logger.Fatal(err)
			return
		}
		// handle other stuff here
	},
}
