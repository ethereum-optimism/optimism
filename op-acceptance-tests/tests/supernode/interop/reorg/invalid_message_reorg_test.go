package reorg

import (
	"errors"
	"math/rand"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/eth/safety"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
)

// TestSupernodeInteropInvalidMessageReplacement runs the invalid-message
// replacement scenario with the supernode virtual sequencer.
func TestSupernodeInteropInvalidMessageReplacement(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewTwoL2SupernodeInterop(t, 0)
	runInteropInvalidMessageReplacementScenario(t, sys)
}

// TestSupernodeLightSequencerInteropInvalidMessageReplacement is the follow-mode
// (light op-node CL sequencer) analogue of TestSupernodeInteropInvalidMessageReplacement:
// the light CLs sequence on their own ELs and follow the shared supernode's safe head via
// EL sync. See https://github.com/ethereum-optimism/optimism/issues/21119.
//
// op-reth only. On op-geth the follower never adopts the replacement (cross-safe stalls,
// follow-source reorgs forever); the virtual variant passes there, so it's specific to the
// light-sequencer path. op-geth is being deprecated, so we skip rather than block on it.
func TestSupernodeLightSequencerInteropInvalidMessageReplacement(gt *testing.T) {
	t := devtest.SerialT(gt)
	// op-reth only: on op-geth the follower never adopts the deposits-only replacement.
	sysgo.SkipOnOpGeth(t, "op-geth does not adopt the invalid-message replacement on the light path (#21119)")
	// Bootstrap via the supernode VN sequencer and hand off to the light ELSync sequencers, then
	// run the invalid-message scenario on the live chain.
	sys := presets.NewTwoL2SupernodeLightSequencerInterop(t, 0, presets.WithSupernodeVNSequencerForBootstrap())
	sys.BootstrapLightSequencersViaVNHandoff()
	runInteropInvalidMessageReplacementScenario(t, sys)
}

// runInteropInvalidMessageReplacementScenario drives the invalid-message replacement
// scenario against an already-constructed two-L2 supernode interop system, so the
// caller owns the sequencer topology (virtual sequencer vs light op-node follow-CL).
//
// WHEN: an invalid Executing Message is included in a chain
// THEN:
// - The interop activity detects the invalid block
// - The chain container is told to invalidate the block
// - A reset/rewind is triggered if the chain is using that block
// - A replacement block is built at the same height (deposits-only)
// - The replacement block's timestamp eventually becomes verified
func runInteropInvalidMessageReplacementScenario(t devtest.T, sys *presets.TwoL2SupernodeInterop) {
	ctx := t.Ctx()

	// Create funded EOAs on both chains
	alice := sys.FunderA.NewFundedEOA(eth.OneEther)
	bob := sys.FunderB.NewFundedEOA(eth.OneEther)

	// Deploy event logger on chain A
	eventLoggerA := alice.DeployEventLogger()

	// Sync chains
	sys.L2B.CatchUpTo(sys.L2A)
	sys.L2A.CatchUpTo(sys.L2B)

	// Pause interop and verify it has stopped
	// Uses max local safe timestamp from both chains, pauses at +10, awaits validation at +9
	paused := sys.Supernode.EnsureInteropPaused(sys.L2ACL, sys.L2BCL, 10)
	t.Logger().Info("interop paused", "paused", paused)

	rng := rand.New(rand.NewSource(12345))

	// Send an initiating message on chain A
	initMsg := alice.SendRandomInitMessage(rng, eventLoggerA, 2, 10)

	t.Logger().Info("initiating message sent on chain A",
		"block", initMsg.BlockNumber(),
		"hash", initMsg.BlockHash(),
	)

	// Wait for chain B to catch up
	sys.L2B.WaitForBlock()

	// Send an INVALID executing message on chain B
	execMsg := bob.SendInvalidExecMessage(initMsg)
	invalidBlockNumber := bigs.Uint64Strict(execMsg.BlockNumber())
	invalidBlockHash := execMsg.BlockHash()
	invalidBlockTimestamp := sys.L2B.TimestampForBlockNum(invalidBlockNumber)
	t.Logger().Info("invalid executing message sent on chain B",
		"block", invalidBlockNumber,
		"hash", invalidBlockHash,
		"timestamp", invalidBlockTimestamp,
	)

	// Wait for local safety to include the invalid block
	require.Eventually(t, func() bool {
		numSafe := sys.L2BCL.SyncStatus().LocalSafeL2.Number >= invalidBlockNumber
		return numSafe
	}, 60*time.Second, time.Second, "invalid block should become locally safe")

	// Resume interop and observe reorg
	// Interop activity will proceed and invalidate the block, triggering a rewind, and building a replacement block
	// We observe resets and replacements, but only proceed on replacement (we may miss reset if it happens quickly)
	sys.Supernode.ResumeInterop()
	require.Eventually(t, func() bool {
		// Check if the block hash at the invalid block number changed or block doesn't exist
		// Use the EthClient directly to handle errors (block may not exist after rewind)
		currentBlock, err := sys.L2ELB.Escape().EthClient().BlockRefByNumber(ctx, invalidBlockNumber)
		if err != nil {
			if errors.Is(eth.MaybeAsNotFoundErr(err), ethereum.NotFound) {
				t.Logger().Info("RESET DETECTED! Block no longer exists (rewound)",
					"block_number", invalidBlockNumber,
				)
			} else {
				t.Logger().Warn("unexpected error checking block",
					"block_number", invalidBlockNumber,
					"err", err,
				)
			}
		} else if currentBlock.Hash != invalidBlockHash {
			t.Logger().Info("RESET DETECTED! Block hash changed",
				"block_number", invalidBlockNumber,
				"old_hash", invalidBlockHash,
				"new_hash", currentBlock.Hash,
			)
			return true
		}
		return false
	}, 60*time.Second, time.Second, "reset should be detected")

	// Wait for interop to proceed and verify the replacement block at the timestamp
	sys.Supernode.AwaitValidatedTimestamp(invalidBlockTimestamp)

	// Settle before asserting: the follow-source unsafe head oscillates while cross-safe is pinned
	// at the replacement. Gate on cross-safe advancing past it — a match alone passes trivially
	// while both sides are pinned — so reads below see the settled chain.
	dsl.CheckAll(t,
		sys.L2ACL.AdvancedFn(safety.CrossSafe, 3, 45),
		sys.L2BCL.AdvancedFn(safety.CrossSafe, 3, 45),
	)
	dsl.CheckAll(t,
		sys.L2ACL.MatchedFn(sys.L2ASupernodeCL, safety.CrossSafe, 30),
		sys.L2BCL.MatchedFn(sys.L2BSupernodeCL, safety.CrossSafe, 30),
	)

	// The invalid exec-message tx must be gone from the replacement block on BOTH the light
	// sequencer's EL and the supernode VN's EL — distinct nodes joined only by L1 + P2P, so
	// agreement proves one canonical chain. AssertTxNotInBlock reads by number (the oscillating
	// unsafe head), so gate each read on that EL's safe head reaching the block first; blocks
	// at/below safe are irreversible.
	sys.L2ELB.Reached(eth.Safe, invalidBlockNumber, 30)
	sys.L2ELB.AssertTxNotInBlock(invalidBlockNumber, execMsg.Receipt.TxHash)
	sys.L2BSupernodeEL.Reached(eth.Safe, invalidBlockNumber, 30)
	sys.L2BSupernodeEL.AssertTxNotInBlock(invalidBlockNumber, execMsg.Receipt.TxHash)

	t.Logger().Info("test complete: invalid block was replaced and verified",
		"invalid_block_number", invalidBlockNumber,
		"invalid_block_hash", invalidBlockHash,
	)

	// A new tx on the recovered chain must still be includable and durably validated. The light
	// sequencer's unsafe tip oscillates during recovery, so a tx sent into it can be orphaned
	// even as the chain keeps advancing. Re-send the transfer until one lands at or below the
	// supernode VN's L1-derived safe head, then assert it is present on BOTH ELs.
	settledBlock, settledTxHash := sys.L2BSupernodeEL.ResendUntilSafe(func() *txplan.PlannedTx {
		// Fresh funded account per attempt => clean nonce space, immune to an orphaned prior send.
		eoa := sys.FunderB.NewFundedEOA(eth.OneEther)
		return txplan.NewPlannedTx(eoa.PlanTransfer(alice.Address(), eth.OneHundredthEther))
	}, 8, 20)
	sys.Supernode.AwaitValidatedTimestamp(sys.L2B.TimestampForBlockNum(settledBlock))
	// The tx is in a derived-safe block on the supernode; the light sequencer must agree there.
	// Wait for the light seq EL's safe head to reach it.
	sys.L2BSupernodeEL.AssertTxInBlock(settledBlock, settledTxHash)
	sys.L2ELB.Reached(eth.Safe, settledBlock, 30)
	sys.L2ELB.AssertTxInBlock(settledBlock, settledTxHash)
}
