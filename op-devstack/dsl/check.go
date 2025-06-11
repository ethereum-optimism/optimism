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
				logger.Info(msg, "base", base.Number, "ref", ref.Number)
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
