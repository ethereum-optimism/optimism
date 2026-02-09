package reorg

import (
	"context"
	"math/rand"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/devnet-sdk/contracts/constants"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
	"github.com/ethereum-optimism/optimism/op-service/txintent"
)

// TestSupernodeInteropInvalidMessageReplacement tests that:
// WHEN: an invalid Executing Message is included in a chain
// THEN:
// - The interop activity detects the invalid block
// - The chain container is told to invalidate the block
// - A reset/rewind is triggered if the chain is using that block
// - A replacement block is built at the same height (deposits-only)
// - The replacement block's timestamp eventually becomes verified
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
	// set supernode to pause verification just after this timestamp
	sys.Supernode.PauseInterop(targetTimestamp + 1)
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

	// Observe the invalid block is locally safe on Chain B
	require.Eventually(t, func() bool {
		numSame := sys.L2BCL.SyncStatus().LocalSafeL2.Number == invalidBlockNumber
		hashSame := sys.L2BCL.SyncStatus().LocalSafeL2.Hash == invalidBlockHash
		return numSame && hashSame
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
			// Block not found - this means the rewind happened and block was removed
			t.Logger().Info("RESET DETECTED! Block no longer exists (rewound)",
				"block_number", invalidBlockNumber,
				"err", err,
			)
		} else if currentBlock.Hash != invalidBlockHash {
			// Block exists but with different hash - replaced
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
	require.Eventually(t, func() bool {
		resp, err := snClient.SuperRootAtTimestamp(ctx, invalidBlockTimestamp)
		if err != nil {
			return false
		}
		return resp.Data != nil
	}, 60*time.Second, time.Second, "replacement should be verified")

	// ASSERTION: The invalid transaction no longer exists in the chain
	// The invalid exec message transaction should NOT be in the replacement block
	replacementBlockInfo, replacementTxs, err := sys.L2ELB.Escape().EthClient().InfoAndTxsByNumber(ctx, invalidBlockNumber)
	t.Require().NoError(err, "failed to fetch replacement block")
	invalidTxHash := invalidExecReceipt.TxHash
	for _, tx := range replacementTxs {
		if tx.Hash() == invalidTxHash {
			t.Logger().Error("invalid transaction should NOT exist in replacement block",
				"invalid_tx_hash", invalidTxHash,
				"replacement_tx_hash", tx.Hash(),
			)
			t.FailNow()
		}
	}

	t.Logger().Info("test complete: invalid block was replaced and verified",
		"invalid_block_number", invalidBlockNumber,
		"invalid_block_hash", invalidBlockHash,
		"replacement_block_hash", replacementBlockInfo.Hash,
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
