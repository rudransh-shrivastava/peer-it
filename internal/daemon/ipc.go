package daemon

import (
	"crypto/sha256"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/pion/webrtc/v3"
	"github.com/rudransh-shrivastava/peer-it/internal/parser"
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

		channels := Channels{
			LogCh:     make(chan *protocol.NetworkMessage_Log, 100),
			GoodbyeCh: make(chan *protocol.NetworkMessage_Goodbye, 100),
		}
		d.Channels[cliConn.RemoteAddr().String()] = channels
		d.CLIRouter = cliRouter
		go cliRouter.Start()
		go d.handleCLIMsgs(cliRouter)
	}
}

func (d *Daemon) handleCLIMsgs(cliRouter *prouter.MessageRouter) {
	cliAddr := cliRouter.Conn.RemoteAddr().String()
	for {
		select {
		case <-d.Ctx.Done():
			d.Logger.Info("Stopping the CLI message listener")
			return
		case downloadSignal := <-d.CLISignalDownloadCh:
			d.Logger.Debugf("Received a new Download Signal from %s: %v", cliAddr, downloadSignal)
			// Create an empty entry in db and send a register msg to tracker
			// Also create an empty file with 0 bytes written but same size as file we want to download
			// Question is how will we know the size and the amount of chunks in the file we want to download
			// One solution is to have a .pit file or .p2p file with all this info kinda like a .torrent file
			// So, we will have to parse the file and get the info from it
			filePath := downloadSignal.SignalDownload.GetFilePath()
			d.Logger.Infof("File path to parse and start downloading: %s", filePath)

			parserFile, err := parser.ParseP2PFile(filePath)
			if err != nil {
				d.Logger.Warn(err)
				continue
			}
			d.Logger.Debugf("Parsed file: %+v", parserFile)
			// if file already exists in the db, we should not create it again
			// we should just start the download

			exists := false
			_, err = d.FileStore.GetFileByHash(parserFile.FileHash)
			if err == nil {
				exists = true
			}

			// 1. create the file in db
			// 2. create an empty file with the same size as the file we want to download
			// 1.
			created := false
			fileSize, err := strconv.Atoi(parserFile.FileSize)
			if err != nil {
				d.Logger.Warnf("Error converting file size to int: %v", err)
				continue
			}
			fileTotalChunks, err := strconv.Atoi(parserFile.TotalChunks)
			if err != nil {
				d.Logger.Warnf("Error converting total chunks to int: %v", err)
				continue
			}
			// send total chunks to CLI
			fileHash := parserFile.FileHash
			d.messageCLI(fmt.Sprintf("+%d", fileTotalChunks))
			schemaFile := schema.File{
				Name:         parserFile.FileName,
				Size:         int64(fileSize),
				MaxChunkSize: maxChunkSize,
				TotalChunks:  fileTotalChunks,
				Hash:         fileHash,
				CreatedAt:    time.Now().Unix(),
			}
			// initialize the peer chunk map for the file
			d.mu.Lock()
			if _, exists := d.PeerChunkMap[d.ID]; !exists {
				d.PeerChunkMap[d.ID] = make(map[string][]int32)
			}
			d.PeerChunkMap[d.ID][fileHash] = make([]int32, fileTotalChunks)
			d.mu.Unlock()
			// log the stats of the file
			d.Logger.Debugf("file: %s with size %d and hash %s\n", parserFile.FileName, fileSize, fileHash)
			if !exists {
				created, err = d.FileStore.CreateFile(&schemaFile)
				if err != nil {
					d.Logger.Warnf("Error creating file: %v", err)
					return
				}
				for _, chunk := range parserFile.Chunks {
					chunkIndex, err := strconv.Atoi(chunk.ChunkIndex)
					if err != nil {
						d.Logger.Warnf("Error converting chunk index to int: %v", err)
						continue
					}
					chunkSize, err := strconv.Atoi(chunk.ChunkSize)
					if err != nil {
						d.Logger.Warnf("Error converting chunk size to int: %v", err)
						continue
					}
					err = d.ChunkStore.CreateChunk(&schemaFile, chunkSize, chunkIndex, chunk.ChunkHash, false)
					if err != nil {
						d.Logger.Warnf("Error creating chunk: %v", err)
						return
					}
					d.Logger.Debugf("chunk %d: %s with size %d\n", chunkIndex, chunk.ChunkHash, chunkSize)
				}
			}
			// if file was created means it was a fresh file and we do not have any chunks in the ChunkStore
			// in this case we probably need to initialize our map with 0's
			// if file was not created that means we already have some chunks in the ChunkStore
			if created {
				d.mu.Lock()
				// Ensure `d.PeerChunkMap[d.ID]` is initialized
				if _, exists := d.PeerChunkMap[d.ID]; !exists {
					d.PeerChunkMap[d.ID] = make(map[string][]int32)
				}
				d.PeerChunkMap[d.ID][fileHash] = make([]int32, fileTotalChunks)
				d.mu.Unlock()
			}
			// 2.
			sizeInBytes := int64(fileSize) / 8
			downloadDirPath := fmt.Sprintf("downloads/daemon-%s/", d.IPCSocketIndex)
			err = os.MkdirAll(downloadDirPath, os.ModePerm)
			if err != nil {
				d.Logger.Warnf("Error creating directory: %v", err)
				return
			}
			if created {
				file, err := os.Create(downloadDirPath + parserFile.FileName)
				if err != nil {
					d.Logger.Warnf("Error creating file: %v", err)
					return
				}
				defer file.Close()
				// Seek to the desired file size and write a single zero byte
				_, err = file.Seek(int64(sizeInBytes-1), 0) // Move to size-1 position
				if err != nil {
					d.Logger.Warnf("Error seeking file: %v", err)
					return
				}

				_, err = file.Write([]byte{0}) // Write a single null byte to allocate space
				if err != nil {
					d.Logger.Warnf("Error writing to file: %v", err)
					return
				}
				d.Logger.Infof("Created an empty file with size %d or %d bytes", fileSize, sizeInBytes)
			}

			peerListReqMsg := &protocol.NetworkMessage{
				MessageType: &protocol.NetworkMessage_PeerListRequest{
					PeerListRequest: &protocol.PeerListRequest{
						FileHash: fileHash,
					},
				},
			}
			// Request a peer list for the file we want to download
			channel, exists := d.PendingPeerListRequests[fileHash]
			if !exists {
				channel = make(chan *protocol.NetworkMessage_PeerListResponse, 100)
				d.PendingPeerListRequests[fileHash] = channel
			}

			err = d.TrackerRouter.WriteMessage(peerListReqMsg)
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
						d.Logger.Debugf("Therefore, sending an introduction message for file: %s", fileHash)
						err := d.sendIntroduction(peerID, fileHash)
						if err != nil {
							d.Logger.Warnf("Failed to send introduction: %v", err)
						}
						continue
					}
					d.Logger.Debugf("Setting up WebRTC connection with peer: %s", peerID)

					err := d.handleWebRTCConnection(peerID, fileHash, config, true)
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
				// start file download

				d.Logger.Infof("Starting file download for file %s", schemaFile.Name)
				d.startFileDownload(fileHash, fileTotalChunks)
			case <-time.After(10 * time.Second):
				d.Logger.Warn("Timeout waiting for peer list response")
				continue
			}

		case registerSignal := <-d.CLISignalRegisterCh:
			filePath := registerSignal.SignalRegister.GetFilePath()
			d.Logger.Debugf("Received a new Register Signal from %s: %v", cliAddr, registerSignal)
			// Open the file and get its info
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
				Name:         fileName,
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

			parserFile := &parser.ParserFile{
				FileName:     fileName,
				FileHash:     fileHash,
				FileSize:     fmt.Sprintf("%d", fileSize),
				MaxChunkSize: fmt.Sprintf("%d", maxChunkSize),
				TotalChunks:  fmt.Sprintf("%d", fileTotalChunks),
				Chunks:       make([]parser.ParserChunk, 0),
			}
			buffer := make([]byte, maxChunkSize)
			chunkIndex := 0
			file.Seek(0, 0)
			for {
				chunkSize, err := file.Read(buffer)
				if err != nil && err != io.EOF {
					d.Logger.Warnf("Error reading buffer: %v", err)
					return
				}
				if chunkSize == 0 {
					break
				}

				hash := utils.GenerateHash(buffer[:chunkSize])
				err = d.ChunkStore.CreateChunk(&schemaFile, chunkSize, chunkIndex, hash, true)
				if err != nil {
					d.Logger.Warnf("Error creating chunk: %v", err)
					return
				}
				d.Logger.Debugf("chunk %d: %s with size %d\n", chunkIndex, hash, chunkSize)
				chunk := parser.ParserChunk{
					ChunkIndex: fmt.Sprintf("%d", chunkIndex),
					ChunkHash:  hash,
					ChunkSize:  fmt.Sprintf("%d", chunkSize),
				}
				parserFile.Chunks = append(parserFile.Chunks, chunk)
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

			// Create a .p2p file with the file metadata
			err = parser.GenerateP2PFile(parserFile, downloadDirPath+fileName+".p2p")
			if err != nil {
				d.Logger.Warnf("Error generating .p2p file: %v", err)
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
