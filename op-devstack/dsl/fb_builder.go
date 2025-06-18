package dsl

import (
	"fmt"
	"strings"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum/go-ethereum/log"
	"github.com/gorilla/websocket"
)

type FlashblocksBuilderSet []*FlashblocksBuilderNode

func (f *FlashblocksBuilderSet) Leader() *FlashblocksBuilderNode {
	for _, node := range *f {
		if node.Conductor().IsLeader() {
			return node
		}
	}
	return nil
}

func NewFlashblocksBuilderSet(inner []stack.FlashblocksBuilderNode) FlashblocksBuilderSet {
	flashblocksBuilders := make([]*FlashblocksBuilderNode, len(inner))
	for i, c := range inner {
		flashblocksBuilders[i] = NewFlashblocksBuilderNode(c)
	}
	return flashblocksBuilders
}

type FlashblocksBuilderNode struct {
	commonImpl
	inner stack.FlashblocksBuilderNode
}

func NewFlashblocksBuilderNode(inner stack.FlashblocksBuilderNode) *FlashblocksBuilderNode {
	return &FlashblocksBuilderNode{
		commonImpl: commonFromT(inner.T()),
		inner:      inner,
	}
}

func (c *FlashblocksBuilderNode) String() string {
	return c.inner.ID().String()
}

func (c *FlashblocksBuilderNode) Escape() stack.FlashblocksBuilderNode {
	return c.inner
}

func (c *FlashblocksBuilderNode) Conductor() *Conductor {
	return NewConductor(c.inner.Conductor())
}

func (c *FlashblocksBuilderNode) ListenFor(logger log.Logger, duration time.Duration, output chan<- []byte, done chan<- struct{}) error {
	defer close(done)
	wsURL := c.Escape().FlashblocksWsUrl()
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
