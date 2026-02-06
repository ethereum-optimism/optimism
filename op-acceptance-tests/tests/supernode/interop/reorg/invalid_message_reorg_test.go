package reorg

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

	// Observe for reset behavior:
	// When the interop activity detects the invalid message and calls InvalidateBlock,
	// it will trigger a rewind. We observe by watching for the unsafe head to go backwards
	// or for the block at the invalid block number to change.

	observationDuration := 60 * time.Second
	checkInterval := time.Second

	start := time.Now()
	var resetDetected bool

	for time.Since(start) < observationDuration {
		time.Sleep(checkInterval)

		// Check if the block hash at the invalid block number changed or block doesn't exist
		// Use the EthClient directly to handle errors (block may not exist after rewind)
		currentBlock, err := sys.L2ELB.Escape().EthClient().BlockRefByNumber(ctx, invalidBlockNumber)
		if err != nil {
			// Block not found - this means the rewind happened and block was removed
			resetDetected = true
			t.Logger().Info("RESET DETECTED! Block no longer exists (rewound)",
				"block_number", invalidBlockNumber,
				"err", err,
			)
		} else if currentBlock.Hash != invalidBlockHash {
			// Block exists but with different hash - replaced
			resetDetected = true
			t.Logger().Info("RESET DETECTED! Block hash changed",
				"block_number", invalidBlockNumber,
				"old_hash", invalidBlockHash,
				"new_hash", currentBlock.Hash,
			)
		}

		// Check verification status
		resp, err := snClient.SuperRootAtTimestamp(ctx, invalidBlockTimestamp)
		if err != nil {
			t.Logger().Info("SuperRootAtTimestamp error (may be resetting)",
				"elapsed", time.Since(start).Round(time.Second),
				"err", err,
			)
			continue
		}

		var currentHash string
		if currentBlock.Hash != ([32]byte{}) {
			currentHash = currentBlock.Hash.String()[:10]
		} else {
			currentHash = "(none)"
		}

		t.Logger().Info("observation tick",
			"elapsed", time.Since(start).Round(time.Second),
			"invalid_block_ts", invalidBlockTimestamp,
			"current_block_hash", currentHash,
			"reset_detected", resetDetected,
			"verified", resp.Data != nil,
		)

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

	t.Logger().Info("reset confirmed, now waiting for replacement block",
		"invalid_block_number", invalidBlockNumber,
		"invalid_block_hash", invalidBlockHash,
	)

	// PHASE 2: Wait for a replacement block to appear at the same height
	// After rewind, the derivation pipeline should rebuild the block with deposits-only
	var replacementBlockHash eth.BlockID
	var replacementDetected bool

	replacementTimeout := 60 * time.Second
	replacementStart := time.Now()

	for time.Since(replacementStart) < replacementTimeout {
		time.Sleep(checkInterval)

		// Try to get the block at the invalid block number
		currentBlock, err := sys.L2ELB.Escape().EthClient().BlockRefByNumber(ctx, invalidBlockNumber)
		if err != nil {
			t.Logger().Debug("waiting for replacement block",
				"elapsed", time.Since(replacementStart).Round(time.Second),
				"err", err,
			)
			continue
		}

		// Check if we got a different block than the invalid one
		if currentBlock.Hash != invalidBlockHash {
			replacementBlockHash = currentBlock.ID()
			replacementDetected = true
			t.Logger().Info("REPLACEMENT DETECTED! New block at same height",
				"block_number", invalidBlockNumber,
				"old_hash", invalidBlockHash,
				"new_hash", currentBlock.Hash,
			)
			break
		}

		t.Logger().Debug("block exists but still has invalid hash (waiting)",
			"elapsed", time.Since(replacementStart).Round(time.Second),
			"hash", currentBlock.Hash,
		)
	}

	// ASSERTION: Replacement block should have been created
	t.Require().True(replacementDetected,
		"replacement block should be created at the same height after invalidation")
	t.Require().NotEqual(invalidBlockHash, replacementBlockHash.Hash,
		"replacement block should have different hash than invalid block")

	t.Logger().Info("replacement block confirmed, verifying it differs from original",
		"replacement_hash", replacementBlockHash.Hash,
	)

	// ASSERTION: The replacement block is different than the original
	// Fetch the replacement block with its transactions
	replacementBlockInfo, replacementTxs, err := sys.L2ELB.Escape().EthClient().InfoAndTxsByNumber(ctx, invalidBlockNumber)
	t.Require().NoError(err, "failed to fetch replacement block")

	t.Require().NotEqual(invalidBlockHash, replacementBlockInfo.Hash(),
		"replacement block hash must differ from invalid block hash")
	t.Logger().Info("confirmed replacement block differs from original",
		"original_hash", invalidBlockHash,
		"replacement_hash", replacementBlockInfo.Hash(),
	)

	// ASSERTION: The invalid transaction no longer exists in the chain
	// The invalid exec message transaction should NOT be in the replacement block
	invalidTxHash := invalidExecReceipt.TxHash
	txInReplacementBlock := false
	for _, tx := range replacementTxs {
		if tx.Hash() == invalidTxHash {
			txInReplacementBlock = true
			break
		}
	}
	t.Require().False(txInReplacementBlock,
		"invalid transaction should NOT exist in replacement block")

	// Also verify the transaction receipt is no longer available at that block
	// (the tx may have been re-included in a later block, but not at the same height)
	t.Logger().Info("confirmed invalid transaction not in replacement block",
		"invalid_tx_hash", invalidTxHash,
		"replacement_block_tx_count", len(replacementTxs),
	)

	t.Logger().Info("replacement block validated, waiting for verification",
		"replacement_hash", replacementBlockHash.Hash,
	)

	// PHASE 3: Wait for the replacement block's timestamp to become verified
	var verified bool
	verificationTimeout := 60 * time.Second
	verificationStart := time.Now()

	for time.Since(verificationStart) < verificationTimeout {
		time.Sleep(checkInterval)

		resp, err := snClient.SuperRootAtTimestamp(ctx, invalidBlockTimestamp)
		if err != nil {
			t.Logger().Debug("waiting for verification",
				"elapsed", time.Since(verificationStart).Round(time.Second),
				"err", err,
			)
			continue
		}

		if resp.Data != nil {
			verified = true
			t.Logger().Info("VERIFIED! Timestamp now verified with replacement block",
				"timestamp", invalidBlockTimestamp,
				"super_root", resp.Data.SuperRoot,
			)
			break
		}

		t.Logger().Debug("timestamp not yet verified",
			"elapsed", time.Since(verificationStart).Round(time.Second),
		)
	}

	// ASSERTION: The replacement block's timestamp should eventually be verified
	t.Require().True(verified,
		"replacement block timestamp should become verified")

	t.Logger().Info("test complete: invalid block was replaced and verified",
		"invalid_block_number", invalidBlockNumber,
		"invalid_block_hash", invalidBlockHash,
		"replacement_block_hash", replacementBlockHash.Hash,
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
