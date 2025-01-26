package cmd

import (
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rudransh-shrivastava/peer-it/internal/shared/protocol"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/proto"
)

var rootCmd = &cobra.Command{
	Use:  `peer-it`,
	Long: `peer-it is a peer to peer file transfer application`,
	Run: func(cmd *cobra.Command, args []string) {
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

		done := make(chan os.Signal, 1)
		signal.Notify(done, syscall.SIGINT, syscall.SIGTERM)

		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				hb := &protocol.HeartbeatMessage{
					Timestamp: time.Now().Unix(),
				}

				netMsg := &protocol.NetworkMessage{
					MessageType: &protocol.NetworkMessage_Heartbeat{
						Heartbeat: hb,
					},
				}

				data, err := proto.Marshal(netMsg)
				if err != nil {
					fmt.Println("Error marshalling message:", err)
					return
				}
				msgLen := uint32(len(data))
				if err := binary.Write(conn, binary.BigEndian, msgLen); err != nil {
					log.Printf("Error sending message length: %v", err)
					return
				}

				if _, err := conn.Write(data); err != nil {
					log.Printf("Error sending message: %v", err)
					return
				}
				time.Sleep(2 * time.Second)
			case <-done:
				fmt.Println("exiting...")
				return
			}
		}

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
