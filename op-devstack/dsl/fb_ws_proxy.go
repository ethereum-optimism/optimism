package dsl

import (
	"fmt"
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
	return websocketListenFor(logger, c.Escape().WsUrl(), duration, output, done)
}

func websocketListenFor(logger log.Logger, wsURL string, duration time.Duration, output chan<- []byte, done chan<- struct{}) error {
	defer close(done)
	logger.Debug("Testing WebSocket connection to", "url", wsURL)

	dialer := &websocket.Dialer{
		HandshakeTimeout: 6 * time.Second,
	}

	conn, _, err := dialer.Dial(wsURL, nil)
	if err != nil {
		return fmt.Errorf("failed to connect to Flashblocks WebSocket endpoint %s: %w", wsURL, err)
	}
	defer conn.Close()

	logger.Info("WebSocket connection established, reading stream for %s", duration)

	timeout := time.After(duration)
	for {
		select {
		case <-timeout:
			return nil
		default:
			err = conn.SetReadDeadline(time.Now().Add(duration))
			if err != nil {
				return fmt.Errorf("failed to set read deadline: %w", err)
			}
			_, message, err := conn.ReadMessage()
			if err != nil && !strings.Contains(err.Error(), "timeout") {
				return fmt.Errorf("error reading WebSocket message: %w", err)
			}
			if err == nil {
				select {
				case output <- message:
				case <-timeout: // to avoid indefinite hang
					return nil
				}
			}
		}
	}
}
