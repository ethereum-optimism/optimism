package dsl

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum/go-ethereum/log"
	"github.com/gorilla/websocket"
)

type FlashblocksWSClientSet []*FlashblocksWSClient

func NewFlashblocksWSClientSet(inner []stack.FlashblocksWSClient) FlashblocksWSClientSet {
	flashblocksWSClients := make([]*FlashblocksWSClient, len(inner))
	for i, c := range inner {
		flashblocksWSClients[i] = NewFlashblocksWSClient(c)
	}
	return flashblocksWSClients
}

type FlashblocksWSClient struct {
	commonImpl
	inner stack.FlashblocksWSClient
}

func NewFlashblocksWSClient(inner stack.FlashblocksWSClient) *FlashblocksWSClient {
	return &FlashblocksWSClient{
		commonImpl: commonFromT(inner.T()),
		inner:      inner,
	}
}

func (c *FlashblocksWSClient) String() string {
	return c.inner.ID().String()
}

func (c *FlashblocksWSClient) Escape() stack.FlashblocksWSClient {
	return c.inner
}

func (c *FlashblocksWSClient) ListenFor(ctx context.Context, logger log.Logger, duration time.Duration, output chan<- []byte, done chan<- struct{}) error {
	wsURL := c.Escape().WsUrl()
	headers := c.Escape().WsHeaders()
	return websocketListenFor(ctx, logger, wsURL, headers, duration, output, done)
}

func websocketListenFor(ctx context.Context, logger log.Logger, wsURL string, headers http.Header, duration time.Duration, output chan<- []byte, done chan<- struct{}) error {
	defer close(done)

	listenCtx, cancel := context.WithTimeout(ctx, duration)
	defer cancel()

	logger.Debug("Testing WebSocket connection to", "url", wsURL, "headers", headers)

	// Log the headers for debug purposes
	if headers != nil {
		for key, values := range headers {
			logger.Debug("Header", "key", key, "values", values)
		}
	} else {
		logger.Debug("No headers provided")
	}

	dialer := &websocket.Dialer{
		HandshakeTimeout: 6 * time.Second,
	}

	// Always close the response body to prevent resource leaks
	logger.Debug("Attempting WebSocket connection", "url", wsURL)
	conn, resp, err := dialer.DialContext(listenCtx, wsURL, headers)
	if err != nil {
		if listenCtx.Err() != nil {
			logger.Info("Context completed before WebSocket connection established", "reason", listenCtx.Err())
			return nil
		}
		logger.Error("WebSocket connection failed", "url", wsURL, "error", err)
		if resp != nil {
			logger.Error("HTTP response details", "status", resp.Status, "headers", resp.Header)
			resp.Body.Close()
		}
		return fmt.Errorf("failed to connect to Flashblocks WebSocket endpoint %s: %w", wsURL, err)
	}

	if resp != nil {
		defer resp.Body.Close()
	}
	defer conn.Close()

	logger.Info("WebSocket connection established successfully", "url", wsURL, "reading_stream_for", duration)
	go func() {
		<-listenCtx.Done()
		_ = conn.Close()
	}()

	messageCount := 0
	for {
		select {
		case <-listenCtx.Done():
			logListenStop(logger, listenCtx.Err(), messageCount)
			return nil
		default:
		}

		_, message, err := conn.ReadMessage()
		if err != nil {
			if listenCtx.Err() != nil {
				logListenStop(logger, listenCtx.Err(), messageCount)
				return nil
			}

			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway, websocket.CloseNoStatusReceived) {
				logger.Info("WebSocket connection closed by peer", "total_messages", messageCount)
				return nil
			}

			logger.Error("Error reading WebSocket message", "error", err, "message_count", messageCount)
			return fmt.Errorf("error reading WebSocket message: %w", err)
		}

		messageCount++
		logger.Debug("Received WebSocket message", "message_count", messageCount, "message_length", len(message))

		select {
		case output <- message:
			logger.Debug("Message sent to output channel", "message_count", messageCount)
		case <-listenCtx.Done():
			logListenStop(logger, listenCtx.Err(), messageCount)
			return nil
		}
	}
}

func logListenStop(logger log.Logger, reason error, messageCount int) {
	switch {
	case errors.Is(reason, context.DeadlineExceeded):
		logger.Info("WebSocket read duration reached", "total_messages", messageCount)
	case errors.Is(reason, context.Canceled):
		logger.Info("WebSocket listener canceled", "total_messages", messageCount)
	default:
		logger.Info("WebSocket listener stopping", "total_messages", messageCount)
	}
}
