package cmd

import (
	"os"

	"github.com/rudransh-shrivastava/peer-it/internal/logger"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:  `peer-it`,
	Long: `peer-it is a peer to peer file transfer application`,
	Run: func(cmd *cobra.Command, args []string) {
		log := logger.NewLogger()
		log.Info("peer-it is a peer to peer file transfer application")
	},
}

func Execute() {
	log := logger.NewLogger()
	if err := rootCmd.Execute(); err != nil {
		log.Error("Command failed", "error", err)
		os.Exit(1)
	}
}

func init() {
	// rootCmd.AddCommand(daemonCmd)
}
