package sync

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
)

// TestL2CLAheadOfSupervisor tests the below scenario:
// L2CL ahead of supervisor, aka supervisor needs to reset the L2CL, to reproduce old data. Currently supervisor has only managed mode implemented, so the supervisor will ask the L2CL to reset back.
// To create this out-of-sync scenario, we follow the steps below:
// 0. System setup
// - Two supervisor initialized, each managing two L2CLs per chains.
// - Primary supervisor manages sequencer L2CLs for chain A, B.
// - Secondary supervisor manages verifier L2CLs for chain A, B.
// - Each L2CLs per chain is connected via P2P.
// 1. Make sequencers (L2CL), verifiers (L2CL), and supervisors sync for a few blocks.
// - Sequencer and verifier are connected via P2P, which makes their unsafe heads in sync.
// - Both L2CLs are in managed mode, digesting L1 blocks from the supervisor and reporting unsafe and safe blocks back to the supervisor.
// - Wait enough for both L2CLs advance unsafe heads.
// 2. Stop Secondary supervisor.
// - Verifiers stops advancing safe heads because there is no supervisor to provide them L1 data.
// - Verifiers advances unsafe head because they still have P2P connection with each sequencers.
// - Wait enough to make sequencers and primary supervisor advance safe head enough.
// 3. Connect verifiers (L2CL) to primary supervisor.
// - Primary supervisor has safe heads synced with sequencers.
// - After connection, verifiers will sync with primary supervisor, matching supervisor safe head view.
// - Secondary supervisor and verifiers becomes out-of-sync with safe heads.
// - Every L2CLs advance safe head.
// 4. Stop primary supervisor.
// - Every L2CL safe heads will stop advancing.
// - For disconnecting every L2CLs from the supervisor.
// 5. Restart primary supervisor and reconnect sequencers (L2CL) to primary supervisor.
// - Sequencers will resume advancing safe heads, but not verifiers.
// 6. Restart Secondary supervisor and reconnect verifiers (L2CL) to Secondary supervisor.
// - Secondary supervisor will compare its safe head knowledge with L2CLs, and find out L2CLs are ahead of the Secondary supervisor.
// - Secondary supervisor asks the verifiers (L2CL) to rewind(reset) back to match Secondary supervisor safe head view.
// - After rewinding(reset), verifier will advance safe heads again because Secondary supervisor gives L1 data to the verifiers.
// - Wait until verifiers advance safe head enough
func TestL2CLAheadOfSupervisor(gt *testing.T) {
	t := devtest.SerialT(gt)

	sys := presets.NewMultiSupervisorInterop(t)
	logger := sys.Log.With("Test", "TestL2CLAheadOfSupervisor")
	require := sys.T.Require()

	delta := uint64(10)
	logger.Info("make sure verifiers advances unsafe head", "delta", delta)
	dsl.CheckAll(t,
		sys.L2CLA.Advance("UnsafeL2", delta, 30), sys.L2CLA2.Advance("UnsafeL2", delta, 30),
		sys.L2CLB.Advance("UnsafeL2", delta, 30), sys.L2CLB2.Advance("UnsafeL2", delta, 30),
	)

	safeHeadViewA2 := sys.SupervisorSecondary.SyncView(sys.L2CLA.ChainID(), "CrossSafe")
	safeHeadViewB2 := sys.SupervisorSecondary.SyncView(sys.L2CLB.ChainID(), "CrossSafe")

	logger.Info("stop secondary supervisor")
	sys.SupervisorSecondary.Stop()

	safeHeadA2 := sys.L2CLA2.SafeL2BlockRef()
	safeHeadB2 := sys.L2CLB2.SafeL2BlockRef()
	require.Equal(safeHeadViewA2.Hash, safeHeadA2.Hash)
	require.Equal(safeHeadViewB2.Hash, safeHeadB2.Hash)
	logger.Info("secondary supervisor(stopped) safe head view", "chainA", safeHeadA2, "chainB", safeHeadB2)

	logger.Info("sequencers advances safe heads but not verifiers", "delta", delta)
	dsl.CheckAll(t,
		// verifier CLs cannot advance their safe head because secondary supervisor is down
		sys.L2CLA2.DoesNotAdvance("SafeL2", 30), sys.L2CLB2.DoesNotAdvance("SafeL2", 30),
		// sequencer CLs advance
		sys.L2CLA.Advance("SafeL2", delta, 30), sys.L2CLB.Advance("SafeL2", delta, 30),
	)

	logger.Info("connect verifier CLs to primary supervisor to advance verifier safe heads")
	sys.Supervisor.AddManagedL2CL(sys.L2CLA2)
	sys.Supervisor.AddManagedL2CL(sys.L2CLB2)

	target := max(sys.L2CLA.SafeL2BlockRef().Number, sys.L2CLB.SafeL2BlockRef().Number) + delta
	logger.Info("every CLs advance safe heads", "delta", delta, "target", target)
	dsl.CheckAll(t,
		sys.L2CLA.Reach("SafeL2", target, 30), sys.L2CLA2.Reach("SafeL2", target, 30),
		sys.L2CLB.Reach("SafeL2", target, 30), sys.L2CLB2.Reach("SafeL2", target, 30),
	)

	logger.Info("stop primary supervisor to disconnect every CL connection")
	sys.Supervisor.Stop()

	logger.Info("restart primary supervisor")
	sys.Supervisor.Start()

	logger.Info("no CL connected to supervisor so every CL safe head will not advance")
	dsl.CheckAll(t,
		sys.L2CLA.DoesNotAdvance("SafeL2", 30), sys.L2CLA2.DoesNotAdvance("SafeL2", 30),
		sys.L2CLB.DoesNotAdvance("SafeL2", 30), sys.L2CLB2.DoesNotAdvance("SafeL2", 30),
	)

	logger.Info("reconnect sequencer CLs to primary supervisor")
	sys.Supervisor.AddManagedL2CL(sys.L2CLA)
	sys.Supervisor.AddManagedL2CL(sys.L2CLB)

	logger.Info("restart secondary supervisor")
	sys.SupervisorSecondary.Start()

	logger.Info("reconnect verifier CLs to secondary supervisor")
	sys.SupervisorSecondary.AddManagedL2CL(sys.L2CLA2)
	sys.SupervisorSecondary.AddManagedL2CL(sys.L2CLB2)

	rewind := uint64(3)
	logger.Info("check verifier CLs safe head rewinded", "rewind", rewind)
	dsl.CheckAll(t,
		sys.L2CLA2.Rewind("SafeL2", rewind, 30),
		sys.L2CLB2.Rewind("SafeL2", rewind, 30),
	)

	target = max(sys.L2CLA.SafeL2BlockRef().Number, sys.L2CLB.SafeL2BlockRef().Number) + delta
	logger.Info("every CLs advance safe heads", "delta", delta, "target", target)
	dsl.CheckAll(t,
		sys.L2CLA.Reach("SafeL2", target, 30), sys.L2CLA2.Reach("SafeL2", target, 30),
		sys.L2CLB.Reach("SafeL2", target, 30), sys.L2CLB2.Reach("SafeL2", target, 30),
	)

	// Make sure each chain did not diverge
	require.Equal(sys.L2ELA.BlockRefByNumber(target).Hash, sys.L2ELA2.BlockRefByNumber(target).Hash)
	require.Equal(sys.L2ELB.BlockRefByNumber(target).Hash, sys.L2ELB2.BlockRefByNumber(target).Hash)
}
