package lokahi

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/wait"
	"github.com/ethereum-optimism/optimism/op-service/eth/safety"
)

// lokahiInteropDelay is how long after genesis interop activates, in seconds: long enough for L1
// finality to reach blocks the chains derived before activation.
const lokahiInteropDelay = 180

// TestLokahiInteropActivation checks interop activating after genesis. Before activation each
// node's safe head is its local-safe head and finality follows the node's L1 finality, also while
// lokahi is down. Across activation no label moves back, and after it the safe head follows
// lokahi: it holds while lokahi is down although local-safe advances.
func TestLokahiInteropActivation(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewTwoL2LokahiWithInteropDelay(t, lokahiInteropDelay)
	nodes := []*dsl.L2CLNode{sys.L2ACL, sys.L2BCL}
	els := map[*dsl.L2CLNode]*dsl.L2ELNode{sys.L2ACL: sys.L2ELA, sys.L2BCL: sys.L2ELB}
	activation := sys.InteropActivationTime
	activationBlock, err := sys.L2A.Escape().RollupConfig().TargetBlockNumber(activation)
	t.Require().NoError(err)
	t.Logger().Info("Interop activates after genesis", "activation", activation, "block", activationBlock)
	labels := watchLabels(t, sys.Lokahi, nodes, els)
	safeHeads := watchSafeHeads(t, nodes)

	// Before activation the nodes need no verifier: with lokahi down the safe heads follow
	// local-safe and finality advances with L1.
	sys.Lokahi.Stop()
	var checks []dsl.CheckFunc
	for _, node := range nodes {
		checks = append(checks, safeIsLocalSafeUntilFinal(t, node, activation))
	}
	dsl.CheckAll(t, checks...)
	logHeads(t, "before activation, lokahi down", nodes)
	sys.Lokahi.Start()

	// The safe heads pass activation, which only lokahi's verified heads move them past.
	checks = nil
	for _, node := range nodes {
		checks = append(checks, node.ReachedFn(safety.CrossSafe, activationBlock+3, 120))
	}
	dsl.CheckAll(t, checks...)
	logHeads(t, "after activation", nodes)

	// After activation the safe heads follow lokahi: with lokahi down they hold although
	// local-safe advances.
	sys.Lokahi.Stop()
	checks = nil
	for _, node := range nodes {
		node.WaitForStall(safety.CrossSafe)
		checks = append(checks,
			node.AdvancedFn(safety.LocalSafe, 5, 60),
			node.NotAdvancedFn(safety.CrossSafe, 20))
	}
	dsl.CheckAll(t, checks...)
	logHeads(t, "after activation, lokahi down", nodes)
	sys.Lokahi.Start()
	checks = nil
	for _, node := range nodes {
		checks = append(checks, node.AdvancedFn(safety.CrossSafe, 3, 60))
	}
	dsl.CheckAll(t, checks...)

	errs := append(labels.stop().errs, safeHeads.stop())
	t.Require().NoError(errors.Join(errs...))
}

// safeIsLocalSafeUntilFinal waits until node's finalized head and safe head have advanced while
// its local-safe head is before activation, failing if a sync status reports a safe head other
// than the local-safe head before activation, or if activation comes first.
func safeIsLocalSafeUntilFinal(t devtest.T, node *dsl.L2CLNode, activation uint64) dsl.CheckFunc {
	return func() error {
		ctx, cancel := context.WithTimeout(t.Ctx(), 3*time.Minute)
		defer cancel()
		return wait.For(ctx, 500*time.Millisecond, func() (bool, error) {
			status, err := node.Escape().RollupAPI().SyncStatus(ctx)
			if err != nil {
				return false, nil
			}
			if status.LocalSafeL2.Time >= activation {
				return false, fmt.Errorf("%s: activation came before finality moved: local-safe %s, finalized %s",
					node, status.LocalSafeL2, status.FinalizedL2)
			}
			if status.SafeL2 != status.LocalSafeL2 {
				return false, fmt.Errorf("%s: safe head %s is not the local-safe head %s before activation",
					node, status.SafeL2, status.LocalSafeL2)
			}
			t.Logger().Info("Before activation", "node", node.String(), "safe", status.SafeL2.Number,
				"finalized", status.FinalizedL2.Number)
			return status.FinalizedL2.Number > 0 && status.SafeL2.Number >= 5, nil
		})
	}
}
