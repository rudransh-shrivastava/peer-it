package daemon

import (
	"math/rand"
	"sync"
	"time"

	"github.com/rudransh-shrivastava/peer-it/internal/shared/protocol"
	"google.golang.org/protobuf/proto"
)

type FileDownload struct {
	FileHash     string
	TotalChunks  int
	ActivePeers  map[string]bool // peer IDs that have the file
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
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-dl.downloadDone:
			d.mu.Lock()
			delete(d.ActiveDownloads, dl.FileHash)
			d.mu.Unlock()
			return

		case <-ticker.C:
			chunks, err := d.ChunkStore.GetChunks(dl.FileHash)
			if err != nil {
				d.Logger.Errorf("Error checking chunks: %v", err)
				continue
			}

			// Check if download complete
			complete := true
			for _, chunk := range *chunks {
				if !chunk.IsAvailable {
					complete = false
					break
				}
			}

			if complete {
				close(dl.downloadDone)
				return
			}

			// Request missing chunks
			dl.mu.Lock()
			if len(dl.ActivePeers) > 0 {
				d.requestOneMissingChunk(dl.FileHash)
			}
			dl.mu.Unlock()
		}
	}
}

func (d *Daemon) requestOneMissingChunk(fileHash string) {
	localChunks, err := d.ChunkStore.GetChunks(fileHash)
	if err != nil {
		d.Logger.Errorf("Error getting local chunks: %v", err)
		return
	}

	// Find first missing chunk
	// TODO: Implement a better strategy for selecting missing chunks
	var missingChunkIndex int32 = -1
	for i, chunk := range *localChunks {
		if !chunk.IsAvailable {
			missingChunkIndex = int32(i)
			break
		}
	}

	if missingChunkIndex == -1 {
		return
	}

	// Get eligible peers for this chunk
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

	// Select a random peer
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

	// Check if peer still connected
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
			d.Logger.Infof("Requested chunk %d from %s (attempt %d)", missingChunkIndex, selectedPeer, i+1)
			return
		}
		time.Sleep(time.Duration(i+1) * time.Second)
	}
	d.Logger.Warnf("Failed to request chunk %d from %s after %d attempts", missingChunkIndex, selectedPeer, maxRetries)
}

// // Find missing chunks
// var missing []int32
// for i, chunk := range *localChunks {
// 	if !chunk.IsAvailable {
// 		missing = append(missing, int32(i))
// 	}
// }
// d.Logger.Debugf("Missing chunks for %s: %v", fileHash, missing)

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
