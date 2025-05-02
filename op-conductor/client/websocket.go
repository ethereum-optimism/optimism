package client

import (
	"context"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/gorilla/websocket"
)

// WebSocketControl defines the interface for connecting to and disconnecting from a WebSocket proxy.
type WebSocketControl interface {
	Connect(ctx context.Context) error
	Disconnect(ctx context.Context) error
	IsConnected(ctx context.Context) (bool, error)
}

// NewWebSocketControl creates a new WebSocketControl instance.
func NewWebSocketControl(url string, logger log.Logger) WebSocketControl {
	return &webSocketController{
		url:    url,
		log:    logger,
		isOpen: false,
		mu:     sync.Mutex{},
	}
}

type webSocketController struct {
	url    string
	conn   *websocket.Conn
	log    log.Logger
	isOpen bool
	mu     sync.Mutex
}

var _ WebSocketControl = (*webSocketController)(nil)

// Connect implements WebSocketControl.
func (w *webSocketController) Connect(ctx context.Context) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.isOpen {
		return nil // Already connected
	}

	w.log.Info("Connecting to WebSocket proxy", "url", w.url)

	dialer := websocket.DefaultDialer
	conn, _, err := dialer.DialContext(ctx, w.url, nil)
	if err != nil {
		w.log.Error("Failed to connect to WebSocket proxy", "err", err)
		return err
	}

	w.conn = conn
	w.isOpen = true

	// Set up ping handler
	w.conn.SetPingHandler(func(data string) error {
		w.log.Debug("Received ping", "data", data)
		return w.conn.WriteControl(websocket.PongMessage, []byte(data), time.Now().Add(10*time.Second))
	})

	// Start heartbeat in a separate goroutine
	go w.heartbeat(ctx)

	return nil
}

// Disconnect implements WebSocketControl.
func (w *webSocketController) Disconnect(ctx context.Context) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if !w.isOpen {
		return nil // Already disconnected
	}

	w.log.Info("Disconnecting from WebSocket proxy")

	// Send close message
	closeMsg := websocket.FormatCloseMessage(websocket.CloseNormalClosure, "")
	err := w.conn.WriteControl(websocket.CloseMessage, closeMsg, time.Now().Add(time.Second))
	if err != nil {
		w.log.Warn("Error sending close message", "err", err)
	}

	// Close the connection
	err = w.conn.Close()
	w.isOpen = false
	w.conn = nil

	return err
}

// IsConnected implements WebSocketControl.
func (w *webSocketController) IsConnected(ctx context.Context) (bool, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	return w.isOpen, nil
}

// heartbeat sends periodic pings to keep the connection alive.
func (w *webSocketController) heartbeat(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.mu.Lock()
			if w.isOpen && w.conn != nil {
				if err := w.conn.WriteControl(
					websocket.PingMessage,
					[]byte{},
					time.Now().Add(10*time.Second),
				); err != nil {
					w.log.Warn("Failed to send ping, connection may be down", "err", err)
					// Let it reconnect naturally through the leadership mechanisms
				}
			}
			w.mu.Unlock()
		}
	}
}
