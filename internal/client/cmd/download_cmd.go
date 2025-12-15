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
	_ = w.bar.Clear()
	n, err = os.Stdout.Write(p)
	_ = w.bar.RenderBlank()
	return
}

var downloadIndex string

var downloadCmd = &cobra.Command{
	Use:   "download <file-path>",
	Short: "download a file",
	Long:  `download a file from the swarm, requires a .pit file's path`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		filePath := args[0]
		logger := logger.NewLogger()

		bar := progressbar.NewOptions(-1,
			progressbar.OptionSetWidth(40),
			progressbar.OptionSetWriter(os.Stderr),
			progressbar.OptionClearOnFinish(),
			progressbar.OptionSetTheme(progressbar.Theme{
				Saucer:        "#",
				SaucerHead:    ">",
				SaucerPadding: "-",
				BarStart:      "[",
				BarEnd:        "]",
			}),
		)

		pw := &ProgressWriter{bar: bar}
		logger.SetOutput(pw)
		logger.SetFormatter(&logrus.TextFormatter{
			DisableTimestamp: true,
			ForceColors:      true,
		})

		client, err := client.NewClient(downloadIndex, logger)
		if err != nil {
			logger.Fatal(err)
			return
		}

		err = client.SendDownloadSignal(filePath)
		if err != nil {
			logger.Fatal(err)
			return
		}

		done := make(chan struct{})
		go client.ListenForDaemonLogs(done)

		go func() {
			totalChunks := <-client.TotalChunksChan
			bar.ChangeMax64(int64(totalChunks))
			for range client.ProgressBarChan {
				_ = bar.Add(1)
			}
		}()

		<-done
		fmt.Println()
		fileName := strings.Split(filePath, "/")[len(strings.Split(filePath, "/"))-1]
		actualFileName := strings.Join(strings.Split(fileName, ".")[:len(strings.Split(fileName, "."))-1], ".")
		logger.Infof("Downloaded %s successfully! ✅", actualFileName)
	},
}

func init() {
	downloadCmd.Flags().StringVarP(&downloadIndex, "index", "i", "0", "daemon index for local testing (default: 0)")
}
