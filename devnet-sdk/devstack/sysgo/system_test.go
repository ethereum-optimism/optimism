package sysgo

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/devnet-sdk/devstack/devtest"
	"github.com/ethereum-optimism/optimism/devnet-sdk/devstack/shim"
	"github.com/ethereum-optimism/optimism/devnet-sdk/devstack/stack"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

func TestSystem(gt *testing.T) {
	var ids DefaultInteropSystemIDs
	opt := DefaultInteropSystem(&ids)

	logger := testlog.Logger(gt, log.LevelInfo)

	p := devtest.NewP(logger, func() {
		gt.Helper()
		gt.FailNow()
	})
	gt.Cleanup(p.Close)

	orch := NewOrchestrator(p)
	opt(orch)

	// Run two tests in parallel: see if we can share the same orchestrator
	// between two test scopes, with two different hydrated system frontends.
	gt.Run("testA", func(gt *testing.T) {
		gt.Parallel()

		t := devtest.SerialT(gt)
		system := shim.NewSystem(t)
		orch.Hydrate(system)

		testSystem(ids, system)
	})

	gt.Run("testB", func(gt *testing.T) {
		gt.Parallel()

		t := devtest.SerialT(gt)
		system := shim.NewSystem(t)
		orch.Hydrate(system)

		testSystem(ids, system)
	})
}

func testSystem(ids DefaultInteropSystemIDs, system stack.System) {
	t := system.T()
	logger := t.Logger()
	seqA := system.L2Network(ids.L2A).L2CLNode(ids.L2ACL)
	seqB := system.L2Network(ids.L2B).L2CLNode(ids.L2BCL)
	blocks := uint64(10)
	// wait for this many blocks, with some margin for delays
	for i := uint64(0); i < blocks*2+10; i++ {
		time.Sleep(time.Second * 2)

		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		statusA, err := seqA.RollupAPI().SyncStatus(ctx)
		require.NoError(t, err)
		statusB, err := seqB.RollupAPI().SyncStatus(ctx)
		require.NoError(t, err)
		cancel()
		logger.Info("chain A", "tip", statusA.UnsafeL2)
		logger.Info("chain B", "tip", statusB.UnsafeL2)

		if statusA.UnsafeL2.Number > blocks && statusB.UnsafeL2.Number > blocks {
			return
		}
	}
	t.Errorf("Expected to reach block %d on both chains", blocks)
	t.FailNow()
}
