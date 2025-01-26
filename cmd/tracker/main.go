package main

import (
	"log"

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

	peerStore := store.NewPeerStore(db)
	fileStore := store.NewFileStore(db)
	chunkStore := store.NewChunkStore(db)
	tracker := tracker.NewTracker(peerStore, fileStore, chunkStore)

	tracker.Start()
}
