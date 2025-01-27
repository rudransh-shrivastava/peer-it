package cmd

import (
	"log"
	"net"
	"os"

	"github.com/spf13/cobra"
)

var downloadCmd = &cobra.Command{
	Use:   "download file_hash",
	Short: "download a file",
	Long:  `download a file, this requests the tracker server for the peers having the file`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// file_hash := args[0]

		// Send a PeerListRequest to tracker
		remoteAddr := "localhost:8080"

		raddr, err := net.ResolveTCPAddr("tcp", remoteAddr)
		if err != nil {
			log.Println("Error resolving remote address:", err)
			os.Exit(1)
		}
		laddr := &net.TCPAddr{
			IP:   net.ParseIP("0.0.0.0"),
			Port: 26098,
		}

		conn, err := net.DialTCP("tcp", laddr, raddr)
		if err != nil {
			log.Println("Error dialing:", err)
			os.Exit(1)
		}
		defer conn.Close()

		log.Println("Connected to", conn.RemoteAddr())

		// networkMsg peerList := &protocol.PeerListRequest{
		// 	FileHash: file_hash,
		// }

	},
}
