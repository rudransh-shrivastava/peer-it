package tracker

import (
	"crypto/rand"
	"crypto/sha256"
	"fmt"

	"github.com/rudransh-shrivastava/peer-it/internal/protocol"
)

func generateHash(file *protocol.FileEntry) protocol.FileHash {
	data := fmt.Sprintf("%s%d", file.Name, file.Size)
	return sha256.Sum256([]byte(data))
}

func generatePeerID() (id protocol.NodeID) {
	_, _ = rand.Read(id[:])
	return id
}
