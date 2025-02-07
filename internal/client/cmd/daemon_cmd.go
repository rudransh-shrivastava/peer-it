package cmd

import (
	"context"

	"github.com/rudransh-shrivastava/peer-it/internal/daemon"
	"github.com/rudransh-shrivastava/peer-it/internal/shared/utils/logger"
	"github.com/spf13/cobra"
)

var daemonCmd = &cobra.Command{
	Use:   "daemon port ipc-socket-index remote-address",
	Short: "runs peer-it daemon",
	Long:  `runs the peer-it daemon in the background, the daemon uses unix sockets to communicate with the CLI`,
	Args:  cobra.ExactArgs(3),
	Run: func(cmd *cobra.Command, args []string) {
		daemonPort := args[0]
		daemonAddr := "localhost:8080"
		if daemonPort != "" {
			daemonAddr = "localhost" + ":" + daemonPort
		}
		logger := logger.NewLogger()
		logger.Debugf("Daemon port: %s", daemonPort)
		ipcSocketIndex := args[1]
		logger.Debugf("IPC Socket Index: %s", ipcSocketIndex)
		trackerAddr := args[2]
		logger.Debugf("Tracker Address: %s", trackerAddr)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		daemon, err := daemon.NewDaemon(ctx, trackerAddr, ipcSocketIndex, daemonAddr)
		if err != nil {
			logger.Fatal(err)
			return
		}
		daemon.Start()
	},
}
