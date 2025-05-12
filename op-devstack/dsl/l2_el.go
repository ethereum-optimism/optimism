package dsl

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/retry"
)

// L2ELNode wraps a stack.L2ELNode interface for DSL operations
type L2ELNode struct {
	commonImpl
	elNode
	inner stack.L2ELNode
}

// NewL2ELNode creates a new L2ELNode DSL wrapper
func NewL2ELNode(inner stack.L2ELNode) *L2ELNode {
	return &L2ELNode{
		commonImpl: commonFromT(inner.T()),
		elNode:     elNode{inner: inner},
		inner:      inner,
	}
}

func (el *L2ELNode) String() string {
	return el.inner.ID().String()
}

// Escape returns the underlying stack.L2ELNode
func (el *L2ELNode) Escape() stack.L2ELNode {
	return el.inner
}

func (el *L2ELNode) BlockRefByLabel(label eth.BlockLabel) eth.BlockRef {
	ctx, cancel := context.WithTimeout(el.ctx, DefaultTimeout)
	defer cancel()
	block, err := el.inner.EthClient().BlockRefByLabel(ctx, label)
	el.require.NoError(err, "block not found using block label")
	return block
}

func (el *L2ELNode) Advance(label eth.BlockLabel, block uint64) CheckFunc {
	return func() error {
		initial := el.BlockRefByLabel(label)
		target := initial.Number + block
		el.log.Info("expecting chain to advance", "chain", el.inner.ChainID(), "label", label, "target", target)
		attempts := int(block + 3) // intentionally allow few more attempts for avoid flaking
		return retry.Do0(el.ctx, attempts, &retry.FixedStrategy{Dur: 2 * time.Second},
			func() error {
				head := el.BlockRefByLabel(label)
				if head.Number >= target {
					el.log.Info("chain advanced", "chain", el.inner.ChainID(), "target", target)
					return nil
				}
				el.log.Info("Chain sync status", "chain", el.inner.ChainID(), "initial", initial.Number, "current", head.Number, "target", target)
				return fmt.Errorf("expected head to advance: %s", label)
			})
	}
}

func (el *L2ELNode) DoesNotAdvance(label eth.BlockLabel) CheckFunc {
	return func() error {
		el.log.Info("expecting chain not to advance", "chain", el.inner.ChainID(), "label", label)
		initial := el.BlockRefByLabel(label)
		attempts := 5 // check few times to make sure head does not advance
		for range attempts {
			time.Sleep(2 * time.Second)
			head := el.BlockRefByLabel(label)
			el.log.Info("Chain sync status", "chain", el.inner.ChainID(), "initial", initial.Number, "current", head.Number, "target", initial.Number)
			if head.Hash == initial.Hash {
				continue
			}
			return fmt.Errorf("expected head not to advance: %s", label)
		}
		return nil
	}
}
