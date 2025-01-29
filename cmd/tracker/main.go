package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/rudransh-shrivastava/peer-it/internal/shared/store"
	"github.com/rudransh-shrivastava/peer-it/internal/tracker"
	"github.com/rudransh-shrivastava/peer-it/internal/tracker/db"
)

func main() {
	db, err := db.NewDB()
	if err != nil {
		log.Fatal(err)
		return
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)

	peerStore := store.NewPeerStore(db)
	fileStore := store.NewFileStore(db)
	chunkStore := store.NewChunkStore(db)
	tracker := tracker.NewTracker(peerStore, fileStore, chunkStore)

	go tracker.Start()

	<-sigChan
	log.Println("Stopping the tracker...")
	tracker.Stop()
}
