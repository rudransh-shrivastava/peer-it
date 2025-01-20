package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/gorilla/websocket"
)

func main() {
	serverUrl := "ws://localhost:8080/"
	conn, _, err := websocket.DefaultDialer.Dial(serverUrl, nil)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer conn.Close()

	fmt.Println("successfully connected to the web socket server")

	done := make(chan os.Signal, 1)
	signal.Notify(done, syscall.SIGINT, syscall.SIGTERM)

	// listen for messages from the server
	go func() {
		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				log.Println("Error reading message:", err)
				return
			}
			fmt.Println("Received message:", string(msg))
		}
	}()

	// send some random message i guess
	message := "Hello from Go client"
	err = conn.WriteMessage(websocket.TextMessage, []byte(message))
	if err != nil {
		log.Println("Error sending message:", err)
		return
	}
	fmt.Println("Sent message:", message)

	<-done
	fmt.Println("exiting...")
}
