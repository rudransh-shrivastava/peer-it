package cmd

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
)

// Chunks the file into 256kb chunks, generates checksums for each chunk and stores them in the database
var registerCmd = &cobra.Command{
	Use:   "register path/to/file",
	Short: "register a file to the tracker server",
	Long:  `registers a file to the tracker server to make it available for download by other peers.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		filePath := args[0]

		file, err := os.Open(filePath)
		if err != nil {
			fmt.Println(err)
			return
		}
		defer file.Close()

		const chunkSize = 256 * 1024
		buffer := make([]byte, chunkSize)
		chunkIndex := 0

		for {
			n, err := file.Read(buffer)
			if err != nil && err != io.EOF {
				fmt.Println(err)
				return
			}
			if n == 0 {
				break
			}

			checksum := genCheckSum(buffer[:n])
			fmt.Printf("chunk %d: %s with size %d\n", chunkIndex, checksum, n)
			chunkIndex++
		}

		fmt.Printf("registered %s successfully \n", filePath)
	},
}

func genCheckSum(data []byte) string {
	hash := sha256.New()
	hash.Write(data)
	return fmt.Sprintf("%x", hash.Sum(nil))
}
