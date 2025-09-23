package ws

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/coder/websocket"

	"github.com/ethereum-optimism/optimism/op-conductor/metrics"
	"github.com/ethereum-optimism/optimism/op-service/retry"
	"github.com/ethereum/go-ethereum/log"
)

const (
	// reconnectDelay is the delay between reconnection attempts
	reconnectDelay = 5 * time.Second
	// pingInterval is how often to send pings to keep connections alive
	pingInterval = 15 * time.Second
	// pongTimeout is how long to wait for a pong response
	pongTimeout = 10 * time.Second
	// writeTimeout for all message writes
	writeTimeout = 5 * time.Second
	// send channel buffer size
	sendChannelBufferSize = 256
)

// FlashblockHandler manages WebSocket connections for flashblocks
type FlashblockHandler interface {
	// Start initializes and starts the flashblocks handler
	Start(ctx context.Context) error
	// Stop closes any open WebSocket connections
	Stop()
	// BroadcastMessage sends a message to all connected WebSocket clients
	BroadcastMessage(message []byte)
}

// Config contains configuration for the flashblocks handler
type Config struct {
	// WebsocketServerPort is the port to listen for WebSocket connections
	WebsocketServerPort int
	// RollupBoostWsURL is the URL of the rollup boost WebSocket
	RollupBoostWsURL string
}

// Handler implements the FlashblockHandler interface
type Handler struct {
	cfg                 Config
	log                 log.Logger
	isLeaderFn          func(context.Context) bool
	metrics             metrics.Metricer
	rollupBoostConn     *websocket.Conn
	rollupBoostCtx      context.Context
	rollupBoostWsCancel context.CancelFunc
	server              *http.Server
	hub                 *Hub
}

// NewHandler creates a new flashblocks handler
func NewHandler(cfg Config, log log.Logger, isLeaderFn func(context.Context) bool, m metrics.Metricer) (FlashblockHandler, error) {
	// Validate configuration
	if cfg.RollupBoostWsURL == "" || cfg.WebsocketServerPort <= 0 {
		log.Error("rollup boost WebSocket URL or websocket server port not configured")
		return nil, errors.New("rollup boost WebSocket URL or websocket server port not configured")
	}

	// Initialize the handler
	handler := &Handler{
		cfg:        cfg,
		log:        log,
		isLeaderFn: isLeaderFn,
		metrics:    m,
	}

	// Try to establish initial connection to rollup boost WebSocket
	maxConnectionAttempts := 5
	var err error
	handler.rollupBoostConn, err = retry.Do(context.Background(), maxConnectionAttempts, retry.Fixed(reconnectDelay), func() (*websocket.Conn, error) {
		log.Info("attempting to connect to rollup boost WebSocket", "url", cfg.RollupBoostWsURL)
		dialCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		conn, resp, err := websocket.Dial(dialCtx, cfg.RollupBoostWsURL, nil)
		if resp != nil && resp.Body != nil {
			resp.Body.Close()
		}
		return conn, err
	})

	if err != nil {
		return nil, fmt.Errorf("failed to connect to rollup boost WebSocket: %w", err)
	}

	return handler, nil
}

// Start initializes and starts the flashblocks handler
func (h *Handler) Start(ctx context.Context) error {
	// Start the WebSocket server
	if err := h.startWebSocketServer(ctx); err != nil {
		return err
	}

	// Start the rollup boost listener
	h.rollupBoostCtx, h.rollupBoostWsCancel = context.WithCancel(ctx)
	go h.listenToRollupBoost(h.rollupBoostCtx)

	return nil
}

// Stop closes any open WebSocket connections and shuts down the server
func (h *Handler) Stop() {
	// Signal the hub to stop if it exists
	if h.hub != nil {
		close(h.hub.done)
	}

	// Cancel the rollup boost context if it exists
	if h.rollupBoostWsCancel != nil {
		h.rollupBoostWsCancel()
	}

	// Close the rollup boost connection if it exists
	if h.rollupBoostConn != nil {
		h.log.Info("closing rollup boost WebSocket connection")
		h.rollupBoostConn.Close(websocket.StatusNormalClosure, "shutting down")
		h.rollupBoostConn = nil
	}

	// Close the HTTP server if it's running
	if h.server != nil {
		h.log.Info("closing WebSocket server")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err := h.server.Shutdown(ctx)
		if err != nil {
			h.log.Error("Error shutting down WebSocket server", "err", err)
		}
		h.log.Info("WebSocket server closed")
	}
}

// BroadcastMessage sends a message to all connected WebSocket clients
func (h *Handler) BroadcastMessage(message []byte) {
	h.hub.broadcast <- message
}

func (h *Handler) startWebSocketServer(_ context.Context) error {
	if h.cfg.WebsocketServerPort <= 0 {
		return fmt.Errorf("WebSocket server port not configured or invalid: %d", h.cfg.WebsocketServerPort)
	}

	h.hub = newHub(h.metrics)
	go h.hub.run()

	// Create HTTP server with WebSocket endpoint
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", h.handleWebSocket)

	// Start HTTP server
	h.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", h.cfg.WebsocketServerPort),
		Handler: mux,
	}

	h.log.Info("starting WebSocket server", "port", h.cfg.WebsocketServerPort)
	go func() {
		if err := h.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			h.log.Error("WebSocket server error", "err", err)
		}
	}()

	return nil
}

// handleWebSocket handles WebSocket upgrade requests
func (h *Handler) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	h.serveWs(w, r)
}

// listenToRollupBoost listens for messages from the rollup boost WebSocket
func (h *Handler) listenToRollupBoost(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			// Try to connect if not connected indefinitely
			if h.rollupBoostConn == nil {
				h.log.Info("reconnecting to rollup boost WebSocket", "url", h.cfg.RollupBoostWsURL)

				// Connect with timeout
				dialCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
				defer cancel()
				conn, resp, err := websocket.Dial(dialCtx, h.cfg.RollupBoostWsURL, nil)
				if resp != nil && resp.Body != nil {
					resp.Body.Close()
				}

				if err != nil {
					h.log.Warn("failed to connect to rollup boost WebSocket, will retry",
						"err", err, "retryIn", reconnectDelay)
					// add a metric for the number of times we've tried to connect
					h.metrics.RecordRollupBoostConnectionAttempts(false, h.cfg.RollupBoostWsURL)
					time.Sleep(reconnectDelay)
					continue
				}

				h.rollupBoostConn = conn
				h.log.Info("successfully connected to rollup boost WebSocket")
				h.metrics.RecordRollupBoostConnectionAttempts(true, h.cfg.RollupBoostWsURL)
			}

			// Read with timeout
			readCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
			defer cancel()
			_, message, err := h.rollupBoostConn.Read(readCtx)

			if err != nil {
				h.log.Warn("error reading from rollup boost WebSocket", "err", err)
				// Close and reset for reconnection
				if h.rollupBoostConn != nil {
					h.rollupBoostConn.Close(websocket.StatusInternalError, "read error")
					h.rollupBoostConn = nil
				}
				continue
			}

			h.handleRollupBoostMessage(ctx, message)
		}
	}
}

// handleRollupBoostMessage processes a message received from rollup boost
func (h *Handler) handleRollupBoostMessage(ctx context.Context, message []byte) {
	// Only forward messages if we're the leader - check dynamically each time
	if !h.isLeaderFn(ctx) {
		h.log.Trace("not forwarding rollup boost message, not the leader")
		return
	}

	// Forward the message to connected clients
	h.BroadcastMessage(message)
}
