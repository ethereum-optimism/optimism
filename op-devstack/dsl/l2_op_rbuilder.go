package dsl

import (
	opclient "github.com/ethereum-optimism/optimism/op-service/client"

	"github.com/ethereum-optimism/optimism/op-devstack/stack"
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
	wsClient *opclient.WSClient
	control  stack.ControlPlane
}

func NewOPRBuilderNode(inner stack.OPRBuilderNode, control stack.ControlPlane) *OPRBuilderNode {
	return &OPRBuilderNode{
		commonImpl: commonFromT(inner.T()),
		inner:      inner,
		wsClient:   inner.FlashblocksClient(),
		control:    control,
	}
}

func (c *OPRBuilderNode) String() string {
	return c.inner.ID().String()
}

func (c *OPRBuilderNode) Escape() stack.OPRBuilderNode {
	return c.inner
}

func (c *OPRBuilderNode) FlashblocksClient() *opclient.WSClient {
	return c.wsClient
}

func (el *OPRBuilderNode) Stop() {
	el.log.Info("Stopping", "id", el.inner.ID())
	el.control.OPRBuilderNodeState(el.inner.ID(), stack.Stop)
}

func (el *OPRBuilderNode) Start() {
	el.control.OPRBuilderNodeState(el.inner.ID(), stack.Start)
}
