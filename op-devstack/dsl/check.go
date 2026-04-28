package dsl

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/retry"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
	"github.com/ethereum/go-ethereum/log"
	"golang.org/x/sync/errgroup"
)

type CheckFunc func() error

func CheckAll(t devtest.T, checks ...CheckFunc) {
	var g errgroup.Group
	for _, check := range checks {
		check := check
		g.Go(func() error {
			return check()
		})
	}
	t.Require().NoError(g.Wait())
}

type SyncStatusProvider interface {
	ChainSyncStatus(chainID eth.ChainID, lvl types.SafetyLevel) eth.BlockID
	String() string
}

type ChainBlockProvider interface {
	ChainBlockID(chainID eth.ChainID, number uint64) (eth.BlockID, error)
}

var _ SyncStatusProvider = (*L2CLNode)(nil)
var _ SyncStatusProvider = (*Supervisor)(nil)

// LaggedFn returns a lambda that checks the baseNode head with given safety level is lagged with the refNode chain sync status provider
// Composable with other lambdas to wait in parallel
func LaggedFn(baseNode, refNode SyncStatusProvider, log log.Logger, ctx context.Context, lvl types.SafetyLevel, chainID eth.ChainID, attempts int, allowMatch bool) CheckFunc {
	return func() error {
		base := baseNode.ChainSyncStatus(chainID, lvl)
		ref := refNode.ChainSyncStatus(chainID, lvl)
		logger := log.With("base_id", baseNode, "ref_id", refNode, "chain", chainID, "label", lvl)
		logger.Info("Expecting node to lag with reference", "base", base.Number, "ref", ref.Number)
		for range attempts {
			base = baseNode.ChainSyncStatus(chainID, lvl)
			ref = refNode.ChainSyncStatus(chainID, lvl)
			cmp := base.Number > ref.Number
			msg := "Base chain surpassed"
			if !allowMatch {
				cmp = base.Number >= ref.Number
				msg += " or caught up"
			}
			if cmp {
				logger.Warn(msg, "base", base.Number, "ref", ref.Number)
				return fmt.Errorf("expected head to lag: %s", lvl)
			}
			logger.Info("Node sync status", "base", base.Number, "ref", ref.Number)
			time.Sleep(2 * time.Second)
		}
		logger.Info("Node lagged as expected")
		return nil
	}
}

// MatchedFn returns a lambda that checks the baseNode head with given safety level is matched with the refNode chain sync status provider
// Composable with other lambdas to wait in parallel
func MatchedFn(baseNode, refNode SyncStatusProvider, log log.Logger, ctx context.Context, lvl types.SafetyLevel, chainID eth.ChainID, attempts int) CheckFunc {
	return func() error {
		base := baseNode.ChainSyncStatus(chainID, lvl)
		ref := refNode.ChainSyncStatus(chainID, lvl)
		logger := log.With("base_id", baseNode, "ref_id", refNode, "chain", chainID, "label", lvl)
		logger.Info("Expecting node to match with reference", "base", base.Number, "ref", ref.Number)
		return retry.Do0(ctx, attempts, &retry.FixedStrategy{Dur: 2 * time.Second},
			func() error {
				base = baseNode.ChainSyncStatus(chainID, lvl)
				ref = refNode.ChainSyncStatus(chainID, lvl)
				if ref.Hash == base.Hash && ref.Number == base.Number {
					logger.Info("Node matched", "ref", ref.Number)
					return nil
				}
				logger.Info("Node sync status", "base", base.Number, "ref", ref.Number)
				return fmt.Errorf("expected head to match: %s", lvl)
			})
	}
}

// InSyncFn checks that baseNode has incorporated the refNode head observed when the check starts.
// Unlike MatchedFn, this does not require both live heads to be equal in the same polling tick.
func InSyncFn(baseNode, refNode SyncStatusProvider, log log.Logger, ctx context.Context, lvl types.SafetyLevel, chainID eth.ChainID, attempts int) CheckFunc {
	return func() error {
		target := refNode.ChainSyncStatus(chainID, lvl)
		logger := log.With("base_id", baseNode, "ref_id", refNode, "chain", chainID, "label", lvl, "target", target)
		logger.Info("Expecting node to sync to reference")
		blockProvider, canVerifyCanonicalBlock := baseNode.(ChainBlockProvider)
		return retry.Do0(ctx, attempts, &retry.FixedStrategy{Dur: 2 * time.Second},
			func() error {
				base := baseNode.ChainSyncStatus(chainID, lvl)
				if base.Number < target.Number {
					logger.Info("Node sync status", "base", base.Number, "target", target.Number)
					return fmt.Errorf("expected head to reach reference: %s", lvl)
				}
				if base.Number == target.Number {
					if base.Hash == target.Hash {
						logger.Info("Node reached reference", "target", target.Number)
						return nil
					}
					logger.Info("Node reached target number with different hash", "base", base, "target", target)
					return fmt.Errorf("expected head to match reference at target number: %s", lvl)
				}
				if !canVerifyCanonicalBlock {
					return fmt.Errorf("base head advanced past reference but canonical block cannot be verified: %s", lvl)
				}
				block, err := blockProvider.ChainBlockID(chainID, target.Number)
				if err != nil {
					logger.Warn("Failed to fetch canonical block at reference height; will retry", "err", err)
					return err
				}
				if block.Hash == target.Hash {
					logger.Info("Node includes reference", "target", target.Number, "base", base.Number)
					return nil
				}
				logger.Info("Node has different canonical block at reference height", "base_block", block, "target", target)
				return fmt.Errorf("expected canonical block to match reference: %s", lvl)
			})
	}
}
