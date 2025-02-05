package cmd

import (
	"github.com/rudransh-shrivastava/peer-it/internal/client/client"
	"github.com/rudransh-shrivastava/peer-it/internal/shared/utils/logger"
	"github.com/spf13/cobra"
)

var downloadCmd = &cobra.Command{
	Use:   "download file_hash ipc-socket-index",
	Short: "download a file",
	Long:  `download a file, this requests the tracker server for the peers having the file`,
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		fileHash := args[0]
		ipcSocketIndex := args[1]
		logger := logger.NewLogger()
		client, err := client.NewClient(ipcSocketIndex)
		if err != nil {
			logger.Fatal(err)
			return
		}
		err = client.SendDownloadSignal(fileHash)
		if err != nil {
			logger.Fatal(err)
			return
		}
		// handle other stuff here
	},
}
