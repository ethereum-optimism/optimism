package dsl

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum/go-ethereum/log"
	"github.com/gorilla/websocket"
)

type FlashblocksWebsocketProxySet []*FlashblocksWebsocketProxy

func NewFlashblocksWebsocketProxySet(inner []stack.FlashblocksWebsocketProxy) FlashblocksWebsocketProxySet {
	flashblocksWebsocketProxies := make([]*FlashblocksWebsocketProxy, len(inner))
	for i, c := range inner {
		flashblocksWebsocketProxies[i] = NewFlashblocksWebsocketProxy(c)
	}
	return flashblocksWebsocketProxies
}

type FlashblocksWebsocketProxy struct {
	commonImpl
	inner stack.FlashblocksWebsocketProxy
}

func NewFlashblocksWebsocketProxy(inner stack.FlashblocksWebsocketProxy) *FlashblocksWebsocketProxy {
	return &FlashblocksWebsocketProxy{
		commonImpl: commonFromT(inner.T()),
		inner:      inner,
	}
}

func (c *FlashblocksWebsocketProxy) String() string {
	return c.inner.ID().String()
}

func (c *FlashblocksWebsocketProxy) Escape() stack.FlashblocksWebsocketProxy {
	return c.inner
}

func (c *FlashblocksWebsocketProxy) ListenFor(logger log.Logger, duration time.Duration, output chan<- []byte, done chan<- struct{}) error {
	wsURL := c.Escape().WsUrl()
	headers := c.Escape().WsHeaders()
	return websocketListenFor(logger, wsURL, headers, duration, output, done)
}

func websocketListenFor(logger log.Logger, wsURL string, headers http.Header, duration time.Duration, output chan<- []byte, done chan<- struct{}) error {
	defer close(done)
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
	conn, resp, err := dialer.Dial(wsURL, headers)
	if err != nil {
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

	logger.Info("WebSocket connection established successfully", "url", wsURL, "reading stream for", duration)

	timeout := time.After(duration)
	messageCount := 0
	for {
		select {
		case <-timeout:
			logger.Info("WebSocket read timeout reached", "total_messages", messageCount)
			return nil
		default:
			err = conn.SetReadDeadline(time.Now().Add(duration))
			if err != nil {
				return fmt.Errorf("failed to set read deadline: %w", err)
			}
			_, message, err := conn.ReadMessage()
			if err != nil && !strings.Contains(err.Error(), "timeout") {
				logger.Error("Error reading WebSocket message", "error", err, "message_count", messageCount)
				return fmt.Errorf("error reading WebSocket message: %w", err)
			}
			if err == nil {
				messageCount++
				logger.Debug("Received WebSocket message", "message_count", messageCount, "message_length", len(message))
				select {
				case output <- message:
					logger.Debug("Message sent to output channel", "message_count", messageCount)
				case <-timeout: // to avoid indefinite hang
					logger.Info("Timeout while sending message to output channel", "total_messages", messageCount)
					return nil
				}
			}
		}
	}
}
