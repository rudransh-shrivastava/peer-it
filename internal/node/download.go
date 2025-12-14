package node

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/rudransh-shrivastava/peer-it/internal/parser"
	"github.com/rudransh-shrivastava/peer-it/internal/shared/protocol"
)

func (n *Node) handleDownloadSignal(ctx context.Context, cliAddr string, downloadSignal *protocol.NetworkMessage_SignalDownload) {
	n.logger.Debugf("Received a new Download Signal from %s: %v", cliAddr, downloadSignal)
	filePath := downloadSignal.SignalDownload.GetFilePath()
	n.logger.Infof("File path to parse and start downloading: %s", filePath)

	parserFile, err := parser.ParseP2PFile(filePath)
	if err != nil {
		n.logger.Warn(err)
		return
	}
	n.logger.Debugf("Parsed file: %+v", parserFile)

	_, err = n.files.GetFileByHash(ctx, parserFile.FileHash)
	fileExists := err == nil

	fileSize, err := strconv.Atoi(parserFile.FileSize)
	if err != nil {
		n.logger.Warnf("Error converting file size to int: %v", err)
		return
	}
	fileTotalChunks, err := strconv.Atoi(parserFile.TotalChunks)
	if err != nil {
		n.logger.Warnf("Error converting total chunks to int: %v", err)
		return
	}

	fileHash := parserFile.FileHash
	n.messageCLI(fmt.Sprintf("+%d", fileTotalChunks))

	n.mu.Lock()
	if _, exists := n.peerChunkMap[n.id]; !exists {
		n.peerChunkMap[n.id] = make(map[string][]int32)
	}
	n.peerChunkMap[n.id][fileHash] = make([]int32, fileTotalChunks)
	n.mu.Unlock()

	n.logger.Debugf("file: %s with size %d and hash %s", parserFile.FileName, fileSize, fileHash)

	var createdFile bool
	if !fileExists {
		dbFile, created, createErr := n.files.CreateFile(ctx,
			parserFile.FileName,
			int64(fileSize),
			maxChunkSize,
			fileTotalChunks,
			fileHash,
		)
		if createErr != nil {
			n.logger.Warnf("Error creating file: %v", createErr)
			return
		}
		createdFile = created

		if created {
			for _, chunk := range parserFile.Chunks {
				chunkIndex, parseErr := strconv.Atoi(chunk.ChunkIndex)
				if parseErr != nil {
					n.logger.Warnf("Error converting chunk index to int: %v", parseErr)
					continue
				}
				chunkSize, parseErr := strconv.Atoi(chunk.ChunkSize)
				if parseErr != nil {
					n.logger.Warnf("Error converting chunk size to int: %v", parseErr)
					continue
				}
				_, chunkErr := n.chunks.CreateChunk(ctx, dbFile.ID, chunkIndex, chunkSize, chunk.ChunkHash, false)
				if chunkErr != nil {
					n.logger.Warnf("Error creating chunk: %v", chunkErr)
					return
				}
				n.logger.Debugf("chunk %d: %s with size %d", chunkIndex, chunk.ChunkHash, chunkSize)
			}
		}
	}

	if createdFile {
		n.mu.Lock()
		if _, exists := n.peerChunkMap[n.id]; !exists {
			n.peerChunkMap[n.id] = make(map[string][]int32)
		}
		n.peerChunkMap[n.id][fileHash] = make([]int32, fileTotalChunks)
		n.mu.Unlock()
	}

	downloadDirPath := fmt.Sprintf("downloads/daemon-%s/", n.ipcSocketIndex)
	if err := os.MkdirAll(downloadDirPath, os.ModePerm); err != nil {
		n.logger.Warnf("Error creating directory: %v", err)
		return
	}

	if createdFile {
		sizeInBytes := int64(fileSize) / 8
		if err := CreatePreallocatedFile(downloadDirPath+parserFile.FileName, sizeInBytes); err != nil {
			n.logger.Warnf("Error creating preallocated file: %v", err)
			return
		}
		n.logger.Infof("Created an empty file with size %d or %d bytes", fileSize, sizeInBytes)
	}

	channel, exists := n.pendingPeerListRequests[fileHash]
	if !exists {
		channel = make(chan *protocol.NetworkMessage_PeerListResponse, 100)
		n.pendingPeerListRequests[fileHash] = channel
	}

	if err := n.trackerRouter.WriteMessage(BuildPeerListRequestMessage(fileHash)); err != nil {
		n.logger.Warnf("Error sending message to Tracker: %v", err)
	}

	config := DefaultSTUNConfig()

	select {
	case peerListResponse := <-channel:
		n.logger.Debugf("Received peer list response from tracker using a channel: %v", peerListResponse)
		if fileHash != peerListResponse.PeerListResponse.GetFileHash() {
			n.logger.Warnf("Response file hash does not match requested file hash")
			return
		}
		peers := peerListResponse.PeerListResponse.GetPeers()
		n.logger.Debugf("Received peer list %+v", peers)
		for _, peer := range peers {
			peerID := peer.GetId()
			if _, peerExists := n.peerConnections[peerID]; peerExists {
				n.logger.Debugf("Already connected to peer: %s", peerID)
				if introErr := n.sendIntroduction(peerID, fileHash); introErr != nil {
					n.logger.Warnf("Failed to send introduction: %v", introErr)
				}
				continue
			}
			n.logger.Debugf("Setting up WebRTC connection with peer: %s", peerID)

			if webrtcErr := n.handleWebRTCConnection(peerID, fileHash, config, true); webrtcErr != nil {
				n.logger.Warnf("Failed to setup WebRTC connection: %v", webrtcErr)
				continue
			}

			peerConnection := n.peerConnections[peerID]
			offer, offerErr := peerConnection.CreateOffer(nil)
			if offerErr != nil {
				n.logger.Warnf("Failed to create offer: %v", offerErr)
				continue
			}

			if sdpErr := peerConnection.SetLocalDescription(offer); sdpErr != nil {
				n.logger.Warnf("Failed to set local description: %v", sdpErr)
				continue
			}

			if sigErr := n.trackerRouter.WriteMessage(BuildSignalingOfferMessage(peerID, offer.SDP)); sigErr != nil {
				n.logger.Warnf("Failed to send offer: %v", sigErr)
			}
		}

		n.logger.Infof("Starting file download for file %s", parserFile.FileName)
		n.startFileDownload(fileHash, fileTotalChunks)

	case <-time.After(10 * time.Second):
		n.logger.Warn("Timeout waiting for peer list response")
	}
}

func CreatePreallocatedFile(path string, size int64) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()

	if size > 0 {
		if _, err = file.Seek(size-1, 0); err != nil {
			return err
		}
		if _, err = file.Write([]byte{0}); err != nil {
			return err
		}
	}
	return nil
}
