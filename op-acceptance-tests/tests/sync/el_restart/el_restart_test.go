package el_restart

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

// TestELRestartRecovery verifies that when an op-reth EL is restarted,
// the op-node CL detects the state divergence (SYNCING response from FCU)
// and triggers a reset to recover, catching up to the sequencer.
//
// This is a regression test for the bug where op-node unconditionally set
// needFCUCall=false after any FCU response, causing a permanent stall when
// the EL returned SYNCING after a restart.
func TestELRestartRecovery(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewSingleChainMultiNodeWithoutCheck(t)

	// 1. Wait for both nodes to be synced and advancing
	dsl.CheckAll(t,
		sys.L2CL.AdvancedFn(types.LocalUnsafe, 5, 30),
		sys.L2CLB.AdvancedFn(types.LocalUnsafe, 5, 30),
	)

	// Record the verifier's state before stopping
	verifierHeadBefore := sys.L2ELB.BlockRefByLabel(eth.Unsafe)
	t.Logf("Verifier EL unsafe head before stop: %d (%s)", verifierHeadBefore.Number, verifierHeadBefore.Hash)

	// 2. Stop the verifier EL (op-reth) — the CL stays running and will
	//    get temporary errors until the EL comes back.
	t.Logf("Stopping verifier EL (op-reth)")
	sys.L2ELB.Stop()
	t.Cleanup(func() {
		// Ensure the EL is restarted even if the test fails, so cleanup works.
		sys.L2ELB.Start()
	})

	// 3. Let the sequencer produce several more blocks while verifier EL is down.
	//    This creates a gap between the CL's cached heads and the EL's actual state.
	blocksToAdvance := uint64(10)
	t.Logf("Waiting for sequencer to advance %d blocks while verifier EL is down", blocksToAdvance)
	sys.L2CL.Advanced(types.LocalUnsafe, blocksToAdvance, 60)

	seqHead := sys.L2EL.BlockRefByLabel(eth.Unsafe)
	t.Logf("Sequencer EL unsafe head after gap: %d (%s)", seqHead.Number, seqHead.Hash)

	// 4. Start the verifier EL back up.
	//    op-reth restarts with persisted state, which may be behind the CL's cached heads.
	//    When op-node sends FCU with its cached heads, op-reth returns SYNCING.
	//    With the fix, op-node detects SYNCING in CL-sync mode and triggers a reset
	//    via FindL2Heads to re-discover the EL's actual chain state.
	t.Logf("Starting verifier EL (op-reth)")
	sys.L2ELB.Start()

	// Re-establish EL P2P peering so the verifier EL can sync blocks from the sequencer EL.
	sys.L2ELB.PeerWith(sys.L2EL)

	// 5. Wait for the verifier to recover and catch up.
	//    Without the fix, the verifier would stall permanently here.
	t.Logf("Waiting for verifier to recover and catch up to block %d", seqHead.Number)
	dsl.CheckAll(t,
		sys.L2ELB.ReachedFn(eth.Unsafe, seqHead.Number, 120),
		sys.L2CLB.ReachedFn(types.LocalUnsafe, seqHead.Number, 120),
	)

	verifierHeadAfter := sys.L2ELB.BlockRefByLabel(eth.Unsafe)
	t.Logf("Verifier EL recovered: unsafe head at %d (%s)", verifierHeadAfter.Number, verifierHeadAfter.Hash)

	t.Require().GreaterOrEqual(verifierHeadAfter.Number, seqHead.Number,
		"verifier should have caught up to sequencer's unsafe head")
}
