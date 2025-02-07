package daemon

import (
	"crypto/sha256"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"time"

	"github.com/pion/webrtc/v3"
	"github.com/rudransh-shrivastava/peer-it/internal/shared/protocol"
	"github.com/rudransh-shrivastava/peer-it/internal/shared/prouter"
	"github.com/rudransh-shrivastava/peer-it/internal/shared/schema"
	"github.com/rudransh-shrivastava/peer-it/internal/shared/utils"
	"google.golang.org/protobuf/proto"
)

func (d *Daemon) startIPCServer() {
	socketUrl := "/tmp/pit-daemon-" + d.IPCSocketIndex + ".sock"
	os.Remove(socketUrl)

	l, err := net.Listen("unix", socketUrl)
	if err != nil {
		panic(err)
	}
	d.Logger.Info("IPC Server started successfuly")
	for {
		cliConn, err := l.Accept()
		d.Logger.Info("Accepted a new socket connection")
		if err != nil {
			continue
		}
		cliRouter := prouter.NewMessageRouter(cliConn)
		cliRouter.AddRoute(d.CLISignalRegisterCh, func(msg proto.Message) bool {
			_, ok := msg.(*protocol.NetworkMessage).MessageType.(*protocol.NetworkMessage_SignalRegister)
			return ok
		})
		cliRouter.AddRoute(d.CLISignalDownloadCh, func(msg proto.Message) bool {
			_, ok := msg.(*protocol.NetworkMessage).MessageType.(*protocol.NetworkMessage_SignalDownload)
			return ok
		})

		go cliRouter.Start()
		go d.handleCLIMsgs(cliRouter) // problem here // using multiple go routines for same channels
	}
}

func (d *Daemon) handleCLIMsgs(msgRouter *prouter.MessageRouter) {
	cliAddr := msgRouter.Conn.RemoteAddr().String()
	for {
		select {
		case <-d.Ctx.Done():
			d.Logger.Info("Stopping the CLI message listener")
			return
		case downloadSignal := <-d.CLISignalDownloadCh:
			d.Logger.Debugf("Received a new Download Signal from %s: %v", cliAddr, downloadSignal)
			fileHash := downloadSignal.SignalDownload.GetFileHash()
			peerListReqMsg := &protocol.NetworkMessage{
				MessageType: &protocol.NetworkMessage_PeerListRequest{
					PeerListRequest: &protocol.PeerListRequest{
						FileHash: fileHash,
					},
				},
			}
			channel, exists := d.PendingPeerListRequests[fileHash]
			if !exists {
				channel = make(chan *protocol.NetworkMessage_PeerListResponse, 100)
				d.PendingPeerListRequests[fileHash] = channel
			}

			err := d.TrackerRouter.WriteMessage(peerListReqMsg)
			if err != nil {
				d.Logger.Warnf("Error sending message to Tracker: %v", err)
			}
			// WebRTC configuration
			config := webrtc.Configuration{
				ICEServers: []webrtc.ICEServer{
					{
						URLs: []string{
							"stun:stun.l.google.com:19302",
							"stun:stun1.l.google.com:19302",
							"stun:stun2.l.google.com:19302",
							"stun:stun3.l.google.com:19302",
							"stun:stun4.l.google.com:19302",
						},
					},
				},
				ICETransportPolicy: webrtc.ICETransportPolicyAll,
			}

			select {
			case peerListResponse := <-channel:
				d.Logger.Debugf("Received peer list response from tracker using a channel: %v", peerListResponse)
				if fileHash != peerListResponse.PeerListResponse.GetFileHash() {
					d.Logger.Warnf("Response file hash does not match requested file hash, response: %s requested: %s",
						peerListResponse.PeerListResponse.GetFileHash(), fileHash)
					continue
				}
				peers := peerListResponse.PeerListResponse.GetPeers()
				d.Logger.Debugf("Received peer list %+v", peers)
				for _, peer := range peers {
					peerID := peer.GetId()
					if _, exists := d.PeerConnections[peerID]; exists {
						d.Logger.Debugf("Already connected to peer: %s", peerID)
						continue
					}
					d.Logger.Debugf("Setting up WebRTC connection with peer: %s", peerID)

					err := d.handleWebRTCConnection(peerID, config)
					if err != nil {
						d.Logger.Warnf("Failed to setup WebRTC connection: %v", err)
						continue
					}

					// Create and send offer
					peerConnection := d.PeerConnections[peerID]

					offer, err := peerConnection.CreateOffer(nil)
					if err != nil {
						d.Logger.Warnf("Failed to create offer: %v", err)
						continue
					}

					err = peerConnection.SetLocalDescription(offer)
					if err != nil {
						d.Logger.Warnf("Failed to set local description: %v", err)
						continue
					}

					// Send offer through tracker
					signalingMsg := &protocol.NetworkMessage{
						MessageType: &protocol.NetworkMessage_Signaling{
							Signaling: &protocol.SignalingMessage{
								TargetPeerId: peerID,
								Message: &protocol.SignalingMessage_Offer{
									Offer: &protocol.Offer{
										Sdp: offer.SDP,
									},
								},
							},
						},
					}
					err = d.TrackerRouter.WriteMessage(signalingMsg)
					if err != nil {
						d.Logger.Warnf("Failed to send offer: %v", err)
					}
				}

			case <-time.After(10 * time.Second):
				d.Logger.Warn("Timeout waiting for peer list response")
				continue
			}

		case registerSignal := <-d.CLISignalRegisterCh:
			filePath := registerSignal.SignalRegister.GetFilePath()
			d.Logger.Debugf("Received a new Register Signal from %s: %v", cliAddr, registerSignal)
			file, err := os.Open(filePath)
			if err != nil {
				d.Logger.Warnf("Error opening file: %v", err)
				return
			}
			defer file.Close()

			fileName := strings.Split(file.Name(), "/")[len(strings.Split(file.Name(), "/"))-1]
			fileInfo, err := file.Stat()
			if err != nil {
				d.Logger.Warnf("Error getting file info: %v", err)
				return
			}
			fileSize := fileInfo.Size()
			fileTotalChunks := int((fileSize + maxChunkSize - 1) / maxChunkSize)
			hash := sha256.New()
			if _, err := io.Copy(hash, file); err != nil {
				d.Logger.Warnf("Error copying file hash to hash: %v", err)
				return
			}
			fileHash := fmt.Sprintf("%x", hash.Sum(nil))

			schemaFile := schema.File{
				Size:         fileSize,
				MaxChunkSize: maxChunkSize,
				TotalChunks:  fileTotalChunks,
				Hash:         fileHash,
				CreatedAt:    time.Now().Unix(),
			}
			// log the stats of the file
			d.Logger.Debugf("file: %s with size %d and hash %s\n", fileName, fileSize, fileHash)
			created, err := d.FileStore.CreateFile(&schemaFile)
			if !created {
				d.Logger.Info("file already registered with tracker")
				return
			}
			if err != nil {
				d.Logger.Warnf("Error creating file: %v", err)
				return
			}

			d.Logger.Info("Attempting to create chunks with metadata...")

			buffer := make([]byte, maxChunkSize)
			chunkIndex := 0
			file.Seek(0, 0)
			for {
				n, err := file.Read(buffer)
				if err != nil && err != io.EOF {
					d.Logger.Warnf("Error reading buffer: %v", err)
					return
				}
				if n == 0 {
					break
				}

				hash := utils.GenerateHash(buffer[:n])
				err = d.ChunkStore.CreateChunk(&schemaFile, n, chunkIndex, hash, false)
				if err != nil {
					d.Logger.Warnf("Error creating chunk: %v", err)
					return
				}
				d.Logger.Debugf("chunk %d: %s with size %d\n", chunkIndex, hash, n)
				chunkIndex++
			}

			// copy the file to downlaoads/complete
			downloadDirPath := fmt.Sprintf("downloads/daemon-%s/", d.IPCSocketIndex)
			err = os.MkdirAll(downloadDirPath, os.ModePerm)
			if err != nil {
				d.Logger.Warnf("Error creating directory: %v", err)
				return
			}

			createdFile, err := os.Create(downloadDirPath + fileName)
			if err != nil {
				d.Logger.Warnf("Error creating file: %v", err)
				return
			}
			defer createdFile.Close()

			file.Seek(0, 0)
			_, err = io.Copy(createdFile, file)
			if err != nil {
				d.Logger.Warnf("Error copying file: %v", err)
				return
			}

			// send announce message to tracker to tell it to add the file
			fileInfoMsgs := make([]*protocol.FileInfo, 0)
			fileInfoMsgs = append(fileInfoMsgs, &protocol.FileInfo{
				FileSize:    schemaFile.Size,
				ChunkSize:   int32(schemaFile.MaxChunkSize),
				FileHash:    schemaFile.Hash,
				TotalChunks: int32(schemaFile.TotalChunks),
			})
			d.Logger.Infof("Preparing to send Announce message to Tracker with newly created file: %v", fileInfoMsgs)

			netMsg := &protocol.NetworkMessage{
				MessageType: &protocol.NetworkMessage_Announce{
					Announce: &protocol.AnnounceMessage{
						Files: fileInfoMsgs,
					},
				},
			}

			d.TrackerRouter.WriteMessage(netMsg)
			d.Logger.Info("Successfully registered a new file with tracker")
		}
	}
}
