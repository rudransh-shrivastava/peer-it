package cmd

import (
	"context"

	"github.com/rudransh-shrivastava/peer-it/internal/node"
	"github.com/rudransh-shrivastava/peer-it/internal/shared/utils/logger"
	"github.com/spf13/cobra"
)

var daemonIndex string

var daemonCmd = &cobra.Command{
	Use:   "daemon <tracker-address>",
	Short: "runs peer-it daemon",
	Long:  `runs the peer-it daemon in the background, connecting to the specified tracker`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		trackerAddr := args[0]
		log := logger.NewLogger()
		log.Debugf("Index: %s, Tracker: %s", daemonIndex, trackerAddr)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		n, err := node.New(ctx, node.Options{
			TrackerAddr:    trackerAddr,
			IPCSocketIndex: daemonIndex,
			Logger:         log,
		})
		if err != nil {
			log.Fatal(err)
			return
		}
		n.Start()
	},
}

func init() {
	daemonCmd.Flags().StringVarP(&daemonIndex, "index", "i", "0", "daemon index for local testing (default: 0)")
}
