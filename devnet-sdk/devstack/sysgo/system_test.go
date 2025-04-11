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

func TestController(gt *testing.T) {
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

	t := devtest.SerialT(gt)
	system := shim.NewSystem(t)
	orch.Hydrate(system)

	controlPanel := orch.ControlPlane()

	stopPauseStartSupervisor := func() {
		log.Info("stopPauseStartSupervisor start")
		controlPanel.SupervisorState(ids.Supervisor, stack.Stop)
		time.Sleep(time.Second * 4)
		controlPanel.SupervisorState(ids.Supervisor, stack.Start)
		log.Info("stopPauseStartSupervisor done")
	}

	stopPauseStartOpNodes := func() {
		log.Info("stopPauseStartOpNodes start")
		controlPanel.L2CLNodeState(ids.L2ACL, stack.Stop)
		controlPanel.L2CLNodeState(ids.L2BCL, stack.Stop)
		time.Sleep(time.Second * 4)
		controlPanel.L2CLNodeState(ids.L2ACL, stack.Start)
		controlPanel.L2CLNodeState(ids.L2BCL, stack.Start)
		// logger.Info("Restoreeeeeeeeeeee start")
		// Restore(orch, ids)
		// logger.Info("Restoreeeeeeeeeeee done")
		log.Info("stopPauseStartOpNodes done")
	}

	// supervisor := system.Supervisor(ids.Supervisor)
	// for i := uint64(0); i < 5; i++ {
	// 	res, err := supervisor.QueryAPI().SyncStatus(context.Background())
	// 	time.Sleep(time.Second * 2)
	// 	if err != nil {
	// 		logger.Info("supersupseruper", "err", err)
	// 		continue
	// 	}
	// 	prettyJSON, err := json.MarshalIndent(res, "", "  ")
	// 	if err != nil {
	// 		fmt.Println("Error marshalling JSON:", err)
	// 		return
	// 	}
	// 	logger.Info("wowowoowwo")
	// 	fmt.Println(string(prettyJSON))
	// }

	// return
	// This does not work, must not interfere at block number 0. Fix this
	// stopPauseStartSupervisor()

	// TODO: investigate that op-node loops forever when supervisor is restarted at the very beginning

	{
		logger := system.T().Logger()
		// seqA := system.L2Network(ids.L2A).L2CLNode(ids.L2ACL)
		// seqB := system.L2Network(ids.L2B).L2CLNode(ids.L2BCL)
		elA := system.L2Network(ids.L2A).L2ELNode(ids.L2AEL)
		elB := system.L2Network(ids.L2B).L2ELNode(ids.L2BEL)
		blocks := uint64(10)
		for i := uint64(0); i < blocks*2+10; i++ {
			if i == 3 {
				stopPauseStartOpNodes()
			}
			if i == 10 {
				stopPauseStartSupervisor()
			}
			time.Sleep(time.Second * 2)

			ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
			blockA, err := elA.EthClient().BlockRefByLabel(ctx, "latest")
			require.NoError(t, err)
			blockB, err := elB.EthClient().BlockRefByLabel(ctx, "latest")
			require.NoError(t, err)
			cancel()

			logger.Info("chain A", "tip", blockA)
			logger.Info("chain B", "tip", blockB)

			// ctx2, cancel := context.WithTimeout(context.Background(), time.Second*10)
			// statusA, err := seqA.RollupAPI().SyncStatus(ctx2)
			// require.NoError(t, err)
			// statusB, err := seqB.RollupAPI().SyncStatus(ctx2)
			// require.NoError(t, err)
			// cancel()
			// logger.Info("chain A", "tip2", statusA.UnsafeL2)
			// logger.Info("chain B", "tip2", statusB.UnsafeL2)

			// if statusA.UnsafeL2.Number > blocks && statusB.UnsafeL2.Number > blocks {
			// 	return
			// }
		}
	}
}

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
