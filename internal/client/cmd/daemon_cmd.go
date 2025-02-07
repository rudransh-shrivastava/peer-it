package cmd

import (
	"context"

	"github.com/rudransh-shrivastava/peer-it/internal/daemon"
	"github.com/rudransh-shrivastava/peer-it/internal/shared/utils/logger"
	"github.com/spf13/cobra"
)

var daemonCmd = &cobra.Command{
	Use:   "daemon index tracker-address",
	Short: "runs peer-it daemon",
	Long:  `runs the peer-it daemon in the background, the daemon uses unix sockets to communicate with the CLI`,
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		ipcSocketIndex := args[0]
		trackerAddr := args[1]
		logger := logger.NewLogger()
		logger.Debugf("IPC Socket Index: %s", ipcSocketIndex)
		logger.Debugf("Tracker Address: %s", trackerAddr)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		daemon, err := daemon.NewDaemon(ctx, trackerAddr, ipcSocketIndex)
		if err != nil {
			logger.Fatal(err)
			return
		}
		daemon.Start()
	},
}
