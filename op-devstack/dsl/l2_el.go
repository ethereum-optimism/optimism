package dsl

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/retry"
	"github.com/ethereum/go-ethereum/common"
)

var emptyHash = common.Hash{}

// L2ELNode wraps a stack.L2ELNode interface for DSL operations
type L2ELNode struct {
	*elNode
	inner   stack.L2ELNode
	control stack.ControlPlane
}

// NewL2ELNode creates a new L2ELNode DSL wrapper
func NewL2ELNode(inner stack.L2ELNode, control stack.ControlPlane) *L2ELNode {
	return &L2ELNode{
		elNode:  newELNode(commonFromT(inner.T()), inner),
		inner:   inner,
		control: control,
	}
}

func (el *L2ELNode) String() string {
	return el.inner.ID().String()
}

// Escape returns the underlying stack.L2ELNode
func (el *L2ELNode) Escape() stack.L2ELNode {
	return el.inner
}

func (el *L2ELNode) ID() stack.L2ELNodeID {
	return el.inner.ID()
}

func (el *L2ELNode) BlockRefByLabel(label eth.BlockLabel) eth.L2BlockRef {
	ctx, cancel := context.WithTimeout(el.ctx, DefaultTimeout)
	defer cancel()
	block, err := el.inner.L2EthClient().L2BlockRefByLabel(ctx, label)
	el.require.NoError(err, "block not found using block label")
	return block
}

func (el *L2ELNode) AdvancedFn(label eth.BlockLabel, block uint64) CheckFunc {
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
				el.log.Info("chain sync status", "chain", el.inner.ChainID(), "initial", initial.Number, "current", head.Number, "target", target)
				return fmt.Errorf("expected head to advance: %s", label)
			})
	}
}

func (el *L2ELNode) NotAdvancedFn(label eth.BlockLabel) CheckFunc {
	return func() error {
		el.log.Info("expecting chain not to advance", "chain", el.inner.ChainID(), "label", label)
		initial := el.BlockRefByLabel(label)
		attempts := 5 // check few times to make sure head does not advance
		for range attempts {
			time.Sleep(2 * time.Second)
			head := el.BlockRefByLabel(label)
			el.log.Info("chain sync status", "chain", el.inner.ChainID(), "initial", initial.Number, "current", head.Number, "target", initial.Number)
			if head.Hash == initial.Hash {
				continue
			}
			return fmt.Errorf("expected head not to advance: %s", label)
		}
		return nil
	}
}

func (el *L2ELNode) BlockRefByNumber(num uint64) eth.L2BlockRef {
	ctx, cancel := context.WithTimeout(el.ctx, DefaultTimeout)
	defer cancel()
	block, err := el.inner.L2EthClient().L2BlockRefByNumber(ctx, num)
	el.require.NoError(err, "block not found using block number %d", num)
	return block
}

// ReorgTriggeredFn returns a lambda that checks that a L2 reorg occurred on the expected block
// Composable with other lambdas to wait in parallel
func (el *L2ELNode) ReorgTriggeredFn(target eth.L2BlockRef, attempts int) CheckFunc {
	return func() error {
		el.log.Info("expecting chain to reorg on block ref", "id", el.inner.ID(), "chain", el.inner.ID().ChainID(), "target", target)
		return retry.Do0(el.ctx, attempts, &retry.FixedStrategy{Dur: 2 * time.Second},
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

				if target.ParentHash != reorged.ParentHash && target.ParentHash != emptyHash {
					return fmt.Errorf("expected parent of target to be the same as the parent of the reorged head, but they are different")
				}

				el.log.Info("reorg on divergence block", "chain", el.inner.ID().ChainID(), "pre_blockref", target)
				el.log.Info("reorg on divergence block", "chain", el.inner.ID().ChainID(), "post_blockref", reorged)

				return nil
			})
	}
}

func (el *L2ELNode) Advanced(label eth.BlockLabel, block uint64) {
	el.require.NoError(el.AdvancedFn(label, block)())
}

func (el *L2ELNode) NotAdvanced(label eth.BlockLabel) {
	el.require.NoError(el.NotAdvancedFn(label)())
}

func (el *L2ELNode) ReorgTriggered(target eth.L2BlockRef, attempts int) {
	el.require.NoError(el.ReorgTriggeredFn(target, attempts)())
}

func (el *L2ELNode) TransactionTimeout() time.Duration {
	return el.inner.TransactionTimeout()
}

func (el *L2ELNode) Stop() {
	el.log.Info("Stopping", "id", el.inner.ID())
	el.control.L2ELNodeState(el.inner.ID(), stack.Stop)
}

func (el *L2ELNode) Start() {
	el.control.L2ELNodeState(el.inner.ID(), stack.Start)
}

func (el *L2ELNode) PeerWith(peer *L2ELNode) {
	sysgo.ConnectP2P(el.ctx, el.require, el.inner.L2EthClient().RPC(), peer.inner.L2EthClient().RPC())
}
