package dsl

import (
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// L1ELNode wraps a stack.L1ELNode interface for DSL operations
type L1ELNode struct {
	*elNode
	inner stack.L1ELNode
}

// NewL1ELNode creates a new L1ELNode DSL wrapper
func NewL1ELNode(inner stack.L1ELNode) *L1ELNode {
	return &L1ELNode{
		elNode: newELNode(commonFromT(inner.T()), inner),
		inner:  inner,
	}
}

func (el *L1ELNode) String() string {
	return el.inner.ID().String()
}

// Escape returns the underlying stack.L1ELNode
func (el *L1ELNode) Escape() stack.L1ELNode {
	return el.inner
}

// EstimateBlockTime estimates the L1 block based on the last 1000 blocks
// (or since genesis, if insufficient blocks).
func (el *L1ELNode) EstimateBlockTime() time.Duration {
	latest, err := el.inner.EthClient().BlockRefByLabel(el.t.Ctx(), eth.Unsafe)
	el.require.NoError(err)
	if latest.Number == 0 {
		return time.Second * 12
	}
	lowerNum := uint64(0)
	if latest.Number > 1000 {
		lowerNum = latest.Number - 1000
	}
	lowerBlock, err := el.inner.EthClient().BlockRefByNumber(el.t.Ctx(), lowerNum)
	el.require.NoError(err)
	deltaTime := latest.Time - lowerBlock.Time
	deltaNum := latest.Number - lowerBlock.Number
	return time.Duration(deltaTime) * time.Second / time.Duration(deltaNum)
}
