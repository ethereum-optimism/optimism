package wsutil

import (
	"context"
	"log"
	"net/http"
)

// WSServer maintains the set of active clients and broadcasts messages to the
// clients.
type WSServer struct {
	// Registered clients.
	clients map[*Client]bool

	// Inbound messages from the clients.
	broadcast chan []byte

	// Register requests from the clients.
	register chan *Client

	// Unregister requests from clients.
	unregister chan *Client
}

func NewWebSocketServer() *WSServer {
	return &WSServer{
		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
	}
}

func (h *WSServer) Start() {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = true
		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
		case message := <-h.broadcast:
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
		}
	}
}

func (h *WSServer) Close() error {
	close(h.broadcast)
	close(h.register)
	close(h.unregister)
	for client := range h.clients {
		close(client.send)
		delete(h.clients, client)
	}
	return nil
}

func (h *WSServer) Stop(ctx context.Context) error {
	return h.Close()
}

func (h *WSServer) Broadcast(ctx context.Context, msg []byte) error {
	for client := range h.clients {
		client.send <- msg
	}
	return nil
}

func (h *WSServer) HTTPHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	client := &Client{hub: h, conn: conn, send: make(chan []byte, 256)}
	client.hub.register <- client

	go client.writePump()
	go client.readPump()
}
