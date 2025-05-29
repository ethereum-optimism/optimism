package sync

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

// TestL2CLAheadOfSupervisor tests the below scenario:
// L2CL ahead of supervisor, aka supervisor needs to reset the L2CL, to reproduce old data. Currently supervisor has only managed mode implemented, so the supervisor will ask the L2CL to reset back.
func TestL2CLAheadOfSupervisor(gt *testing.T) {
	t := devtest.SerialT(gt)

	// Two supervisor initialized, each managing two L2CLs per chains.
	// Primary supervisor manages sequencer L2CLs for chain A, B.
	// Secondary supervisor manages verifier L2CLs for chain A, B.
	// Each L2CLs per chain is connected via P2P.
	sys := presets.NewMultiSupervisorInterop(t)
	logger := sys.Log.With("Test", "TestL2CLAheadOfSupervisor")
	require := sys.T.Require()

	// Make sequencers (L2CL), verifiers (L2CL), and supervisors sync for a few blocks.
	// Sequencer and verifier are connected via P2P, which makes their unsafe heads in sync.
	// Both L2CLs are in managed mode, digesting L1 blocks from the supervisor and reporting unsafe and safe blocks back to the supervisor.
	delta := uint64(10)
	logger.Info("Make sure verifiers advances unsafe head", "delta", delta)
	dsl.CheckAll(t,
		sys.L2CLA.AdvancedFn(types.LocalUnsafe, delta, 30), sys.L2CLA2.AdvancedFn(types.LocalUnsafe, delta, 30),
		sys.L2CLB.AdvancedFn(types.LocalUnsafe, delta, 30), sys.L2CLB2.AdvancedFn(types.LocalUnsafe, delta, 30),
	)

	safeHeadViewA2 := sys.SupervisorSecondary.SafeBlockID(sys.L2CLA.ChainID())
	safeHeadViewB2 := sys.SupervisorSecondary.SafeBlockID(sys.L2CLB.ChainID())

	logger.Info("Stop secondary supervisor")
	sys.SupervisorSecondary.Stop()

	safeHeadA2 := sys.L2CLA2.SafeL2BlockRef()
	safeHeadB2 := sys.L2CLB2.SafeL2BlockRef()
	require.Equal(safeHeadViewA2.Hash, safeHeadA2.Hash)
	require.Equal(safeHeadViewB2.Hash, safeHeadB2.Hash)
	logger.Info("Secondary supervisor(stopped) safe head view", "chainA", safeHeadA2, "chainB", safeHeadB2)

	// Wait enough to make sequencers and primary supervisor advance safe head enough.
	logger.Info("Sequencers advances safe heads but not verifiers", "delta", delta)
	dsl.CheckAll(t,
		// verifier CLs cannot advance their safe head because secondary supervisor is down, no supervisor to provide them L1 data.
		sys.L2CLA2.NotAdvancedFn(types.CrossSafe, 30), sys.L2CLB2.NotAdvancedFn(types.CrossSafe, 30),
		// sequencer CLs advance their safe heads
		sys.L2CLA.AdvancedFn(types.CrossSafe, delta, 30), sys.L2CLB.AdvancedFn(types.CrossSafe, delta, 30),
		// All the L2CLs advance their unsafe heads
		// Verifiers advances unsafe head because they still have P2P connection with each sequencers
		sys.L2CLA.AdvancedFn(types.LocalUnsafe, delta, 30), sys.L2CLB.AdvancedFn(types.LocalUnsafe, delta, 30),
		sys.L2CLA2.AdvancedFn(types.LocalUnsafe, delta, 30), sys.L2CLB2.AdvancedFn(types.LocalUnsafe, delta, 30),
	)

	// Primary supervisor has safe heads synced with sequencers.
	// After connection, verifiers will sync with primary supervisor, matching supervisor safe head view.
	logger.Info("Connect verifier CLs to primary supervisor to advance verifier safe heads")
	sys.Supervisor.AddManagedL2CL(sys.L2CLA2)
	sys.Supervisor.AddManagedL2CL(sys.L2CLB2)

	// Secondary supervisor and verifiers becomes out-of-sync with safe heads.
	target := max(sys.L2CLA.SafeL2BlockRef().Number, sys.L2CLB.SafeL2BlockRef().Number) + delta
	logger.Info("Every CLs advance safe heads", "delta", delta, "target", target)
	dsl.CheckAll(t,
		sys.L2CLA.ReachedFn(types.CrossSafe, target, 30), sys.L2CLA2.ReachedFn(types.CrossSafe, target, 30),
		sys.L2CLB.ReachedFn(types.CrossSafe, target, 30), sys.L2CLB2.ReachedFn(types.CrossSafe, target, 30),
	)

	logger.Info("Stop primary supervisor to disconnect every CL connection")
	sys.Supervisor.Stop()

	logger.Info("Restart primary supervisor")
	sys.Supervisor.Start()

	logger.Info("No CL connected to supervisor so every CL safe head will not advance")
	dsl.CheckAll(t,
		sys.L2CLA.NotAdvancedFn(types.CrossSafe, 30), sys.L2CLA2.NotAdvancedFn(types.CrossSafe, 30),
		sys.L2CLB.NotAdvancedFn(types.CrossSafe, 30), sys.L2CLB2.NotAdvancedFn(types.CrossSafe, 30),
	)

	// Sequencers will resume advancing safe heads, but not verifiers.
	logger.Info("Reconnect sequencer CLs to primary supervisor")
	sys.Supervisor.AddManagedL2CL(sys.L2CLA)
	sys.Supervisor.AddManagedL2CL(sys.L2CLB)

	logger.Info("Restart secondary supervisor")
	sys.SupervisorSecondary.Start()

	logger.Info("Reconnect verifier CLs to secondary supervisor")
	sys.SupervisorSecondary.AddManagedL2CL(sys.L2CLA2)
	sys.SupervisorSecondary.AddManagedL2CL(sys.L2CLB2)

	// Secondary supervisor will compare its safe head knowledge with L2CLs, and find out L2CLs are ahead of the Secondary supervisor.
	// Secondary supervisor asks the verifiers (L2CL) to rewind(reset) back to match Secondary supervisor safe head view.
	rewind := uint64(3)
	logger.Info("Check verifier CLs safe head rewinded", "rewind", rewind)
	dsl.CheckAll(t,
		sys.L2CLA2.RewindedFn(types.CrossSafe, rewind, 60),
		sys.L2CLB2.RewindedFn(types.CrossSafe, rewind, 60),
	)

	// After rewinding(reset), verifier will advance safe heads again because Secondary supervisor gives L1 data to the verifiers.
	// Wait until verifiers advance safe head enough
	target = max(sys.L2CLA.SafeL2BlockRef().Number, sys.L2CLB.SafeL2BlockRef().Number) + delta
	logger.Info("Every CLs advance safe heads", "delta", delta, "target", target)
	dsl.CheckAll(t,
		sys.L2CLA.ReachedFn(types.CrossSafe, target, 30), sys.L2CLA2.ReachedFn(types.CrossSafe, target, 30),
		sys.L2CLB.ReachedFn(types.CrossSafe, target, 30), sys.L2CLB2.ReachedFn(types.CrossSafe, target, 30),
	)

	// Make sure each chain did not diverge
	require.Equal(sys.L2ELA.BlockRefByNumber(target).Hash, sys.L2ELA2.BlockRefByNumber(target).Hash)
	require.Equal(sys.L2ELB.BlockRefByNumber(target).Hash, sys.L2ELB2.BlockRefByNumber(target).Hash)
}
