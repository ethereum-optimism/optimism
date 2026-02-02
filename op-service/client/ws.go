package client

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/coder/websocket"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-service/retry"
)

// WSConfig configures a websocket connection.
// This is the shared configuration type for all outbound websocket clients in the codebase.
// Higher level users can build additional behavior on top, but should prefer DialWS / WSClient
// instead of constructing websocket.Dial calls directly.
type WSConfig struct {
	// URL is the websocket endpoint, e.g. wss://example:8546/ws.
	URL string
	// Headers are optional HTTP headers included in the websocket handshake.
	Headers http.Header

	// DialTimeout bounds the initial websocket dial.
	DialTimeout time.Duration
	// ReadTimeout bounds individual Read calls when a context without deadline is used.
	ReadTimeout time.Duration
	// WriteTimeout bounds individual Write calls when a context without deadline is used.
	WriteTimeout time.Duration

	// MaxAttempts configures how many dial attempts are made with backoff.
	// Defaults to 1 if zero.
	MaxAttempts int
	// Backoff is the backoff strategy used between dial attempts.
	// Defaults to retry.Exponential() if nil.
	Backoff retry.Strategy

	// Log is used for connection level logging.
	// If nil, logging is disabled.
	Log log.Logger
}

// applyDefaults fills empty WSConfig fields with conservative defaults.
func (c *WSConfig) applyDefaults() {
	if c.DialTimeout == 0 {
		c.DialTimeout = 10 * time.Second
	}
	if c.ReadTimeout == 0 {
		c.ReadTimeout = 30 * time.Second
	}
	if c.WriteTimeout == 0 {
		c.WriteTimeout = 30 * time.Second
	}
	if c.MaxAttempts < 1 {
		c.MaxAttempts = 1
	}
	if c.Backoff == nil {
		c.Backoff = retry.Exponential()
	}
}

// WSClient is the canonical outbound websocket client for the monorepo.
// It wraps a coder/websocket connection, handles dialing with backoff, and exposes
// context-aware read/write helpers. New outbound websocket integrations should go
// through this type (or helpers built on top of it) rather than using websocket.Dial
// directly.
type WSClient struct {
	conn   *websocket.Conn
	config WSConfig
}

// DialWS establishes a websocket connection using the given configuration.
// It performs MaxAttempts connection attempts with the configured backoff.
func DialWS(ctx context.Context, cfg WSConfig) (*WSClient, error) {
	cfg.applyDefaults()

	if cfg.Log != nil {
		cfg.Log.Info("Dialing websocket", "url", cfg.URL)
	}

	conn, err := retry.Do(ctx, cfg.MaxAttempts, cfg.Backoff, func() (*websocket.Conn, error) {
		dialCtx, cancel := context.WithTimeout(ctx, cfg.DialTimeout)
		defer cancel()
		conn, resp, err := websocket.Dial(dialCtx, cfg.URL, &websocket.DialOptions{
			HTTPHeader: cfg.Headers,
		})
		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close()
		}
		return conn, err
	})
	if err != nil {
		if cfg.Log != nil {
			cfg.Log.Warn("Failed to dial websocket", "url", cfg.URL, "err", err)
		}
		return nil, err
	}

	if cfg.Log != nil {
		cfg.Log.Info("Websocket connection established", "url", cfg.URL)
	}

	return &WSClient{
		conn:   conn,
		config: cfg,
	}, nil
}

// Close closes the websocket connection with the given status and reason.
func (c *WSClient) Close(status websocket.StatusCode, reason string) error {
	if c == nil || c.conn == nil {
		return nil
	}
	return c.conn.Close(status, reason)
}

// Read reads the next message from the websocket connection.
// If the context has no deadline, a default read timeout from the config is applied.
func (c *WSClient) Read(ctx context.Context) (websocket.MessageType, []byte, error) {
	if deadline, ok := ctx.Deadline(); !ok || time.Until(deadline) <= 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, c.config.ReadTimeout)
		defer cancel()
	}
	return c.conn.Read(ctx)
}

// Write writes a message to the websocket connection.
// If the context has no deadline, a default write timeout from the config is applied.
func (c *WSClient) Write(ctx context.Context, msgType websocket.MessageType, data []byte) error {
	if deadline, ok := ctx.Deadline(); !ok || time.Until(deadline) <= 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, c.config.WriteTimeout)
		defer cancel()
	}
	return c.conn.Write(ctx, msgType, data)
}

// ReadAll streams all websocket messages for the given duration into the provided output channel.
// It closes the done channel (when provided) after finishing the read loop.
func (c *WSClient) ReadAll(ctx context.Context, logger log.Logger, duration time.Duration, output chan<- []byte, done chan<- struct{}) error {
	if c == nil {
		return errors.New("ws client is nil")
	}
	if logger == nil {
		return errors.New("logger is nil")
	}
	if done != nil {
		defer close(done)
	}

	logger.Info("Listening on WebSocket client", "duration", duration)

	ctx, cancel := context.WithTimeout(ctx, duration)
	defer cancel()

	for {
		_, message, err := c.Read(ctx)
		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
				logger.Info("WebSocket read finished")
				return nil
			}
			logger.Error("Error reading WebSocket message", "error", err)
			return err
		}
		logger.Debug("Received WebSocket message", "message_length", len(message))

		select {
		case output <- message:
		case <-ctx.Done():
			logger.Info("Context done while sending message")
			return nil
		}
	}
}

// ProbeWS performs a lightweight websocket handshake against the given URL and closes immediately.
// It can be used in readiness checks to verify that the endpoint accepts websocket connections.
func ProbeWS(ctx context.Context, url string) error {
	cfg := WSConfig{
		URL:         url,
		DialTimeout: 5 * time.Second,
		MaxAttempts: 1,
	}
	cfg.applyDefaults()

	conn, err := DialWS(ctx, cfg)
	if err != nil {
		return err
	}
	// Close without waiting for the peer's close frame. Some flashblocks endpoints
	// immediately drop the connection after a successful handshake which makes the
	// full close handshake fail spuriously even though the endpoint is healthy.
	if conn.conn == nil {
		return nil
	}
	return conn.conn.CloseNow()
}
