package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/rudransh-shrivastava/peer-it/internal/tracker"
)

func main() {

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)

	tracker := tracker.NewTracker()

	go tracker.Start()

	<-sigChan
	tracker.Stop()
}
