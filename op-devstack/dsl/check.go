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
const maxInSyncGap = 5

// InSyncFn checks that two peer nodes are converged on the same canonical chain
// at the given safety level. Before the retry loop it records the higher of the
// two starting heads as a catch-up target, so the slower node must reach at
// least where the faster node was when the check began. On each attempt it
// re-samples both heads live and considers the nodes in sync when:
//  1. the slower head has reached the catch-up target; and
//  2. the two head numbers differ by at most maxInSyncGap; and
//  3. at the lower of the two heights, both nodes agree on the canonical block hash.
//
// The catch-up target prevents falsely passing when both heads sit below a
// recent divergence point and happen to agree on shared pre-reorg history.
// Unlike MatchedFn this does not require both live heads to be equal in the
// same polling tick. Unlike a single-snapshot approach it tolerates either side
// reorging during the wait, since both heads are re-sampled every attempt.
func InSyncFn(node1, node2 SyncStatusProvider, log log.Logger, ctx context.Context, lvl types.SafetyLevel, chainID eth.ChainID, attempts int) CheckFunc {
	return func() error {
		logger := log.With("node1_id", node1, "node2_id", node2, "chain", chainID, "label", lvl)
		provider1, canLookup1 := node1.(ChainBlockProvider)
		provider2, canLookup2 := node2.(ChainBlockProvider)

		initial1 := node1.ChainSyncStatus(chainID, lvl)
		initial2 := node2.ChainSyncStatus(chainID, lvl)
		catchupTarget := initial1.Number
		if initial2.Number > catchupTarget {
			catchupTarget = initial2.Number
		}
		logger.Info("Expecting nodes to converge",
			"initial_node1", initial1, "initial_node2", initial2,
			"catchup_target", catchupTarget, "max_gap", maxInSyncGap)

		return retry.Do0(ctx, attempts, &retry.FixedStrategy{Dur: 2 * time.Second},
			func() error {
				h1 := node1.ChainSyncStatus(chainID, lvl)
				h2 := node2.ChainSyncStatus(chainID, lvl)

				lower, higher := h1, h2
				lowerSide, higherSide := "node1", "node2"
				higherProvider, higherCanLookup := provider2, canLookup2
				if h2.Number < h1.Number {
					lower, higher = h2, h1
					lowerSide, higherSide = "node2", "node1"
					higherProvider, higherCanLookup = provider1, canLookup1
				}
				gap := higher.Number - lower.Number

				if lower.Number < catchupTarget {
					logger.Info("Slower node still catching up to initial high water mark",
						"node1", h1, "node2", h2, "catchup_target", catchupTarget)
					return fmt.Errorf("nodes not in sync: slower head at %d, must reach %d: %s", lower.Number, catchupTarget, lvl)
				}
				if gap > maxInSyncGap {
					logger.Info("Nodes too far apart to be in sync", "node1", h1, "node2", h2, "gap", gap)
					return fmt.Errorf("nodes not in sync: heads %d blocks apart (max %d): %s", gap, maxInSyncGap, lvl)
				}

				if gap == 0 {
					if lower.Hash == higher.Hash {
						logger.Info("Nodes in sync at matching head", "head", lower)
						return nil
					}
					logger.Info("Nodes diverged at matching head height", "node1", h1, "node2", h2)
					return fmt.Errorf("nodes not in sync: same height %d but different hash: %s", lower.Number, lvl)
				}

				// Different heights within the allowed gap: check the higher node's
				// canonical block at the lower height matches the lower node's hash.
				if !higherCanLookup {
					logger.Info("Cannot verify canonical block on higher node",
						"lower_side", lowerSide, "lower", lower,
						"higher_side", higherSide, "higher", higher)
					return fmt.Errorf("nodes not in sync: %s ahead but cannot verify its canonical block: %s", higherSide, lvl)
				}
				canonical, err := higherProvider.ChainBlockID(chainID, lower.Number)
				if err != nil {
					logger.Warn("Failed to fetch canonical block on higher node; will retry",
						"lower_side", lowerSide, "lower", lower,
						"higher_side", higherSide, "higher", higher, "err", err)
					return err
				}
				if canonical.Hash == lower.Hash {
					logger.Info("Nodes in sync; higher includes lower as canonical",
						"lower_side", lowerSide, "lower", lower,
						"higher_side", higherSide, "higher", higher)
					return nil
				}
				logger.Info("Nodes diverged at lower height",
					"lower_side", lowerSide, "lower", lower,
					"higher_side", higherSide, "higher", higher,
					"higher_canonical_at_lower", canonical)
				return fmt.Errorf("nodes not in sync: %s canonical block at height %d does not match %s head: %s", higherSide, lower.Number, lowerSide, lvl)
			})
	}
}
