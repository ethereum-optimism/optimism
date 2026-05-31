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
	"github.com/ethereum-optimism/optimism/op-service/retry"
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
// (light op-node CL sequencer) analogue of TestSupernodeInteropInvalidMessageReplacement.
// The light CLs sequence unsafe blocks on their own ELs and follow the shared
// supernode's safe head via EL sync. It asserts the follower adopts the supernode's
// deposits-only replacement after an invalid executing message and the chain keeps
// making validated progress. See https://github.com/ethereum-optimism/optimism/issues/21119.
//
// op-reth only: on op-geth this scenario consistently does not converge (0/12 locally).
// After the deposits-only replacement, the op-geth follower never adopts the supernode's
// replacement block — cross-safe stalls at the replacement height while the sequencer
// keeps building a divergent chain above it, so the follow-source reorg fires every tick
// and no block past the replacement is ever validated. The new transaction therefore
// never gets a stable receipt (`tx.Included` fails with "after 5 attempts: not found").
// The virtual-sequencer variant above passes on op-geth, so this is specific to the
// light-sequencer follow path. op-geth is being deprecated, so we skip rather than block
// on it; tracked for follow-up.
func TestSupernodeLightSequencerInteropInvalidMessageReplacement(gt *testing.T) {
	t := devtest.SerialT(gt)
	sysgo.SkipOnOpGeth(t, "light-sequencer follow-mode reorg recovery does not converge on op-geth (op-geth deprecating); see #21119")
	sys := presets.NewTwoL2SupernodeLightSequencerInterop(t, 0)
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

	// ASSERTION: The invalid transaction no longer exists in the chain
	// The invalid exec message transaction should NOT be in the replacement block
	sys.L2ELB.AssertTxNotInBlock(invalidBlockNumber, execMsg.Receipt.TxHash)

	t.Logger().Info("test complete: invalid block was replaced and verified",
		"invalid_block_number", invalidBlockNumber,
		"invalid_block_hash", invalidBlockHash,
	)

	// CONVERGENCE & STABILITY (settle first): every node on the reorged chain must agree on
	// the single canonical replacement chain AND keep building on top of it — no oscillation
	// back onto the invalidated fork (the #21119 failure mode). Each chain's block producer
	// (L2{A,B}CL — the supernode's virtual sequencer, or the light op-node sequencer in
	// follow-mode presets) must converge with the supernode's verifier route
	// (L2{A,B}SupernodeCL) at cross-safe while its local-unsafe head keeps advancing, and
	// agree at local-safe. MatchedWithProgressFn is progress-aware — it fails on a stall, not
	// a wall-clock deadline — so it tolerates a slow/loaded executor while still catching a
	// genuine non-convergence stall. We settle here before transacting below, so the new tx
	// lands on an already-converged chain instead of racing the in-flight reorgs.
	dsl.CheckAll(t,
		sys.L2BCL.MatchedWithProgressFn(sys.L2BSupernodeCL, safety.CrossSafe, safety.LocalUnsafe, 90*time.Second, 30*time.Second),
		sys.L2ACL.MatchedWithProgressFn(sys.L2ASupernodeCL, safety.CrossSafe, safety.LocalUnsafe, 90*time.Second, 30*time.Second),
	)
	dsl.CheckAll(t,
		sys.L2BCL.MatchedFn(sys.L2BSupernodeCL, safety.LocalSafe, 30),
		sys.L2ACL.MatchedFn(sys.L2ASupernodeCL, safety.LocalSafe, 30),
	)

	// With the chain settled on the replacement, a new transaction must still be includable
	// and fully validated. On the now-converged chain it lands cleanly, so the idiomatic
	// timestamp-validation flow (the same one used for the replacement block above) is
	// robust: include it, wait for the supernode to validate its timestamp (which makes its
	// block cross-safe and reorg-immune), and confirm it is still in its block. We widen the
	// inclusion budget from the default 5 to 10 attempts (matching the contention-sensitive
	// txs in isthmus/superroot) so a saturated CI executor with slow block production still
	// produces the receipt — this is an attempt-bounded retry, not a wall-clock deadline.
	bruce := sys.FunderB.NewFundedEOA(eth.OneEther)
	tx := bruce.Transact(
		bruce.PlanTransfer(alice.Address(), eth.OneHundredthEther),
		txplan.WithRetryInclusion(sys.L2ELB.Escape().EthClient(), 10, retry.Exponential()),
	)
	txBlock := bigs.Uint64Strict(tx.Included.Value().BlockNumber)
	txTimestamp := sys.L2B.TimestampForBlockNum(txBlock)
	sys.Supernode.AwaitValidatedTimestamp(txTimestamp)
	sys.L2ELB.AssertTxInBlock(txBlock, tx.Included.Value().TxHash)
}
