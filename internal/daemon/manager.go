package daemon

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/rudransh-shrivastava/peer-it/internal/shared/protocol"
	"google.golang.org/protobuf/proto"
)

type FileDownload struct {
	FileHash     string
	TotalChunks  int
	ActivePeers  map[string]bool
	mu           sync.Mutex
	downloadDone chan struct{}
}

func (d *Daemon) startFileDownload(fileHash string, totalChunks int) {
	d.mu.Lock()
	if _, exists := d.ActiveDownloads[fileHash]; exists {
		d.mu.Unlock()
		return
	}

	download := &FileDownload{
		FileHash:     fileHash,
		TotalChunks:  totalChunks,
		ActivePeers:  make(map[string]bool),
		downloadDone: make(chan struct{}),
	}
	d.ActiveDownloads[fileHash] = download
	d.mu.Unlock()

	go d.downloadManager(download)
}

func (d *Daemon) downloadManager(dl *FileDownload) {
	ctx := context.Background()
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for range ticker.C {
		chunks, err := d.ChunkStore.GetChunks(ctx, dl.FileHash)
		if err != nil {
			d.Logger.Errorf("Error checking chunks: %v", err)
			continue
		}

		complete := true
		for _, chunk := range chunks {
			if chunk.IsAvailable != 1 {
				complete = false
				break
			}
		}

		if complete {
			d.Logger.Infof("Download %s complete", dl.FileHash)
			d.mu.Lock()
			delete(d.ActiveDownloads, dl.FileHash)
			d.mu.Unlock()
			d.messageCLI("done")
			return
		}

		dl.mu.Lock()
		if len(dl.ActivePeers) > 0 {
			d.requestOneMissingChunk(dl.FileHash)
		}
		dl.mu.Unlock()
	}
}

func (d *Daemon) requestOneMissingChunk(fileHash string) {
	ctx := context.Background()
	localChunks, err := d.ChunkStore.GetChunks(ctx, fileHash)
	if err != nil {
		d.Logger.Errorf("Error getting local chunks: %v", err)
		return
	}

	var missingChunkIndex int32 = -1
	for i, chunk := range localChunks {
		if chunk.IsAvailable != 1 {
			missingChunkIndex = int32(i)
			break
		}
	}

	if missingChunkIndex == -1 {
		return
	}

	var eligiblePeers []string
	for peerID, peerFiles := range d.PeerChunkMap {
		if chunks, exists := peerFiles[fileHash]; exists {
			if missingChunkIndex < int32(len(chunks)) && chunks[missingChunkIndex] == 1 {
				eligiblePeers = append(eligiblePeers, peerID)
			}
		}
	}
	d.Logger.Debugf("Peers available for chunk %d of %s: %v", missingChunkIndex, fileHash, eligiblePeers)
	if len(eligiblePeers) == 0 {
		return
	}

	selectedPeer := eligiblePeers[rand.Intn(len(eligiblePeers))]
	d.Logger.Infof("Selecting peer %s to request chunk: %d", selectedPeer, missingChunkIndex)

	reqMsg := &protocol.NetworkMessage{
		MessageType: &protocol.NetworkMessage_ChunkRequest{
			ChunkRequest: &protocol.ChunkRequest{
				FileHash:   fileHash,
				ChunkIndex: missingChunkIndex,
			},
		},
	}

	if _, exists := d.PeerDataChannels[selectedPeer]; !exists {
		d.Logger.Warnf("Peer %s disconnected, skipping chunk request", selectedPeer)
		return
	}

	data, err := proto.Marshal(reqMsg)
	if err != nil {
		return
	}

	maxRetries := 3
	for i := 0; i < maxRetries; i++ {
		err = d.PeerDataChannels[selectedPeer].Send(data)
		if err == nil {
			log := fmt.Sprintf("Requested chunk %d from peer %s (attempt %d)", missingChunkIndex, selectedPeer, i+1)
			d.Logger.Info(log)
			return
		}
		time.Sleep(time.Duration(i+1) * time.Second)
	}
	d.Logger.Warnf("Failed to request chunk %d from %s after %d attempts", missingChunkIndex, selectedPeer, maxRetries)
}

func (d *Daemon) addPeerToDownload(peerID string, fileHash string) {
	d.mu.Lock()
	download, exists := d.ActiveDownloads[fileHash]
	d.mu.Unlock()

	if exists {
		download.mu.Lock()
		download.ActivePeers[peerID] = true
		download.mu.Unlock()
	}
}

func (d *Daemon) removePeerFromDownloads(peerID string) {
	d.mu.Lock()
	for _, download := range d.ActiveDownloads {
		delete(download.ActivePeers, peerID)
	}
	d.mu.Unlock()
}
