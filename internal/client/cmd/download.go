package cmd

import (
	"log"
	"net"

	"github.com/rudransh-shrivastava/peer-it/internal/client/client"
	"github.com/rudransh-shrivastava/peer-it/internal/shared/protocol"
	"github.com/rudransh-shrivastava/peer-it/internal/shared/utils"
	"github.com/spf13/cobra"
)

var downloadCmd = &cobra.Command{
	Use:   "download file_hash ipc-socket-index",
	Short: "download a file",
	Long:  `download a file, this requests the tracker server for the peers having the file`,
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		fileHash := args[0]
		ipcSocketIndex := args[1]
		client, err := client.NewClient(ipcSocketIndex)
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

		peerListResponse := client.WaitForPeerList()
		if fileHash != peerListReqMsg.GetFileHash() {
			log.Fatal("Response file hash does not match requested file hash")
		}
		log.Printf("Received peer list %+v", peerListResponse.GetPeers())
		peers := peerListResponse.GetPeers()

		fileInfo, err := client.FileStore.GetFileByChecksum(fileHash)
		chunksMap := make([]int32, fileInfo.TotalChunks)

		fileChunks, err := client.FileStore.GetChunks(fileHash)
		if err != nil {
			log.Println(err)

		}
		for _, chunk := range fileChunks {
			chunksMap[chunk.Index] = 1
		}
		for _, peer := range peers {
			// Send Introduction msg to peer
			log.Printf("Sending Introduction msg to peer: %+v", peer)
			addr := peer.GetIpAddress() + ":" + string(peer.GetPort())
			conn, err := net.Dial("udp", addr)
			if err != nil {
				log.Println(err)
				continue
			}
			introductionMsg := protocol.IntroductionMessage{
				FileHash:  fileHash,
				ChunksMap: chunksMap,
			}

			log.Printf("Sending chunks map %+v to peer %s", chunksMap, addr)
			err = utils.SendIntroductionMsg(conn, &introductionMsg)
			if err != nil {
				log.Println(err)
			}
			log.Printf("Sent introduction message to peer %s: %+v", addr, chunksMap)
		}
		log.Printf("Tried all peers")
	},
}
