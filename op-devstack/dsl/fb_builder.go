package dsl

import (
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum/go-ethereum/log"
)

type FlashblocksBuilderSet []*FlashblocksBuilderNode

func (f FlashblocksBuilderSet) Leader() *FlashblocksBuilderNode {
	for _, node := range f {
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
	return websocketListenFor(logger, c.inner.FlashblocksWsUrl(), duration, output, done)
}
