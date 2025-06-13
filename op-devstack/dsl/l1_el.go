package dsl

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/retry"
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

func (el *L1ELNode) EthClient() apis.EthClient {
	return el.inner.EthClient()
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

func (el *L1ELNode) BlockRefByLabel(label eth.BlockLabel) eth.L1BlockRef {
	ctx, cancel := context.WithTimeout(el.ctx, DefaultTimeout)
	defer cancel()
	block, err := el.inner.EthClient().BlockRefByLabel(ctx, label)
	el.require.NoError(err, "block not found using block label")
	return block
}

func (el *L1ELNode) BlockRefByNumber(number uint64) eth.L1BlockRef {
	ctx, cancel := context.WithTimeout(el.ctx, DefaultTimeout)
	defer cancel()
	block, err := el.inner.EthClient().BlockRefByNumber(ctx, number)
	el.require.NoError(err, "block not found using block number %d", number)
	return block
}

// ReorgTriggeredFn returns a lambda that checks that a L1 reorg occurred on the expected block
// Composable with other lambdas to wait in parallel
func (el *L1ELNode) ReorgTriggeredFn(target eth.L1BlockRef, attempts int) CheckFunc {
	return func() error {
		el.log.Info("expecting chain to reorg on block ref", "id", el.inner.ID(), "chain", el.inner.ID().ChainID(), "target", target)
		return retry.Do0(el.ctx, attempts, &retry.FixedStrategy{Dur: 7 * time.Second},
			func() error {
				reorged, err := el.inner.EthClient().BlockRefByNumber(el.ctx, target.Number)
				if err != nil {
					if strings.Contains(err.Error(), "not found") { // reorg is happening wait a bit longer
						el.log.Info("chain still hasn't been reorged", "chain", el.inner.ID().ChainID(), "error", err)
						return err
					}
					return err
				}

				if target.Hash == reorged.Hash { // want not equal
					el.log.Info("chain still hasn't been reorged", "chain", el.inner.ID().ChainID(), "ref", reorged)
					return fmt.Errorf("expected head to reorg %s, but got %s", target, reorged)
				}

				if target.ParentHash != reorged.ParentHash {
					return fmt.Errorf("expected parent of target to be the same as the parent of the reorged head, but they are different")
				}

				el.log.Info("reorg on divergence block", "chain", el.inner.ID().ChainID(), "pre_blockref", target)
				el.log.Info("reorg on divergence block", "chain", el.inner.ID().ChainID(), "post_blockref", reorged)

				return nil
			})
	}
}

func (el *L1ELNode) ReorgTriggered(target eth.L1BlockRef, attempts int) {
	el.require.NoError(el.ReorgTriggeredFn(target, attempts)())
}
