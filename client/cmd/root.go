package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/gorilla/websocket"
	"github.com/rudransh-shrivastava/peer-it/client/db"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:  `peer-it`,
	Long: `peer-it is a peer to peer file transfer application`,
	Run: func(cmd *cobra.Command, args []string) {
		_, err := db.NewDB()
		if err != nil {
			fmt.Println(err)
			return
		}

		serverUrl := "ws://localhost:8080/"
		conn, _, err := websocket.DefaultDialer.Dial(serverUrl, nil)
		if err != nil {
			fmt.Println(err)
			return
		}
		defer conn.Close()

		fmt.Println("successfully connected to the web socket server")

		done := make(chan os.Signal, 1)
		signal.Notify(done, syscall.SIGINT, syscall.SIGTERM)

		<-done
		fmt.Println("exiting...")
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error while executing'%s'", err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(registerCmd)
}
