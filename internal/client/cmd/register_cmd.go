package cmd

import (
	"strings"

	"github.com/rudransh-shrivastava/peer-it/internal/client/client"
	"github.com/rudransh-shrivastava/peer-it/internal/shared/utils/logger"
	"github.com/spf13/cobra"
)

var registerIndex string

var registerCmd = &cobra.Command{
	Use:   "register <path/to/file>",
	Short: "register a file to the tracker server",
	Long:  `registers a file to the tracker server to make it available for download by other peers`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		filePath := args[0]
		logger := logger.NewLogger()

		client, err := client.NewClient(registerIndex, logger)
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
		logger.Infof("File available at: downloads/daemon-%s/%s", registerIndex, fileName)
	},
}

func init() {
	registerCmd.Flags().StringVarP(&registerIndex, "index", "i", "0", "daemon index for local testing (default: 0)")
}
