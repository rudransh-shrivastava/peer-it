package handler

import (
	"fmt"
	"net"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/rudransh-shrivastava/peer-it/tracker/store"
)

type Handler struct {
	ClientStore *store.ClientStore
}

func NewHandler(clientStore *store.ClientStore) *Handler {
	return &Handler{
		ClientStore: clientStore,
	}
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func (h *Handler) NewClientConnect(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("recieved a new connection from %s\n", r.RemoteAddr)
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer func() {
		conn.Close()
		// store the client in the db
		host, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			fmt.Println(err)
		}
		h.ClientStore.DeleteClient(host)
	}()

	// store the client in the db
	host, port, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		fmt.Println(err)
	}
	h.ClientStore.CreateClient(host, port)
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			fmt.Println(err)
			break
		}

		fmt.Printf("Recieved: %s \n", message)

		// temp test
		// Echo the message back to the client
		if err := conn.WriteMessage(websocket.TextMessage, message); err != nil {
			fmt.Println("Error writing message:", err)
			break
		}
	}
}

// prints all clients to terminal
func (h *Handler) GetClients(w http.ResponseWriter, r *http.Request) {
	clients := h.ClientStore.GetClients()
	// repond with clients
	for _, client := range clients {
		fmt.Fprintf(w, "Client: %s:%s\n", client.IPAddress, client.Port)
	}
}
