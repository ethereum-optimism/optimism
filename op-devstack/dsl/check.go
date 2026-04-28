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

// maxInSyncGap is the largest difference (in blocks) between two node heads
// that InSyncFn will tolerate while still considering the nodes in sync. If
// the heads are further apart than this the slower node has not caught up yet.
const maxInSyncGap = 10

// InSyncFn checks that baseNode and refNode are converged on the same canonical
// chain at the given safety level. On each attempt it re-samples both heads
// live and considers the nodes in sync when:
//  1. the two head numbers differ by at most maxInSyncGap; and
//  2. at the lower of the two heights, both nodes agree on the canonical block hash.
//
// Unlike MatchedFn this does not require both live heads to be equal in the
// same polling tick. Unlike a single-snapshot approach it tolerates either side
// reorging during the wait, since both heads are re-sampled every attempt.
func InSyncFn(baseNode, refNode SyncStatusProvider, log log.Logger, ctx context.Context, lvl types.SafetyLevel, chainID eth.ChainID, attempts int) CheckFunc {
	return func() error {
		logger := log.With("base_id", baseNode, "ref_id", refNode, "chain", chainID, "label", lvl)
		logger.Info("Expecting nodes to converge", "max_gap", maxInSyncGap)
		baseProvider, baseCanLookup := baseNode.(ChainBlockProvider)
		refProvider, refCanLookup := refNode.(ChainBlockProvider)
		return retry.Do0(ctx, attempts, &retry.FixedStrategy{Dur: 2 * time.Second},
			func() error {
				base := baseNode.ChainSyncStatus(chainID, lvl)
				ref := refNode.ChainSyncStatus(chainID, lvl)

				var gap uint64
				if base.Number > ref.Number {
					gap = base.Number - ref.Number
				} else {
					gap = ref.Number - base.Number
				}
				if gap > maxInSyncGap {
					logger.Info("Nodes too far apart to be in sync", "base", base, "ref", ref, "gap", gap)
					return fmt.Errorf("nodes not in sync: heads %d blocks apart (max %d): %s", gap, maxInSyncGap, lvl)
				}

				if base.Number == ref.Number {
					if base.Hash == ref.Hash {
						logger.Info("Nodes in sync at matching head", "head", base)
						return nil
					}
					logger.Info("Nodes diverged at matching head height", "base", base, "ref", ref)
					return fmt.Errorf("nodes not in sync: same height %d but different hash: %s", base.Number, lvl)
				}

				// Different heights within the allowed gap: check the higher node's
				// canonical block at the lower height matches the lower node's hash.
				lowerID, lowerSide := base, "base"
				higherID, higherSide := ref, "ref"
				higherProvider, higherCanLookup := refProvider, refCanLookup
				if ref.Number < base.Number {
					lowerID, lowerSide = ref, "ref"
					higherID, higherSide = base, "base"
					higherProvider, higherCanLookup = baseProvider, baseCanLookup
				}

				if !higherCanLookup {
					logger.Info("Cannot verify canonical block on higher node",
						"lower_side", lowerSide, "lower", lowerID,
						"higher_side", higherSide, "higher", higherID)
					return fmt.Errorf("nodes not in sync: %s ahead but cannot verify its canonical block: %s", higherSide, lvl)
				}
				canonical, err := higherProvider.ChainBlockID(chainID, lowerID.Number)
				if err != nil {
					logger.Warn("Failed to fetch canonical block on higher node; will retry",
						"lower_side", lowerSide, "lower", lowerID,
						"higher_side", higherSide, "higher", higherID, "err", err)
					return err
				}
				if canonical.Hash == lowerID.Hash {
					logger.Info("Nodes in sync; higher includes lower as canonical",
						"lower_side", lowerSide, "lower", lowerID,
						"higher_side", higherSide, "higher", higherID)
					return nil
				}
				logger.Info("Nodes diverged at lower height",
					"lower_side", lowerSide, "lower", lowerID,
					"higher_side", higherSide, "higher", higherID,
					"higher_canonical_at_lower", canonical)
				return fmt.Errorf("nodes not in sync: %s canonical block at height %d does not match %s head: %s", higherSide, lowerID.Number, lowerSide, lvl)
			})
	}
}
