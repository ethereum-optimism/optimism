package reorg

import (
	"errors"
	"math/rand"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/eth"
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
	initTx, initReceipt := alice.SendRandomInitMessage(rng, eventLoggerA, 2, 10)

	t.Logger().Info("initiating message sent on chain A",
		"block", initReceipt.BlockNumber,
		"hash", initReceipt.BlockHash,
	)

	// Wait for chain B to catch up
	sys.L2B.WaitForBlock()

	// Send an INVALID executing message on chain B
	_, invalidExecReceipt := bob.SendInvalidExecMessage(initTx, 0)
	invalidBlockNumber := bigs.Uint64Strict(invalidExecReceipt.BlockNumber)
	invalidBlockHash := invalidExecReceipt.BlockHash
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
	sys.L2ELB.AssertTxNotInBlock(invalidBlockNumber, invalidExecReceipt.TxHash)

	t.Logger().Info("test complete: invalid block was replaced and verified",
		"invalid_block_number", invalidBlockNumber,
		"invalid_block_hash", invalidBlockHash,
	)
}
