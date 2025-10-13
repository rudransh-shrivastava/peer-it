package main

import (
	"os"
	"os/signal"

	"github.com/rudransh-shrivastava/peer-it/internal/tracker"
)

func main() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)

	tracker := tracker.NewTracker()

	go tracker.Start()

	<-sigChan
	tracker.Stop()
}
