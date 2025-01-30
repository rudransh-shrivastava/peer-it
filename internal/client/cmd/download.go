package cmd

import (
	"log"

	"github.com/rudransh-shrivastava/peer-it/internal/client/client"
	"github.com/rudransh-shrivastava/peer-it/internal/shared/protocol"
	"github.com/rudransh-shrivastava/peer-it/internal/shared/utils"
	"github.com/spf13/cobra"
)

var downloadCmd = &cobra.Command{
	Use:   "download file_hash",
	Short: "download a file",
	Long:  `download a file, this requests the tracker server for the peers having the file`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		fileHash := args[0]
		client, err := client.NewClient()
		if err != nil {
			log.Println(err)
			return
		}
		peerListReqMsg := protocol.PeerListRequest{
			FileHash: fileHash,
		}
		err = utils.SendPeerListRequestMsg(client.DaemonConn, &peerListReqMsg)
		if err != nil {
			log.Fatal(err)
		}

		client.WaitForPeerList()
		log.Printf("Done waiting... now send the damn connection to the peers")
	},
}
