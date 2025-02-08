package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/rudransh-shrivastava/peer-it/internal/client/client"
	"github.com/rudransh-shrivastava/peer-it/internal/shared/utils/logger"
	"github.com/schollz/progressbar/v3"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type ProgressWriter struct {
	bar *progressbar.ProgressBar
}

func (w *ProgressWriter) Write(p []byte) (n int, err error) {
	w.bar.Clear()
	n, err = os.Stdout.Write(p)
	w.bar.RenderBlank()
	return
}

var downloadCmd = &cobra.Command{
	Use:   "download file-path ipc-socket-index",
	Short: "download a file",
	Long:  `download a file from the swarm, requires a .pit file's path`,
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		filePath := args[0]
		ipcSocketIndex := args[1]
		logger := logger.NewLogger()
		// Setup progress bar first
		bar := progressbar.NewOptions(-1,
			progressbar.OptionSetWidth(40),
			progressbar.OptionSetWriter(os.Stderr),
			progressbar.OptionClearOnFinish(),
		)

		// Configure logger to use custom writer
		pw := &ProgressWriter{bar: bar}
		logger.SetOutput(pw)
		logger.SetFormatter(&logrus.TextFormatter{
			DisableTimestamp: true,
			ForceColors:      true,
		})
		client, err := client.NewClient(ipcSocketIndex, logger)
		if err != nil {
			logger.Fatal(err)
			return
		}
		// Send download signal
		err = client.SendDownloadSignal(filePath)
		if err != nil {
			logger.Fatal(err)
			return
		}

		done := make(chan struct{})
		go client.ListenForDaemonLogs(done)

		// Update progress bar with actual total when known
		go func() {
			totalChunks := <-client.TotalChunksChan
			bar.ChangeMax64(int64(totalChunks))
			for range client.ProgressBarChan {
				bar.Add(1)
			}
		}()

		<-done
		fmt.Println() // Final newline after progress bar
		fileName := strings.Split(filePath, "/")[len(strings.Split(filePath, "/"))-1]
		logger.Infof("Downloaded %s successfully! ✅", fileName)
	},
}
