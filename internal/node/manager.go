package node

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"google.golang.org/protobuf/proto"
)

type FileDownload struct {
	FileHash     string
	TotalChunks  int
	ActivePeers  map[string]bool
	mu           sync.Mutex
	downloadDone chan struct{}
}

func (n *Node) startFileDownload(fileHash string, totalChunks int) {
	n.mu.Lock()
	if _, exists := n.activeDownloads[fileHash]; exists {
		n.mu.Unlock()
		return
	}

	download := &FileDownload{
		FileHash:     fileHash,
		TotalChunks:  totalChunks,
		ActivePeers:  make(map[string]bool),
		downloadDone: make(chan struct{}),
	}
	n.activeDownloads[fileHash] = download
	n.mu.Unlock()

	go n.downloadManager(download)
}

func (n *Node) downloadManager(dl *FileDownload) {
	ctx := context.Background()
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for range ticker.C {
		chunks, err := n.chunks.GetChunks(ctx, dl.FileHash)
		if err != nil {
			n.logger.Errorf("Error checking chunks: %v", err)
			continue
		}

		if IsDownloadComplete(chunks) {
			n.logger.Infof("Download %s complete", dl.FileHash)
			n.mu.Lock()
			delete(n.activeDownloads, dl.FileHash)
			n.mu.Unlock()
			n.messageCLI("done")
			return
		}

		dl.mu.Lock()
		if len(dl.ActivePeers) > 0 {
			n.requestOneMissingChunk(dl.FileHash)
		}
		dl.mu.Unlock()
	}
}

func (n *Node) requestOneMissingChunk(fileHash string) {
	ctx := context.Background()
	localChunks, err := n.chunks.GetChunks(ctx, fileHash)
	if err != nil {
		n.logger.Errorf("Error getting local chunks: %v", err)
		return
	}

	missingChunkIndex := FindMissingChunkIndex(localChunks)
	if missingChunkIndex == -1 {
		return
	}

	eligiblePeers := FindEligiblePeers(n.peerChunkMap, fileHash, missingChunkIndex)
	n.logger.Debugf("Peers available for chunk %d of %s: %v", missingChunkIndex, fileHash, eligiblePeers)
	if len(eligiblePeers) == 0 {
		return
	}

	selectedPeer := SelectRandomPeer(eligiblePeers, rand.Intn)
	n.logger.Infof("Selecting peer %s to request chunk: %d", selectedPeer, missingChunkIndex)

	if _, exists := n.peerDataChannels[selectedPeer]; !exists {
		n.logger.Warnf("Peer %s disconnected, skipping chunk request", selectedPeer)
		return
	}

	data, err := proto.Marshal(BuildChunkRequestMessage(fileHash, missingChunkIndex))
	if err != nil {
		return
	}

	maxRetries := 3
	for i := 0; i < maxRetries; i++ {
		err = n.peerDataChannels[selectedPeer].Send(data)
		if err == nil {
			log := fmt.Sprintf("Requested chunk %d from peer %s (attempt %d)", missingChunkIndex, selectedPeer, i+1)
			n.logger.Info(log)
			return
		}
		time.Sleep(time.Duration(i+1) * time.Second)
	}
	n.logger.Warnf("Failed to request chunk %d from %s after %d attempts", missingChunkIndex, selectedPeer, maxRetries)
}

func (n *Node) addPeerToDownload(peerID string, fileHash string) {
	n.mu.Lock()
	download, exists := n.activeDownloads[fileHash]
	n.mu.Unlock()

	if exists {
		download.mu.Lock()
		download.ActivePeers[peerID] = true
		download.mu.Unlock()
	}
}

func (n *Node) removePeerFromDownloads(peerID string) {
	n.mu.Lock()
	for _, download := range n.activeDownloads {
		delete(download.ActivePeers, peerID)
	}
	n.mu.Unlock()
}
