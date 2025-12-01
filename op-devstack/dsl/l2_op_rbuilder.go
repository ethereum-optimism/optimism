package dsl

import (
	"context"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum/go-ethereum/log"
)

type OPRBuilderNodeSet []*OPRBuilderNode

func NewOPRBuilderNodeSet(inner []stack.OPRBuilderNode, control stack.ControlPlane) OPRBuilderNodeSet {
	oprbuilders := make([]*OPRBuilderNode, len(inner))
	for i, c := range inner {
		oprbuilders[i] = NewOPRBuilderNode(c, control)
	}
	return oprbuilders
}

type OPRBuilderNode struct {
	commonImpl
	inner    stack.OPRBuilderNode
	wsClient *FlashblocksWSClient
	control  stack.ControlPlane
}

func NewOPRBuilderNode(inner stack.OPRBuilderNode, control stack.ControlPlane) *OPRBuilderNode {
	return &OPRBuilderNode{
		commonImpl: commonFromT(inner.T()),
		inner:      inner,
		wsClient:   NewFlashblocksWSClient(inner.FlashblocksClient()),
		control:    control,
	}
}

func (c *OPRBuilderNode) String() string {
	return c.inner.ID().String()
}

func (c *OPRBuilderNode) Escape() stack.OPRBuilderNode {
	return c.inner
}

func (c *OPRBuilderNode) ListenFor(ctx context.Context, logger log.Logger, duration time.Duration, output chan<- []byte, done chan<- struct{}) error {
	return c.wsClient.ListenFor(ctx, logger, duration, output, done)
}

func (el *OPRBuilderNode) Stop() {
	el.log.Info("Stopping", "id", el.inner.ID())
	el.control.OPRBuilderNodeState(el.inner.ID(), stack.Stop)
}

func (el *OPRBuilderNode) Start() {
	el.control.OPRBuilderNodeState(el.inner.ID(), stack.Start)
}

func (el *OPRBuilderNode) FlashblocksClient() *FlashblocksWSClient {
	return NewFlashblocksWSClient(el.inner.FlashblocksClient())
}
