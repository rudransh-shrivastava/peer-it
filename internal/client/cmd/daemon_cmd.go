package cmd

import (
	"context"

	"github.com/rudransh-shrivastava/peer-it/internal/node"
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
		log := logger.NewLogger()
		log.Debugf("IPC Socket Index: %s", ipcSocketIndex)
		log.Debugf("Tracker Address: %s", trackerAddr)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		n, err := node.New(ctx, node.Options{
			TrackerAddr:    trackerAddr,
			IPCSocketIndex: ipcSocketIndex,
			Logger:         log,
		})
		if err != nil {
			log.Fatal(err)
			return
		}
		n.Start()
	},
}
