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

// TestSupernodeInteropInvalidMessageReplacement tests that:
// WHEN: an invalid Executing Message is included in a chain
// THEN:
// - The block containing the invalid message gets reset backward
// - The chain re-derives and produces a replacement block
// - Validity eventually advances past the replaced block
//
// This test verifies the block invalidation and replacement mechanism.
func TestSupernodeInteropInvalidMessageReplacement(gt *testing.T) {
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
	invalidExecReceipt := sendInvalidExecMessageForReplacement(t, bob, initTx, 0)

	invalidBlockNumber := bigs.Uint64Strict(invalidExecReceipt.BlockNumber)
	invalidBlockHash := invalidExecReceipt.BlockHash
	invalidBlock := sys.L2ELB.BlockRefByHash(invalidExecReceipt.BlockHash)
	invalidBlockTimestamp := invalidBlock.Time

	t.Logger().Info("invalid executing message sent on chain B",
		"block", invalidBlockNumber,
		"hash", invalidBlockHash,
		"timestamp", invalidBlockTimestamp,
	)

	// Record the safety status before waiting
	initialStatusA := sys.L2ACL.SyncStatus()
	initialStatusB := sys.L2BCL.SyncStatus()

	t.Logger().Info("initial safety status",
		"chainA_local_safe", initialStatusA.LocalSafeL2.Number,
		"chainA_unsafe", initialStatusA.UnsafeL2.Number,
		"chainB_local_safe", initialStatusB.LocalSafeL2.Number,
		"chainB_unsafe", initialStatusB.UnsafeL2.Number,
	)

	// Now we verify the key behaviors:
	// 1. The invalid block should be replaced with a different block at the same height
	// 2. Validity should eventually advance past the replaced block

	observationDuration := 60 * time.Second
	checkInterval := time.Second

	start := time.Now()
	var replacementDetected bool
	var replacementBlockHash [32]byte

	for time.Since(start) < observationDuration {
		time.Sleep(checkInterval)

		// Check if the block at the invalid block number has changed
		currentBlock := sys.L2ELB.BlockRefByNumber(invalidBlockNumber)

		if currentBlock.Hash != invalidBlockHash {
			replacementDetected = true
			replacementBlockHash = currentBlock.Hash
			t.Logger().Info("REPLACEMENT DETECTED!",
				"block_number", invalidBlockNumber,
				"old_hash", invalidBlockHash,
				"new_hash", currentBlock.Hash,
			)
		}

		// Check current safety status
		statusA := sys.L2ACL.SyncStatus()
		statusB := sys.L2BCL.SyncStatus()

		// Check if the invalid block's timestamp has been verified
		resp, err := snClient.SuperRootAtTimestamp(ctx, invalidBlockTimestamp)
		t.Require().NoError(err, "SuperRootAtTimestamp should not error")

		t.Logger().Info("observation tick",
			"elapsed", time.Since(start).Round(time.Second),
			"chainA_local_safe", statusA.LocalSafeL2.Number,
			"chainA_unsafe", statusA.UnsafeL2.Number,
			"chainB_local_safe", statusB.LocalSafeL2.Number,
			"chainB_unsafe", statusB.UnsafeL2.Number,
			"invalid_block_ts", invalidBlockTimestamp,
			"replacement_detected", replacementDetected,
			"verified", resp.Data != nil,
		)

		// If replacement was detected and timestamp is now verified, we're done
		if replacementDetected && resp.Data != nil {
			t.Logger().Info("SUCCESS: replacement block is now verified",
				"timestamp", invalidBlockTimestamp,
				"replacement_hash", replacementBlockHash,
			)
			break
		}
	}

	// Final assertions

	// ASSERTION: The invalid block should have been replaced
	t.Require().True(replacementDetected,
		"invalid block should have been replaced with a different block")

	// ASSERTION: The replacement block should be different from the invalid block
	t.Require().NotEqual(invalidBlockHash, replacementBlockHash,
		"replacement block hash should differ from invalid block hash")

	// ASSERTION: The timestamp should eventually be verified (with the replacement block)
	finalResp, err := snClient.SuperRootAtTimestamp(ctx, invalidBlockTimestamp)
	t.Require().NoError(err)
	t.Require().NotNil(finalResp.Data,
		"timestamp should be verified after block replacement")

	finalStatusA := sys.L2ACL.SyncStatus()
	finalStatusB := sys.L2BCL.SyncStatus()

	t.Logger().Info("test complete: invalid block was replaced and validity advanced",
		"final_chainA_local_safe", finalStatusA.LocalSafeL2.Number,
		"final_chainA_unsafe", finalStatusA.UnsafeL2.Number,
		"final_chainB_local_safe", finalStatusB.LocalSafeL2.Number,
		"final_chainB_unsafe", finalStatusB.UnsafeL2.Number,
		"invalid_block_hash", invalidBlockHash,
		"replacement_block_hash", replacementBlockHash,
		"invalid_block_timestamp", invalidBlockTimestamp,
	)
}

// sendInvalidExecMessageForReplacement sends an executing message with a modified (invalid) identifier.
// This makes the message invalid because it references a non-existent log index.
func sendInvalidExecMessageForReplacement(
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
