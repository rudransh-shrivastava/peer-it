package cmd

import (
	"github.com/rudransh-shrivastava/peer-it/internal/parser"
	"github.com/rudransh-shrivastava/peer-it/internal/shared/utils/logger"
	"github.com/spf13/cobra"
)

var parseCmd = &cobra.Command{
	Use:   "parse path/to/file",
	Short: "parses a .p2p file",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		filePath := args[0]
		logger := logger.NewLogger()
		parserFile, err := parser.ParseP2PFile(filePath)
		if err != nil {
			logger.Fatal(err)
		}

		logger.Info("Successfully parsed file")
		logger.Infof("%+v", parserFile)
	},
}
