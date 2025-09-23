package ws

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/ethereum-optimism/optimism/op-conductor/metrics"
	"github.com/ethereum/go-ethereum/log"
)

// Hub maintains the set of active clients and broadcasts messages to them
type Hub struct {
	// Registered clients
	clients map[*Client]bool

	// Register requests from the clients
	register chan *Client

	// Unregister requests from clients
	unregister chan *Client

	// Inbound messages to broadcast to the clients
	broadcast chan []byte

	// Signal to stop the hub
	done chan struct{}

	// Logger
	log log.Logger

	// Metrics
	metrics metrics.Metricer

	// Callbacks for testing and monitoring
	callbacks HubCallbacks
}

// newHub creates a new hub
func newHub(m metrics.Metricer) *Hub {
	return &Hub{
		broadcast:  make(chan []byte, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
		done:       make(chan struct{}),
		log:        log.New("component", "websocket-hub"),
		metrics:    m,
	}
}

// registerClient adds a client to the hub and updates metrics
func (h *Hub) registerClient(client *Client) {
	h.clients[client] = true
	clientCount := len(h.clients)
	h.log.Info("Client registered with hub", "totalClients", clientCount)
	h.metrics.RecordWebSocketClientCount(clientCount)

	if h.callbacks.OnClientRegistered != nil {
		h.callbacks.OnClientRegistered(client)
	}
}

// unregisterClient removes a client from the hub, closes it, and updates metrics
func (h *Hub) unregisterClient(client *Client) {
	if _, ok := h.clients[client]; ok {
		delete(h.clients, client)
		client.Close()
		clientCount := len(h.clients)
		h.log.Info("Client unregistered from hub", "totalClients", clientCount)
		h.metrics.RecordWebSocketClientCount(clientCount)

		if h.callbacks.OnClientUnregistered != nil {
			h.callbacks.OnClientUnregistered(client)
		}
	}
}

// run starts the hub's main loop
func (h *Hub) run() {
	for {
		select {
		case <-h.done:
			// Close all remaining client connections
			for client := range h.clients {
				h.unregisterClient(client)
			}
			h.metrics.RecordWebSocketClientCount(0)

			if h.callbacks.OnShutdown != nil {
				h.callbacks.OnShutdown()
			}
			return
		case client := <-h.register:
			h.registerClient(client)
		case client := <-h.unregister:
			h.unregisterClient(client)
		case message := <-h.broadcast:
			successCount := 0
			dropCount := 0

			for client := range h.clients {
				select {
				case client.send <- message:
					// Message sent successfully
					successCount++
				default:
					// Channel is full, client is likely slow/dead
					// The ping mechanism will detect and clean up dead clients
					h.log.Debug("Failed to send message to client, channel full")
					dropCount++
				}
			}
			if dropCount > 0 {
				h.log.Warn("Failed to send message to all clients, dropped", "successCount", successCount, "dropCount", dropCount)
			}

			if h.callbacks.OnMessageBroadcast != nil {
				h.callbacks.OnMessageBroadcast(message, successCount, dropCount)
			}
		}
	}
}

// Client represents a connected WebSocket client
type Client struct {
	conn   *websocket.Conn
	send   chan []byte
	ctx    context.Context
	cancel context.CancelFunc
	hub    *Hub
	log    log.Logger
	mu     sync.Mutex
}

// Close closes the client's WebSocket connection and send channel
func (c *Client) Close() {
	// this mutex is used to prevent concurrent close operations
	// double close will panic
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cancel()
	c.conn.Close(websocket.StatusNormalClosure, "client closed")
	close(c.send)
}

// newClient creates a new client with default settings
func newClient(conn *websocket.Conn, ctx context.Context, hub *Hub, logger log.Logger) *Client {
	ctx, cancel := context.WithCancel(ctx)
	return &Client{
		conn:   conn,
		send:   make(chan []byte, sendChannelBufferSize),
		ctx:    ctx,
		cancel: cancel,
		hub:    hub,
		log:    logger,
	}
}

// serveWs handles WebSocket requests from clients
func (h *Handler) serveWs(w http.ResponseWriter, r *http.Request) {
	// Upgrade the HTTP connection to a WebSocket connection using coder/websocket
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		CompressionMode: websocket.CompressionDisabled,
	})

	if err != nil {
		h.log.Error("failed to upgrade connection", "err", err)
		return
	}

	client := newClient(conn, r.Context(), h.hub, h.log)

	// Register the client with the hub
	h.hub.register <- client
	h.log.Info("WebSocket client connected")

	// Start client handling
	go h.writePump(client)
	h.readPump(client)
}

// readPump for followers where you don't expect regular data messages
func (h *Handler) readPump(client *Client) {
	defer func() {
		// Unregister the client when the read pump exits
		h.hub.unregister <- client
		h.log.Info("WebSocket read pump exited, client unregistered")
	}()

	for {
		select {
		case <-client.ctx.Done():
			return
		default:
			// Always read to process control frames (ping/pong/close)
			readCtx, cancel := context.WithTimeout(client.ctx, 30*time.Second)
			_, message, err := client.conn.Read(readCtx)
			cancel()

			if err != nil {
				if errors.Is(err, context.DeadlineExceeded) {
					// Timeout is expected when no messages - continue reading
					h.log.Debug("Read timeout occurred, continuing to process control frames")
					continue
				}
				if websocket.CloseStatus(err) != -1 {
					h.log.Debug("Client closed connection", "code", websocket.CloseStatus(err))
					return
				}
				h.log.Debug("Error reading from WebSocket client", "err", err)
				return
			}

			// Handle any data messages from clients if needed
			h.log.Debug("Received message from client", "message", string(message))
		}
	}
}

// writePump pumps messages from the hub to the WebSocket connection
func (h *Handler) writePump(client *Client) {
	defer func() {
		// Don't unregister here - let readPump handle it
		// Just log that the write pump is exiting
		h.log.Debug("WebSocket write pump exited")
	}()

	// Configure ping for connection keepalive
	pingTicker := time.NewTicker(pingInterval)
	defer pingTicker.Stop()

	for {
		select {
		case <-client.ctx.Done():
			return

		case message, ok := <-client.send:
			if !ok {
				// The hub closed the channel, exit the write pump
				h.log.Debug("Client send channel closed")
				return
			}

			// Write with timeout
			writeCtx, cancel := context.WithTimeout(client.ctx, writeTimeout)
			err := client.conn.Write(writeCtx, websocket.MessageText, message)
			cancel()

			if err != nil {
				h.log.Warn("Error writing to client", "err", err)
				return
			}

		case <-pingTicker.C:
			pingCtx, cancel := context.WithTimeout(client.ctx, pongTimeout)
			err := client.conn.Ping(pingCtx)
			cancel()

			if err != nil {
				h.log.Warn("Ping error", "err", err)
				return
			}
		}
	}
}
