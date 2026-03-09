package dsl

import (
	opclient "github.com/ethereum-optimism/optimism/op-service/client"

	"github.com/ethereum-optimism/optimism/op-devstack/stack"
)

type OPRBuilderNodeSet []*OPRBuilderNode

func NewOPRBuilderNodeSet(inner []stack.OPRBuilderNode) OPRBuilderNodeSet {
	oprbuilders := make([]*OPRBuilderNode, len(inner))
	for i, c := range inner {
		oprbuilders[i] = NewOPRBuilderNode(c)
	}
	return oprbuilders
}

type OPRBuilderNode struct {
	commonImpl
	inner    stack.OPRBuilderNode
	wsClient *opclient.WSClient
}

func NewOPRBuilderNode(inner stack.OPRBuilderNode) *OPRBuilderNode {
	return &OPRBuilderNode{
		commonImpl: commonFromT(inner.T()),
		inner:      inner,
		wsClient:   inner.FlashblocksClient(),
	}
}

func (c *OPRBuilderNode) String() string {
	return c.inner.Name()
}

func (c *OPRBuilderNode) Escape() stack.OPRBuilderNode {
	return c.inner
}

func (c *OPRBuilderNode) FlashblocksClient() *opclient.WSClient {
	return c.wsClient
}

func (el *OPRBuilderNode) Stop() {
	el.log.Info("Stopping", "name", el.inner.Name())
	lifecycle, ok := el.inner.(stack.Lifecycle)
	el.require.Truef(ok, "op-rbuilder node %s is not lifecycle-controllable", el.inner.Name())
	lifecycle.Stop()
}

func (el *OPRBuilderNode) Start() {
	lifecycle, ok := el.inner.(stack.Lifecycle)
	el.require.Truef(ok, "op-rbuilder node %s is not lifecycle-controllable", el.inner.Name())
	lifecycle.Start()
}
