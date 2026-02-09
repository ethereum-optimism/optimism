package halt

import (
	"context"
	"math/rand"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/ethereum-optimism/optimism/devnet-sdk/contracts/constants"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
	"github.com/ethereum-optimism/optimism/op-service/txintent"
)

// TestSupernodeInteropInvalidMessageHalt tests that:
// WHEN: an invalid Executing Message is included in a chain
// THEN:
// - Validity Never Advances to include the Invalid Block
// - Local Safety and Unsafety for both chains continue to advance
//
// This is a TDD test that starts a cycle to implement the Interop Activity's actual algorithm.
func TestSupernodeInteropInvalidMessageHalt(gt *testing.T) {
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
	invalidExecReceipt := sendInvalidExecMessage(t, bob, initTx, 0)

	invalidBlockNumber := bigs.Uint64Strict(invalidExecReceipt.BlockNumber)
	invalidBlock := sys.L2ELB.BlockRefByHash(invalidExecReceipt.BlockHash)
	invalidBlockTimestamp := invalidBlock.Time

	t.Logger().Info("invalid executing message sent on chain B",
		"block", invalidExecReceipt.BlockNumber,
		"hash", invalidExecReceipt.BlockHash,
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

	// Now we verify the key behaviors over time:
	// 1. Validity should NEVER advance to include the invalid block
	// 2. Local Safety and Unsafety should continue to advance for both chains

	observationDuration := 30 * time.Second
	checkInterval := time.Second

	start := time.Now()
	var lastVerifiedTimestamp uint64

	for time.Since(start) < observationDuration {
		time.Sleep(checkInterval)

		// Check current safety status
		statusA := sys.L2ACL.SyncStatus()
		statusB := sys.L2BCL.SyncStatus()

		// KEY ASSERTION 1: Validity should NOT advance past the invalid block's timestamp
		// Check if the invalid block's timestamp has been verified (it should NOT be)
		resp, err := snClient.SuperRootAtTimestamp(ctx, invalidBlockTimestamp)
		t.Require().NoError(err, "SuperRootAtTimestamp should not error")

		if resp.Data != nil {
			t.Logger().Error("UNEXPECTED: invalid block timestamp was verified!",
				"timestamp", invalidBlockTimestamp,
				"invalid_block", invalidBlockNumber,
			)
			t.FailNow()
		}

		// Track the last verified timestamp (for timestamps before the invalid block)
		if invalidBlockTimestamp > blockTime {
			checkTs := invalidBlockTimestamp - blockTime
			checkResp, _ := snClient.SuperRootAtTimestamp(ctx, checkTs)
			if checkResp.Data != nil {
				lastVerifiedTimestamp = checkTs
			}
		}

		t.Logger().Info("observation tick",
			"elapsed", time.Since(start).Round(time.Second),
			"chainA_local_safe", statusA.LocalSafeL2.Number,
			"chainA_unsafe", statusA.UnsafeL2.Number,
			"chainB_local_safe", statusB.LocalSafeL2.Number,
			"chainB_unsafe", statusB.UnsafeL2.Number,
			"last_verified_ts", lastVerifiedTimestamp,
			"invalid_block_ts", invalidBlockTimestamp,
		)
	}

	// Final assertions after observation period

	finalStatusA := sys.L2ACL.SyncStatus()
	finalStatusB := sys.L2BCL.SyncStatus()

	// ASSERTION: Local Safety should have advanced for both chains
	t.Require().Greater(finalStatusA.LocalSafeL2.Number, initialStatusA.LocalSafeL2.Number,
		"chain A local safe head should advance")
	t.Require().Greater(finalStatusB.LocalSafeL2.Number, initialStatusB.LocalSafeL2.Number,
		"chain B local safe head should advance")

	// ASSERTION: Unsafety should have advanced for both chains
	t.Require().Greater(finalStatusA.UnsafeL2.Number, initialStatusA.UnsafeL2.Number,
		"chain A unsafe head should advance")
	t.Require().Greater(finalStatusB.UnsafeL2.Number, initialStatusB.UnsafeL2.Number,
		"chain B unsafe head should advance")

	// ASSERTION: The invalid block's timestamp should still NOT be verified
	finalResp, err := snClient.SuperRootAtTimestamp(ctx, invalidBlockTimestamp)
	t.Require().NoError(err)
	t.Require().Nil(finalResp.Data,
		"invalid block timestamp should NEVER be verified")

	t.Logger().Info("test complete: invalid message correctly halted validity advancement",
		"final_chainA_local_safe", finalStatusA.LocalSafeL2.Number,
		"final_chainA_unsafe", finalStatusA.UnsafeL2.Number,
		"final_chainB_local_safe", finalStatusB.LocalSafeL2.Number,
		"final_chainB_unsafe", finalStatusB.UnsafeL2.Number,
		"invalid_block_timestamp", invalidBlockTimestamp,
		"last_verified_timestamp", lastVerifiedTimestamp,
	)
}

// sendInvalidExecMessage sends an executing message with a modified (invalid) identifier.
// This makes the message invalid because it references a non-existent log index.
func sendInvalidExecMessage(
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

// randomInitTrigger creates a random init trigger for testing.
func randomInitTrigger(rng *rand.Rand, eventLoggerAddress common.Address, topicCount, dataLen int) *txintent.InitTrigger {
	if topicCount > 4 {
		topicCount = 4 // Max 4 topics in EVM logs
	}
	if topicCount < 1 {
		topicCount = 1
	}
	if dataLen < 1 {
		dataLen = 1
	}

	topics := make([][32]byte, topicCount)
	for i := range topics {
		copy(topics[i][:], testutils.RandomData(rng, 32))
	}

	return &txintent.InitTrigger{
		Emitter:    eventLoggerAddress,
		Topics:     topics,
		OpaqueData: testutils.RandomData(rng, dataLen),
	}
}
