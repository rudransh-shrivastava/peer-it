package node

import (
	"crypto/sha256"
	"fmt"
	"io"
	"strings"
)

func HashFile(r io.Reader) (string, error) {
	hash := sha256.New()
	if _, err := io.Copy(hash, r); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

func CalculateTotalChunks(fileSize, chunkSize int64) int {
	if chunkSize <= 0 {
		return 0
	}
	return int((fileSize + chunkSize - 1) / chunkSize)
}

func ExtractFileName(path string) string {
	parts := strings.Split(path, "/")
	if len(parts) == 0 {
		return path
	}
	return parts[len(parts)-1]
}

func BuildDownloadPath(ipcIndex, fileName string) string {
	return fmt.Sprintf("downloads/daemon-%s/%s", ipcIndex, fileName)
}

func BuildSocketPath(ipcIndex string) string {
	return fmt.Sprintf("/tmp/pit-daemon-%s.sock", ipcIndex)
}
