package cmd

import (
	"net"

	"github.com/rudransh-shrivastava/peer-it/internal/client/client"
	"github.com/rudransh-shrivastava/peer-it/internal/shared/protocol"
	"github.com/rudransh-shrivastava/peer-it/internal/shared/utils"
	"github.com/rudransh-shrivastava/peer-it/internal/shared/utils/logger"
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
		logger := logger.NewLogger()
		client, err := client.NewClient(ipcSocketIndex, logger)
		if err != nil {
			logger.Fatal(err)
			return
		}
		peerListReqMsg := protocol.PeerListRequest{
			FileHash: fileHash,
		}
		err = utils.SendPeerListRequestMsg(client.DaemonConn, &peerListReqMsg)
		if err != nil {
			logger.Fatal(err)
		}

		peerListResponse := client.WaitForPeerList()
		if fileHash != peerListReqMsg.GetFileHash() {
			logger.Fatal("Response file hash does not match requested file hash")
		}
		logger.Debugf("Received peer list %+v", peerListResponse.GetPeers())
		peers := peerListResponse.GetPeers()

		fileInfo, err := client.FileStore.GetFileByHash(fileHash)
		chunksMap := make([]int32, fileInfo.TotalChunks)

		fileChunks, err := client.FileStore.GetChunks(fileHash)
		if err != nil {
			logger.Fatal(err)

		}
		for _, chunk := range fileChunks {
			chunksMap[chunk.Index] = 1
		}
		for _, peer := range peers {
			// Send Introduction msg to peer
			logger.Infof("Sending Introduction msg to peer: %+v", peer)
			var addr string
			if peer.GetIpAddress() == "::1" {
				addr = "localhost:" + peer.GetPort()
			} else {
				addr = peer.GetIpAddress() + ":" + string(peer.GetPort())
			}
			logger.Debugf("Resolved IP:PORT of peer: %s", addr)
			conn, err := net.Dial("tcp", addr)
			if err != nil {
				logger.Warn(err)
				continue
			}
			introductionMsg := protocol.IntroductionMessage{
				FileHash:  fileHash,
				ChunksMap: chunksMap,
			}

			logger.Debugf("Sending chunks map %+v to peer %s", chunksMap, addr)
			err = utils.SendIntroductionMsg(conn, &introductionMsg)
			if err != nil {
				logger.Warn(err)
			}
			logger.Debugf("Sent introduction message to peer %s: %+v", addr, chunksMap)
		}
		logger.Infof("Tried all peers")
	},
}
