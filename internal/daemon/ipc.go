package daemon

import (
	"context"
	"crypto/sha256"
	"database/sql"
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
	"github.com/rudransh-shrivastava/peer-it/internal/shared/utils"
	"google.golang.org/protobuf/proto"
)

func (d *Daemon) startIPCServer() {
	socketURL := "/tmp/pit-daemon-" + d.IPCSocketIndex + ".sock"
	_ = os.Remove(socketURL)

	l, err := net.Listen("unix", socketURL)
	if err != nil {
		d.Logger.Fatalf("Failed to start IPC server: %v", err)
		return
	}
	d.Logger.Info("IPC Server started successfully")
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
	ctx := context.Background()
	cliAddr := cliRouter.Conn.RemoteAddr().String()
	for {
		select {
		case <-d.Ctx.Done():
			d.Logger.Info("Stopping the CLI message listener")
			return
		case downloadSignal := <-d.CLISignalDownloadCh:
			d.handleDownloadSignal(ctx, cliAddr, downloadSignal)

		case registerSignal := <-d.CLISignalRegisterCh:
			d.handleRegisterSignal(ctx, cliAddr, registerSignal)
		}
	}
}

func (d *Daemon) handleDownloadSignal(ctx context.Context, cliAddr string, downloadSignal *protocol.NetworkMessage_SignalDownload) {
	d.Logger.Debugf("Received a new Download Signal from %s: %v", cliAddr, downloadSignal)
	filePath := downloadSignal.SignalDownload.GetFilePath()
	d.Logger.Infof("File path to parse and start downloading: %s", filePath)

	parserFile, err := parser.ParseP2PFile(filePath)
	if err != nil {
		d.Logger.Warn(err)
		return
	}
	d.Logger.Debugf("Parsed file: %+v", parserFile)

	// Check if file already exists
	_, err = d.FileStore.GetFileByHash(ctx, parserFile.FileHash)
	fileExists := err == nil

	fileSize, err := strconv.Atoi(parserFile.FileSize)
	if err != nil {
		d.Logger.Warnf("Error converting file size to int: %v", err)
		return
	}
	fileTotalChunks, err := strconv.Atoi(parserFile.TotalChunks)
	if err != nil {
		d.Logger.Warnf("Error converting total chunks to int: %v", err)
		return
	}

	fileHash := parserFile.FileHash
	d.messageCLI(fmt.Sprintf("+%d", fileTotalChunks))

	// Initialize the peer chunk map for the file
	d.mu.Lock()
	if _, exists := d.PeerChunkMap[d.ID]; !exists {
		d.PeerChunkMap[d.ID] = make(map[string][]int32)
	}
	d.PeerChunkMap[d.ID][fileHash] = make([]int32, fileTotalChunks)
	d.mu.Unlock()

	d.Logger.Debugf("file: %s with size %d and hash %s", parserFile.FileName, fileSize, fileHash)

	var createdFile bool
	if !fileExists {
		dbFile, created, createErr := d.FileStore.CreateFile(ctx,
			parserFile.FileName,
			int64(fileSize),
			maxChunkSize,
			fileTotalChunks,
			fileHash,
		)
		if createErr != nil {
			d.Logger.Warnf("Error creating file: %v", createErr)
			return
		}
		createdFile = created

		if created {
			for _, chunk := range parserFile.Chunks {
				chunkIndex, parseErr := strconv.Atoi(chunk.ChunkIndex)
				if parseErr != nil {
					d.Logger.Warnf("Error converting chunk index to int: %v", parseErr)
					continue
				}
				chunkSize, parseErr := strconv.Atoi(chunk.ChunkSize)
				if parseErr != nil {
					d.Logger.Warnf("Error converting chunk size to int: %v", parseErr)
					continue
				}
				_, chunkErr := d.ChunkStore.CreateChunk(ctx, dbFile.ID, chunkIndex, chunkSize, chunk.ChunkHash, false)
				if chunkErr != nil {
					d.Logger.Warnf("Error creating chunk: %v", chunkErr)
					return
				}
				d.Logger.Debugf("chunk %d: %s with size %d", chunkIndex, chunk.ChunkHash, chunkSize)
			}
		}
	}

	if createdFile {
		d.mu.Lock()
		if _, exists := d.PeerChunkMap[d.ID]; !exists {
			d.PeerChunkMap[d.ID] = make(map[string][]int32)
		}
		d.PeerChunkMap[d.ID][fileHash] = make([]int32, fileTotalChunks)
		d.mu.Unlock()
	}

	sizeInBytes := int64(fileSize) / 8
	downloadDirPath := fmt.Sprintf("downloads/daemon-%s/", d.IPCSocketIndex)
	if err := os.MkdirAll(downloadDirPath, os.ModePerm); err != nil {
		d.Logger.Warnf("Error creating directory: %v", err)
		return
	}

	if createdFile {
		file, fileErr := os.Create(downloadDirPath + parserFile.FileName)
		if fileErr != nil {
			d.Logger.Warnf("Error creating file: %v", fileErr)
			return
		}
		defer file.Close()

		if _, err = file.Seek(sizeInBytes-1, 0); err != nil {
			d.Logger.Warnf("Error seeking file: %v", err)
			return
		}

		if _, err = file.Write([]byte{0}); err != nil {
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

	channel, exists := d.PendingPeerListRequests[fileHash]
	if !exists {
		channel = make(chan *protocol.NetworkMessage_PeerListResponse, 100)
		d.PendingPeerListRequests[fileHash] = channel
	}

	if err := d.TrackerRouter.WriteMessage(peerListReqMsg); err != nil {
		d.Logger.Warnf("Error sending message to Tracker: %v", err)
	}

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
			d.Logger.Warnf("Response file hash does not match requested file hash")
			return
		}
		peers := peerListResponse.PeerListResponse.GetPeers()
		d.Logger.Debugf("Received peer list %+v", peers)
		for _, peer := range peers {
			peerID := peer.GetId()
			if _, peerExists := d.PeerConnections[peerID]; peerExists {
				d.Logger.Debugf("Already connected to peer: %s", peerID)
				if introErr := d.sendIntroduction(peerID, fileHash); introErr != nil {
					d.Logger.Warnf("Failed to send introduction: %v", introErr)
				}
				continue
			}
			d.Logger.Debugf("Setting up WebRTC connection with peer: %s", peerID)

			if webrtcErr := d.handleWebRTCConnection(peerID, fileHash, config, true); webrtcErr != nil {
				d.Logger.Warnf("Failed to setup WebRTC connection: %v", webrtcErr)
				continue
			}

			peerConnection := d.PeerConnections[peerID]
			offer, offerErr := peerConnection.CreateOffer(nil)
			if offerErr != nil {
				d.Logger.Warnf("Failed to create offer: %v", offerErr)
				continue
			}

			if sdpErr := peerConnection.SetLocalDescription(offer); sdpErr != nil {
				d.Logger.Warnf("Failed to set local description: %v", sdpErr)
				continue
			}

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
			if sigErr := d.TrackerRouter.WriteMessage(signalingMsg); sigErr != nil {
				d.Logger.Warnf("Failed to send offer: %v", sigErr)
			}
		}

		d.Logger.Infof("Starting file download for file %s", parserFile.FileName)
		d.startFileDownload(fileHash, fileTotalChunks)

	case <-time.After(10 * time.Second):
		d.Logger.Warn("Timeout waiting for peer list response")
	}
}

func (d *Daemon) handleRegisterSignal(ctx context.Context, cliAddr string, registerSignal *protocol.NetworkMessage_SignalRegister) {
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

	d.Logger.Debugf("file: %s with size %d and hash %s", fileName, fileSize, fileHash)

	dbFile, created, err := d.FileStore.CreateFile(ctx,
		fileName,
		fileSize,
		maxChunkSize,
		fileTotalChunks,
		fileHash,
	)
	if err != nil && err != sql.ErrNoRows {
		d.Logger.Warnf("Error creating file: %v", err)
		return
	}
	if !created {
		d.Logger.Info("file already registered with tracker")
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
	if _, err := file.Seek(0, 0); err != nil {
		d.Logger.Warnf("Error seeking to start of file: %v", err)
		return
	}
	for {
		chunkSize, readErr := file.Read(buffer)
		if readErr != nil && readErr != io.EOF {
			d.Logger.Warnf("Error reading buffer: %v", readErr)
			return
		}
		if chunkSize == 0 {
			break
		}

		chunkHash := utils.GenerateHash(buffer[:chunkSize])
		if _, chunkErr := d.ChunkStore.CreateChunk(ctx, dbFile.ID, chunkIndex, chunkSize, chunkHash, true); chunkErr != nil {
			d.Logger.Warnf("Error creating chunk: %v", chunkErr)
			return
		}
		d.Logger.Debugf("chunk %d: %s with size %d", chunkIndex, chunkHash, chunkSize)
		parserFile.Chunks = append(parserFile.Chunks, parser.ParserChunk{
			ChunkIndex: fmt.Sprintf("%d", chunkIndex),
			ChunkHash:  chunkHash,
			ChunkSize:  fmt.Sprintf("%d", chunkSize),
		})
		chunkIndex++
	}

	// Copy the file to downloads
	downloadDirPath := fmt.Sprintf("downloads/daemon-%s/", d.IPCSocketIndex)
	if err := os.MkdirAll(downloadDirPath, os.ModePerm); err != nil {
		d.Logger.Warnf("Error creating directory: %v", err)
		return
	}

	createdFile, err := os.Create(downloadDirPath + fileName)
	if err != nil {
		d.Logger.Warnf("Error creating file: %v", err)
		return
	}
	defer createdFile.Close()

	if _, err := file.Seek(0, 0); err != nil {
		d.Logger.Warnf("Error seeking to start of file: %v", err)
		return
	}
	if _, err = io.Copy(createdFile, file); err != nil {
		d.Logger.Warnf("Error copying file: %v", err)
		return
	}

	// Create a .p2p file with the file metadata
	if err := parser.GenerateP2PFile(parserFile, downloadDirPath+fileName+".p2p"); err != nil {
		d.Logger.Warnf("Error generating .p2p file: %v", err)
		return
	}

	// Send announce message to tracker
	fileInfoMsgs := []*protocol.FileInfo{
		{
			FileSize:    dbFile.Size,
			ChunkSize:   int32(dbFile.MaxChunkSize),
			FileHash:    dbFile.Hash,
			TotalChunks: int32(dbFile.TotalChunks),
		},
	}
	d.Logger.Infof("Preparing to send Announce message to Tracker with newly created file: %v", fileInfoMsgs)

	netMsg := &protocol.NetworkMessage{
		MessageType: &protocol.NetworkMessage_Announce{
			Announce: &protocol.AnnounceMessage{
				Files: fileInfoMsgs,
			},
		},
	}

	if err := d.TrackerRouter.WriteMessage(netMsg); err != nil {
		d.Logger.Warnf("Error sending announce message: %v", err)
		return
	}
	d.Logger.Info("Successfully registered a new file with tracker")
}
