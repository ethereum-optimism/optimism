package lokahi

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/wait"
	"github.com/ethereum-optimism/optimism/op-service/eth/safety"
)

// TestSafeLabelFollowsLokahi checks that the kona-nodes of two interop chains take their safe
// label from lokahi rather than from their own derivation: it advances behind local-safe while
// lokahi runs, holds while lokahi is down although both chains keep deriving, and catches up once
// lokahi is back. Finality follows lokahi too.
func TestSafeLabelFollowsLokahi(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := presets.NewTwoL2Lokahi(t)
	nodes := []*dsl.L2CLNode{sys.L2ACL, sys.L2BCL}

	// While lokahi runs, the safe label advances, never ahead of local-safe, and finality follows.
	var checks []dsl.CheckFunc
	for _, node := range nodes {
		safe := node.HeadBlockRef(safety.CrossSafe).Number
		checks = append(checks,
			safeReachesWithinLocalSafe(t, node, safe+5, 3*time.Minute),
			node.AdvancedFn(safety.Finalized, 1, 120))
	}
	dsl.CheckAll(t, checks...)
	logHeads(t, "lokahi running", nodes)

	// With lokahi down, both chains keep deriving local-safe, but nothing moves the safe label:
	// a node that took it from its own derivation would advance it with local-safe.
	sys.Lokahi.Stop()
	checks = nil
	for _, node := range nodes {
		node.WaitForStall(safety.CrossSafe)
		checks = append(checks,
			node.AdvancedFn(safety.LocalSafe, 10, 60),
			node.NotAdvancedFn(safety.CrossSafe, 30))
	}
	dsl.CheckAll(t, checks...)
	held := logHeads(t, "lokahi stopped", nodes)

	// Once lokahi is back, the safe label catches up with the local-safe head reached meanwhile.
	sys.Lokahi.Start()
	checks = nil
	for _, node := range nodes {
		localSafe := node.HeadBlockRef(safety.LocalSafe).Number
		t.Require().Greater(localSafe, held[node], "local-safe must have moved past the held safe head")
		checks = append(checks, safeReachesWithinLocalSafe(t, node, localSafe, 3*time.Minute))
	}
	dsl.CheckAll(t, checks...)
	logHeads(t, "lokahi restarted", nodes)
}

// safeReachesWithinLocalSafe waits until node's safe head reaches target, failing as soon as one
// sync status reports the safe head ahead of the local-safe head.
func safeReachesWithinLocalSafe(t devtest.T, node *dsl.L2CLNode, target uint64, timeout time.Duration) dsl.CheckFunc {
	return func() error {
		ctx, cancel := context.WithTimeout(t.Ctx(), timeout)
		defer cancel()
		logger := t.Logger().With("node", node.String(), "chain", node.ChainID(), "target", target)
		return wait.For(ctx, time.Second, func() (bool, error) {
			status, err := node.Escape().RollupAPI().SyncStatus(ctx)
			if err != nil {
				logger.Warn("Sync status unavailable; retrying", "err", err)
				return false, nil
			}
			if status.SafeL2.Number > status.LocalSafeL2.Number {
				return false, fmt.Errorf("%s: safe head %d is ahead of local-safe head %d",
					node, status.SafeL2.Number, status.LocalSafeL2.Number)
			}
			logger.Info("Waiting for the safe head", "safe", status.SafeL2.Number, "local_safe", status.LocalSafeL2.Number)
			return status.SafeL2.Number >= target, nil
		})
	}
}

// logHeads logs each node's labels and returns its safe head number.
func logHeads(t devtest.T, phase string, nodes []*dsl.L2CLNode) map[*dsl.L2CLNode]uint64 {
	safe := make(map[*dsl.L2CLNode]uint64, len(nodes))
	for _, node := range nodes {
		status := node.SyncStatus()
		t.Logger().Info("Labels", "phase", phase, "node", node.String(), "chain", node.ChainID(),
			"unsafe", status.UnsafeL2.Number, "local_safe", status.LocalSafeL2.Number,
			"safe", status.SafeL2.Number, "finalized", status.FinalizedL2.Number)
		safe[node] = status.SafeL2.Number
	}
	return safe
}
