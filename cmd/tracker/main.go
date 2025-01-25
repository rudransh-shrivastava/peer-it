package main

import (
	"fmt"

	"github.com/rudransh-shrivastava/peer-it/internal/shared/store"
	"github.com/rudransh-shrivastava/peer-it/internal/tracker"
	"github.com/rudransh-shrivastava/peer-it/internal/tracker/db"
)

func main() {
	db, err := db.NewDB()
	if err != nil {
		fmt.Println(err)
		return
	}

	clientStore := store.NewClientStore(db)
	tracker := tracker.NewTracker(clientStore)

	tracker.Start()
}
