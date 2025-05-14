package sysgo

import (
	"context"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/shim"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-devstack/stack/match"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

func TestSystem(gt *testing.T) {
	var ids DefaultInteropSystemIDs
	opt := DefaultInteropSystem(&ids)

	logger := testlog.Logger(gt, log.LevelInfo)

	p := devtest.NewP(context.Background(), logger, func() {
		gt.Helper()
		gt.FailNow()
	})
	gt.Cleanup(p.Close)

	orch := NewOrchestrator(p, stack.Combine[*Orchestrator]())
	stack.ApplyOptionLifecycle(opt, orch)

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

	t.Run("test matchers", func(t devtest.T) {
		require := t.Require()
		require.Equal(ids.L1, system.L1Network(match.FirstL1Network).ID())
		require.Equal(ids.L1EL, system.L1Network(match.FirstL1Network).L1ELNode(match.FirstL1EL).ID())
		require.Equal(ids.L1CL, system.L1Network(match.FirstL1Network).L1CLNode(match.FirstL1CL).ID())
		require.Equal(ids.L2A, system.L2Network(match.L2ChainA).ID())
		require.Equal(ids.L2A, system.L2Network(match.FirstL2Network).ID())
		require.Equal(ids.L2B, system.L2Network(match.L2ChainB).ID())
		require.Equal(ids.Cluster, system.Cluster(match.FirstCluster).ID())
		require.Equal(ids.Superchain, system.Superchain(match.FirstSuperchain).ID())
		require.Equal(ids.Supervisor, system.Supervisor(match.FirstSupervisor).ID())
		l2A := system.L2Network(match.L2ChainA)
		require.Equal(ids.L2ACL, l2A.L2CLNode(match.FirstL2CL).ID())
		require.Equal(ids.L2AEL, l2A.L2ELNode(match.FirstL2EL).ID())
		require.Equal(ids.L2ABatcher, l2A.L2Batcher(match.FirstL2Batcher).ID())
		require.Equal(ids.L2AProposer, l2A.L2Proposer(match.FirstL2Proposer).ID())
	})

	t.Run("test labeling", func(t devtest.T) {
		require := t.Require()
		netB := system.L2Network(match.L2ChainB)
		require.Equal("", netB.Label("nickname"))
		netB.SetLabel("nickname", "Network B")
		require.Equal("Network B", netB.Label("nickname"))
		v := system.L2Network(match.WithLabel[stack.L2NetworkID, stack.L2Network](
			"nickname", "Network B"))
		require.Equal(ids.L2B, v.ID())
	})

	t.Run("op-geth match", func(t devtest.T) {
		elNode := system.L2Network(match.L2ChainA).L2ELNode(match.OpGeth)
		t.Require().Equal(string(match.OpGeth), elNode.Label(match.LabelVendor))
	})

	t.Run("find CL", func(t devtest.T) {
		elNode := system.L2Network(match.L2ChainA).L2ELNode(match.FirstL2EL)
		clNode := system.L2Network(match.L2ChainA).L2CLNode(match.WithEngine(elNode.ID()))
		t.Require().Contains(clNode.ELs(), elNode)
	})

	t.Run("sync", func(t devtest.T) {
		require := t.Require()
		seqA := system.L2Network(ids.L2A).L2CLNode(ids.L2ACL)
		seqB := system.L2Network(ids.L2B).L2CLNode(ids.L2BCL)
		blocks := uint64(5)
		// wait for this many blocks, with some margin for delays
		for i := uint64(0); i < blocks*2+10; i++ {
			time.Sleep(time.Second * 2)

			ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
			statusA, err := seqA.RollupAPI().SyncStatus(ctx)
			require.NoError(err)
			statusB, err := seqB.RollupAPI().SyncStatus(ctx)
			require.NoError(err)
			cancel()
			logger.Info("chain A", "tip", statusA.UnsafeL2)
			logger.Info("chain B", "tip", statusB.UnsafeL2)

			if statusA.UnsafeL2.Number > blocks && statusB.UnsafeL2.Number > blocks {
				return
			}
		}
		t.Errorf("Expected to reach block %d on both chains", blocks)
		t.FailNow()
	})
}
