package main

import (
	"fmt"
	"net/http"

	"github.com/rudransh-shrivastava/peer-it/tracker/db"
	"github.com/rudransh-shrivastava/peer-it/tracker/handler"
	"github.com/rudransh-shrivastava/peer-it/tracker/store"
)

func main() {
	_, err := db.NewDB()
	if err != nil {
		fmt.Println(err)
		return
	}
	db, err := db.NewDB()
	if err != nil {
		fmt.Println(err)
		return
	}

	clientStore := store.NewClientStore(db)
	handler := handler.NewHandler(clientStore)

	http.HandleFunc("/", handler.NewClientConnect)
	http.HandleFunc("/clients", handler.GetClients)
	fmt.Println("Listening for web socket connections on port: 8080")
	err = http.ListenAndServe(":8080", nil)
	if err != nil {
		fmt.Println(err)
		return
	}
}
