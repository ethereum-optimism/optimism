package interop

import (
	"context"
	"math/rand"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/core/types"

	"github.com/ethereum-optimism/optimism/devnet-sdk/contracts/constants"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/txintent"
)

// TestSupernodeInteropInvalidMessageReset tests that:
// WHEN: an invalid Executing Message is included in a chain
// THEN:
// - The interop activity detects the invalid block
// - The chain container is told to invalidate the block
// - A reset/rewind is triggered if the chain is using that block
//
// Note: This test observes reset behavior. Full block replacement requires
// re-derivation which is a separate mechanism.
func TestSupernodeInteropInvalidMessageReset(gt *testing.T) {
	gt.Skip("Skipped: logsDB consistency issues blocking interop progression - see #18944")

	t := devtest.SerialT(gt)
	sys := presets.NewTwoL2SupernodeInterop(t, 0)

	ctx := t.Ctx()
	snClient := sys.SuperNodeClient()

	// Create funded EOAs on both chains
	alice := sys.FunderA.NewFundedEOA(eth.OneEther)
	bob := sys.FunderB.NewFundedEOA(eth.OneEther)

	// Deploy event logger on chain A
	eventLoggerA := alice.DeployEventLogger()

	// Sync chains
	sys.L2B.CatchUpTo(sys.L2A)
	sys.L2A.CatchUpTo(sys.L2B)

	rng := rand.New(rand.NewSource(12345))

	// Send an initiating message on chain A
	initTrigger := randomInitTrigger(rng, eventLoggerA, 2, 10)
	initTx, initReceipt := alice.SendInitMessage(initTrigger)

	t.Logger().Info("initiating message sent on chain A",
		"block", initReceipt.BlockNumber,
		"hash", initReceipt.BlockHash,
	)

	// Wait for chain B to catch up
	sys.L2B.WaitForBlock()

	// Record the verified timestamp before the invalid message
	// We need to know what timestamp was verified before the invalid exec message
	blockTime := sys.L2A.Escape().RollupConfig().BlockTime
	genesisTime := sys.L2A.Escape().RollupConfig().Genesis.L2Time

	// Wait for some timestamps to be verified first
	targetTimestamp := genesisTime + blockTime*2
	t.Require().Eventually(func() bool {
		resp, err := snClient.SuperRootAtTimestamp(ctx, targetTimestamp)
		if err != nil {
			return false
		}
		t.Logger().Info("super root at timestamp", "timestamp", targetTimestamp, "data", resp.Data)
		return resp.Data != nil
	}, 60*time.Second, time.Second, "initial timestamps should be verified")

	t.Logger().Info("initial verification confirmed", "timestamp", targetTimestamp)

	// Send an INVALID executing message on chain B
	// Modify the message identifier to make it invalid (wrong log index)
	invalidExecReceipt := sendInvalidExecMessageForReset(t, bob, initTx, 0)

	invalidBlockNumber := bigs.Uint64Strict(invalidExecReceipt.BlockNumber)
	invalidBlockHash := invalidExecReceipt.BlockHash
	invalidBlock := sys.L2ELB.BlockRefByHash(invalidExecReceipt.BlockHash)
	invalidBlockTimestamp := invalidBlock.Time

	t.Logger().Info("invalid executing message sent on chain B",
		"block", invalidBlockNumber,
		"hash", invalidBlockHash,
		"timestamp", invalidBlockTimestamp,
	)

	// Record the initial unsafe head for chain B
	initialStatusB := sys.L2BCL.SyncStatus()
	initialUnsafeB := initialStatusB.UnsafeL2.Number

	t.Logger().Info("initial status before reset observation",
		"chainB_unsafe", initialUnsafeB,
		"invalid_block", invalidBlockNumber,
	)

	// Observe for reset behavior:
	// When the interop activity detects the invalid message and calls InvalidateBlock,
	// it will trigger a rewind. We observe by watching for the unsafe head to go backwards
	// or for the block at the invalid block number to change.

	observationDuration := 60 * time.Second
	checkInterval := time.Second

	start := time.Now()
	var resetDetected bool
	var lastUnsafeB uint64 = initialUnsafeB

	for time.Since(start) < observationDuration {
		time.Sleep(checkInterval)

		statusB := sys.L2BCL.SyncStatus()
		currentUnsafeB := statusB.UnsafeL2.Number

		// Check if the unsafe head went backwards (reset occurred)
		if currentUnsafeB < lastUnsafeB && lastUnsafeB >= invalidBlockNumber {
			resetDetected = true
			t.Logger().Info("RESET DETECTED! Unsafe head moved backward",
				"previous_unsafe", lastUnsafeB,
				"current_unsafe", currentUnsafeB,
				"invalid_block", invalidBlockNumber,
			)
		}

		// Also check if the block hash at the invalid block number changed
		currentBlock := sys.L2ELB.BlockRefByNumber(invalidBlockNumber)
		if currentBlock.Hash != invalidBlockHash {
			resetDetected = true
			t.Logger().Info("RESET DETECTED! Block hash changed",
				"block_number", invalidBlockNumber,
				"old_hash", invalidBlockHash,
				"new_hash", currentBlock.Hash,
			)
		}

		// Check verification status
		resp, err := snClient.SuperRootAtTimestamp(ctx, invalidBlockTimestamp)
		t.Require().NoError(err, "SuperRootAtTimestamp should not error")

		t.Logger().Info("observation tick",
			"elapsed", time.Since(start).Round(time.Second),
			"chainB_unsafe", currentUnsafeB,
			"invalid_block_ts", invalidBlockTimestamp,
			"reset_detected", resetDetected,
			"verified", resp.Data != nil,
		)

		lastUnsafeB = currentUnsafeB

		// Exit early if we detect reset
		if resetDetected {
			t.Logger().Info("Reset behavior confirmed")
			break
		}
	}

	// ASSERTION: Reset should have been detected
	// (either unsafe head went backward or block hash changed)
	t.Require().True(resetDetected,
		"reset should be triggered when invalid block is detected")

	// ASSERTION: The invalid block's timestamp should NOT be verified
	// (because the reset means the block is no longer valid)
	finalResp, err := snClient.SuperRootAtTimestamp(ctx, invalidBlockTimestamp)
	t.Require().NoError(err)
	t.Require().Nil(finalResp.Data,
		"invalid block timestamp should not be verified after reset")

	t.Logger().Info("test complete: reset was triggered for invalid block",
		"invalid_block_number", invalidBlockNumber,
		"invalid_block_hash", invalidBlockHash,
	)
}

// sendInvalidExecMessageForReset sends an executing message with a modified (invalid) identifier.
// This makes the message invalid because it references a non-existent log index.
func sendInvalidExecMessageForReset(
	t devtest.T,
	bob *dsl.EOA,
	initIntent *txintent.IntentTx[*txintent.InitTrigger, *txintent.InteropOutput],
	eventIdx int,
) *types.Receipt {
	ctx := t.Ctx()

	// Evaluate the init result to get the message entries
	result, err := initIntent.Result.Eval(ctx)
	t.Require().NoError(err, "failed to evaluate init result")
	t.Require().Greater(len(result.Entries), eventIdx, "event index out of range")

	// Get the message and modify it to be invalid
	msg := result.Entries[eventIdx]

	// Make the message invalid by setting an impossible log index
	// This creates a message that claims to reference a log that doesn't exist
	msg.Identifier.LogIndex = 9999

	// Create the exec trigger with the invalid message
	execTrigger := &txintent.ExecTrigger{
		Executor: constants.CrossL2Inbox,
		Msg:      msg,
	}

	// Create the intent with the invalid trigger
	tx := txintent.NewIntent[*txintent.ExecTrigger, *txintent.InteropOutput](bob.Plan())
	tx.Content.DependOn(&initIntent.Result)
	tx.Content.Fn(func(ctx context.Context) (*txintent.ExecTrigger, error) {
		return execTrigger, nil
	})

	receipt, err := tx.PlannedTx.Included.Eval(ctx)
	t.Require().NoError(err, "invalid exec msg receipt not found")
	t.Logger().Info("invalid exec message included", "chain", bob.ChainID(), "block", receipt.BlockNumber)

	return receipt
}
