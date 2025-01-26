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

	clientStore := store.NewClientStore(db)
	fileStore := store.NewFileStore(db)
	tracker := tracker.NewTracker(clientStore, fileStore)

	tracker.Start()
}
