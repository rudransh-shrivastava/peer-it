package main

import (
	"fmt"
	"net"
	"time"

	"github.com/rudransh-shrivastava/peer-it/tracker/db"
	"github.com/rudransh-shrivastava/peer-it/tracker/store"
)

const (
	ClientTimeout = 10
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

	listen, err := net.Listen("tcp", ":8080")
	if err != nil {
		fmt.Println("Error starting TCP server:", err)
		return
	}
	defer listen.Close()

	fmt.Println("Server listening on port 8080")

	for {
		conn, err := listen.Accept()
		if err != nil {
			fmt.Println("Error accepting connection:", err)
			continue
		}

		// Handle each client connection in a new goroutine
		go handleClient(conn, clientStore)
	}
}

func handleClient(conn net.Conn, clientStore *store.ClientStore) {
	clientStore.CreateClient(conn.RemoteAddr().String(), conn.LocalAddr().String())

	ip := conn.RemoteAddr().String()

	fmt.Printf("New client connected: %s\n", ip)

	for {
		// Set a deadline for reading. Read operation will fail if no data is read after 20 seconds
		conn.SetReadDeadline(time.Now().Add(ClientTimeout * time.Second))

		// Read the message from the client
		message := make([]byte, 1024)
		_, err := conn.Read(message)
		if err != nil {
			fmt.Println("Error reading message:", err)
			break
		}

		// Reset the deadline for reading
		fmt.Printf("Resetting the deadline for client %s \n", ip)
		conn.SetReadDeadline(time.Time{})

		// Handle the message
		// handleMessage(client, message[:n], clientStore)
	}

	clientStore.DeleteClient(conn.RemoteAddr().String())
	fmt.Printf("Removing the client from list %s\n", ip)
	conn.Close()
}
