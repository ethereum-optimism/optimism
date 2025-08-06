package sync

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

// TestL2CLAheadOfSupervisor tests the below scenario:
// L2CL ahead of supervisor, aka supervisor needs to reset the L2CL, to reproduce old data. Currently supervisor has only indexing mode implemented, so the supervisor will ask the L2CL to reset back.
func TestL2CLAheadOfSupervisor(gt *testing.T) {
	t := devtest.SerialT(gt)

	sys := presets.NewMultiSupervisorInterop(t)
	logger := sys.Log.With("Test", "TestL2CLAheadOfSupervisor")
	require := sys.T.Require()

	// Make sequencers (L2CL), verifiers (L2CL), and supervisors sync for a few blocks.
	// Sequencer and verifier are connected via P2P, which makes their unsafe heads in sync.
	// Both L2CLs are in indexing mode, digesting L1 blocks from the supervisor and reporting unsafe and safe blocks back to the supervisor.
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
	logger.Info("Every CL advance unsafe heads since sequencer and verifier are connected with P2P", "delta", delta)
	dsl.CheckAll(t,
		// verifier CLs cannot advance their safe head because secondary supervisor is down, no supervisor to provide them L1 data.
		sys.L2CLA2.NotAdvancedFn(types.CrossSafe, 15), sys.L2CLB2.NotAdvancedFn(types.CrossSafe, 15),
		// sequencer CLs advance their safe heads
		sys.L2CLA.AdvancedFn(types.CrossSafe, delta, 30), sys.L2CLB.AdvancedFn(types.CrossSafe, delta, 30),
		// All the L2CLs advance their unsafe heads
		// Verifiers advances unsafe head because they still have P2P connection with each sequencers
		sys.L2CLA.AdvancedFn(types.LocalUnsafe, delta, 30), sys.L2CLB.AdvancedFn(types.LocalUnsafe, delta, 30),
		sys.L2CLA2.AdvancedFn(types.LocalUnsafe, delta, 30), sys.L2CLB2.AdvancedFn(types.LocalUnsafe, delta, 30),
	)

	logger.Info("Stop primary supervisor to disconnect every CL connection")
	sys.Supervisor.Stop()

	logger.Info("Restart primary supervisor")
	sys.Supervisor.Start()

	// Primary supervisor has safe heads synced with sequencers.
	// After connection, verifiers will sync with primary supervisor, matching supervisor safe head view.
	logger.Info("Connect verifier CLs to primary supervisor to advance verifier safe heads")
	sys.Supervisor.AddManagedL2CL(sys.L2CLA2)
	sys.Supervisor.AddManagedL2CL(sys.L2CLB2)

	// Secondary supervisor and verifiers becomes out-of-sync with safe heads.
	target := max(sys.L2CLA.SafeL2BlockRef().Number, sys.L2CLB.SafeL2BlockRef().Number) + delta
	logger.Info("Verifiers advances safe heads but not sequencers", "delta", delta, "target", target)
	logger.Info("Every CL advance unsafe heads since sequencer and verifier are connected with P2P", "delta", delta)
	dsl.CheckAll(t,
		// verifier CLs advance their safe heads
		sys.L2CLA2.ReachedFn(types.CrossSafe, target, 30), sys.L2CLB2.ReachedFn(types.CrossSafe, target, 30),
		// sequencer CLs cannot advance their safe head because no supervisor is connected to provide them L1 data.
		sys.L2CLA.NotAdvancedFn(types.CrossSafe, 15), sys.L2CLB.NotAdvancedFn(types.CrossSafe, 15),
		// Verifiers advances unsafe head because they still have P2P connection with each sequencers
		sys.L2CLA.AdvancedFn(types.LocalUnsafe, delta, 30), sys.L2CLB.AdvancedFn(types.LocalUnsafe, delta, 30),
		sys.L2CLA2.AdvancedFn(types.LocalUnsafe, delta, 30), sys.L2CLB2.AdvancedFn(types.LocalUnsafe, delta, 30),
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
		sys.L2CLA2.RewindedFn(types.CrossSafe, rewind, 120),
		sys.L2CLB2.RewindedFn(types.CrossSafe, rewind, 120),
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

// TestUnsafeChainKnownToL2CL tests the below scenario:
// supervisor cross-safe ahead of L2CL cross-safe, aka L2CL can "skip" forward to match safety of supervisor.
func TestUnsafeChainKnownToL2CL(gt *testing.T) {
	gt.Skip("TODO(#16972): skipping due to flakiness and impending op-node/supervisor refactor")

	t := devtest.SerialT(gt)

	sys := presets.NewMultiSupervisorInterop(t)
	logger := sys.Log.With("Test", "TestUnsafeChainKnownToL2CL")
	require := sys.T.Require()

	logger.Info("Make sure verifier safe head advances")
	dsl.CheckAll(t,
		sys.L2CLA.AdvancedFn(types.CrossSafe, 5, 30),
		sys.L2CLA2.AdvancedFn(types.CrossSafe, 5, 30),
	)

	safeA2 := sys.L2ELA2.BlockRefByLabel(eth.Safe)
	logger.Info("Verifier advanced safe head", "number", safeA2.Number)
	unsafeA2 := sys.L2ELA2.BlockRefByLabel(eth.Unsafe)
	logger.Info("Verifier advanced unsafe head", "number", unsafeA2.Number)

	// The verifier stops advancing unsafe head because it will not receive unsafe heads via P2P, and can only update unsafe heads matching with safe heads by reading L1 batches,
	// The verifier safe head will lag behind or match the sequencer and primary supervisor because all three components share the same L1 view.
	logger.Info("Disconnect p2p between L2CLs")
	sys.L2CLA.DisconnectPeer(sys.L2CLA2)
	sys.L2CLA2.DisconnectPeer(sys.L2CLA)

	// For making verifier not sync at all, both unsafe head and safe head
	// The sequencer will advance unsafe head and safe head, as well as synced with primary supervisor.
	logger.Info("Stop verifier")
	sys.L2CLA2.Stop()

	// Wait until sequencer and primary supervisor diverged enough from the verifier.
	// To make the verifier held unsafe blocks are already as safe by sequencer and primary supervisor, we wait.
	delta := uint64(10)
	logger.Info("Wait until supervisor reaches safe head", "delta", delta)
	sys.Supervisor.AdvancedSafeHead(sys.L2ChainA.ChainID(), delta, 30)

	// Restarted verifier will advance its unsafe head and safe head by reading L1 but not by P2P
	logger.Info("Restart verifier")
	sys.L2CLA2.Start()

	safeA2 = sys.L2ELA2.BlockRefByLabel(eth.Safe)
	logger.Info("Verifier safe head after restart", "number", safeA2.Number)
	unsafeA2 = sys.L2ELA2.BlockRefByLabel(eth.Unsafe)
	logger.Info("Verifier unsafe head after restart", "number", unsafeA2.Number)

	// Make sure there are unsafe blocks to be consolidated:
	// To check verifier does not have to process blocks since unsafe blocks are already processed
	require.Greater(unsafeA2.Number, safeA2.Number)

	// The verifier will quickly catch up with the sequencer safe head as well as the primary supervisor.
	// The verifier will "skip" processing already known unsafe blocks, and consolidate them into safe blocks.
	logger.Info("Make sure verifier unsafe head was consolidated to safe")
	sys.L2CLA2.Reached(types.CrossSafe, unsafeA2.Number, 30)

	safeA := sys.L2ELA.BlockRefByLabel(eth.Safe)
	target := safeA.Number + delta
	logger.Info("Make sure verifier unsafe head advances due to safe head advances", "target", target, "delta", delta)
	sys.L2CLA2.Reached(types.LocalUnsafe, target, 30)

	block := sys.L2ELA2.BlockRefByNumber(unsafeA2.Number)
	require.Equal(unsafeA2.Hash, block.Hash)

	// Cleanup
	logger.Info("Explicit reconnection of L2CL P2P between sequencer and verifier")
	sys.L2CLA2.ConnectPeer(sys.L2CLA)
	sys.L2CLA.ConnectPeer(sys.L2CLA2)
}

// TestUnsafeChainUnknownToL2CL tests the below scenario:
// supervisor unsafe ahead of L2CL unsafe, aka L2CL processes new blocks first.
func TestUnsafeChainUnknownToL2CL(gt *testing.T) {
	t := devtest.SerialT(gt)

	sys := presets.NewMultiSupervisorInterop(t)
	logger := sys.Log.With("Test", "TestUnsafeChainUnknownToL2CL")

	logger.Info("Make sure sequencer and verifier unsafe head advances")
	dsl.CheckAll(t,
		sys.L2CLA.AdvancedFn(types.LocalUnsafe, 5, 30),
		sys.L2CLA2.AdvancedFn(types.LocalUnsafe, 5, 30),
	)

	logger.Info("Disconnect p2p between L2CLs")
	sys.L2CLA.DisconnectPeer(sys.L2CLA2)
	sys.L2CLA2.DisconnectPeer(sys.L2CLA)

	// verifier lost its P2P connection with sequencer, and will advance its unsafe head by reading L1 but not by P2P
	logger.Info("Make sure verifier advances safe head by reading L1")
	sys.L2CLA2.Advanced(types.CrossSafe, 5, 30)

	// The verifier will not receive unsafe heads via P2P, and can only update unsafe heads matching with safe heads by reading L1 batches.
	logger.Info("Verifier heads will lag compared from sequencer heads and primary supervisor view")
	dsl.CheckAll(t,
		sys.L2CLA2.LaggedFn(sys.L2CLA, types.LocalUnsafe, 10, false),
		sys.L2CLA2.LaggedFn(sys.Supervisor, types.LocalUnsafe, 10, true),
	)

	logger.Info("Explicit reconnection of L2CL P2P between sequencer and verifier")
	sys.L2CLA2.ConnectPeer(sys.L2CLA)
	sys.L2CLA.ConnectPeer(sys.L2CLA2)

	// The sequencer will broadcast all unknown unsafe blocks to the verifier.
	// The verifier will quickly catch up with the sequencer unsafe head as well as the primary supervisor.
	// The verifier will process previously unknown unsafe blocks and advance its unsafe head.
	logger.Info("Verifier catches up sequencer unsafe chain with was unknown for verifier")
	sys.L2CLA2.Matched(sys.L2CLA, types.LocalUnsafe, 5)
}

// TestL2CLSyncP2P checks that unsafe head is propagated from sequencer to verifier.
// Tests started/restarted L2CL advances unsafe head via P2P connection.
func TestL2CLSyncP2P(gt *testing.T) {
	t := devtest.SerialT(gt)

	sys := presets.NewMultiSupervisorInterop(t)
	logger := sys.Log.With("Test", "TestL2CLSyncP2P")

	logger.Info("Make sure sequencer and verifier unsafe head advances")
	dsl.CheckAll(t,
		sys.L2CLA.AdvancedFn(types.LocalUnsafe, 5, 30),
		sys.L2CLA2.AdvancedFn(types.LocalUnsafe, 5, 30),
	)

	logger.Info("Stop verifier CL")
	sys.L2CLA2.Stop()

	logger.Info("Make sure verifier EL does not advance")
	sys.L2ELA2.NotAdvanced(eth.Unsafe)

	logger.Info("Restart verifier CL")
	sys.L2CLA2.Start()

	logger.Info("Explicit reconnection of L2CL P2P between sequencer and verifier")
	sys.L2CLA.ConnectPeer(sys.L2CLA2)
	sys.L2CLA2.ConnectPeer(sys.L2CLA)

	logger.Info("Make sure verifier EL advances")
	dsl.CheckAll(t,
		sys.L2CLA.AdvancedFn(types.LocalUnsafe, 10, 30),
		sys.L2CLA2.AdvancedFn(types.LocalUnsafe, 10, 30),
	)

	logger.Info("Check sequencer and verifier holds identical chain")
	sys.L2CLA2.Matched(sys.L2CLA, types.LocalUnsafe, 30)
}
