package cmd

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

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
		remoteAddr := "localhost:8080"
		raddr, err := net.ResolveTCPAddr("tcp", remoteAddr)
		if err != nil {
			fmt.Println("Error resolving remote address:", err)
			os.Exit(1)
		}
		laddr := &net.TCPAddr{
			IP:   net.ParseIP("0.0.0.0"),
			Port: 26098,
		}

		conn, err := net.DialTCP("tcp", laddr, raddr)
		if err != nil {
			fmt.Println("Error dialing:", err)
			os.Exit(1)
		}
		defer conn.Close()

		fmt.Println("Connected to", conn.RemoteAddr())

		// Example: Send a simple message
		_, err = conn.Write([]byte("PING"))
		if err != nil {
			fmt.Println("Error sending PING:", err)
			return
		}

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
