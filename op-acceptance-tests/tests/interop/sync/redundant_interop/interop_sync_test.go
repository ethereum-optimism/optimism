package sync

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

// TestUnsafeChainKnownToL2CL tests the below scenario:
// supervisor cross-safe ahead of L2CL cross-safe, aka L2CL can "skip" forward to match safety of supervisor.
func TestUnsafeChainKnownToL2CL(gt *testing.T) {
	t := devtest.SerialT(gt)

	// Sequencer and verifier are connected via P2P, which makes their unsafe heads in sync.
	// Both L2CLs are in managed mode, digesting L1 blocks from the supervisor and reporting unsafe and safe blocks back to the supervisor.
	sys := presets.NewRedundantInterop(t)
	logger := sys.Log.With("Test", "TestUnsafeChainKnownToL2CL")
	require := sys.T.Require()

	logger.Info("Make sure verifier safe head advances")
	dsl.CheckAll(t,
		sys.L2CLA.Advanced(types.CrossSafe, 5, 30),
		sys.L2CLA2.Advanced(types.CrossSafe, 5, 30),
	)

	safeA2 := sys.L2ELA2.BlockRefByLabel(eth.Safe)
	logger.Info("Verifier advanced safe head", "number", safeA2.Number)
	unsafeA2 := sys.L2ELA2.BlockRefByLabel(eth.Unsafe)
	logger.Info("Verifier advanced unsafe head", "number", unsafeA2.Number)

	// The verifier stops advancing unsafe head because it will not receive unsafe heads via P2P, and can only update unsafe heads matching with safe heads by reading L1 batches,
	// The verifier safe head will lag behind or match the sequencer and supervisor because all three components share the same L1 view.
	logger.Info("Disconnect p2p between L2CLs")
	sys.L2CLA.DisconnectPeer(sys.L2CLA2)
	sys.L2CLA2.DisconnectPeer(sys.L2CLA)

	// For making verifier not sync at all, both unsafe head and safe head
	// The sequencer will advance unsafe head and safe head, as well as synced with supervisor.
	logger.Info("stop verifier")
	sys.L2CLA2.Stop()

	// Wait until sequencer and supervisor diverged enough from the verifier.
	// To make the verifier held unsafe blocks are already as safe by sequencer and supervisor, we wait.
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

	// The verifier will quickly catch up with the sequencer safe head as well as the supervisor.
	// The verifier will "skip" processing already known unsafe blocks, and consolidate them into safe blocks.
	logger.Info("Make sure verifier unsafe head was consolidated to safe")
	dsl.CheckAll(t, sys.L2CLA2.Reached(types.CrossSafe, unsafeA2.Number, 30))

	safeA := sys.L2ELA.BlockRefByLabel(eth.Safe)
	target := safeA.Number + delta
	logger.Info("Make sure verifier unsafe head advances due to safe head advances", "target", target, "delta", delta)
	dsl.CheckAll(t, sys.L2CLA2.Reached(types.LocalUnsafe, target, 30))

	block := sys.L2ELA2.BlockRefByNumber(unsafeA2.Number)
	require.Equal(unsafeA2.Hash, block.Hash)
}
