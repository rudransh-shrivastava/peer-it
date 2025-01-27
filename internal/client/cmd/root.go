package cmd

import (
	"log"
	"os"

	"github.com/spf13/cobra"
)

const heartbeatInterval = 5

var rootCmd = &cobra.Command{
	Use:  `peer-it`,
	Long: `peer-it is a peer to peer file transfer application`,
	Run: func(cmd *cobra.Command, args []string) {
		log.Println("peer-it is a peer to peer file transfer application")
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(registerCmd)
	rootCmd.AddCommand(downloadCmd)
	rootCmd.AddCommand(daemonCmd)
}
