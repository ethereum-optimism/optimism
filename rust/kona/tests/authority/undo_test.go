package authority_test

import (
	"slices"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-e2e/actions/helpers"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// TestUndoDerivesTheValidBlockOfTheNewL1Fork checks, on a deterministic L1, the undo of a denial
// whose L1 block a reorg removes, where the new L1 fork makes a different, valid block at the
// denied height.
//
// On the first fork the chain's block at height H executes a message that does not exist, so
// lokahi denies it, decided on the L1 block that carries its batch, and the kona-node replaces it
// with its deposits-only block, which lokahi verifies in turn. A reorg then removes that L1 block,
// and the new fork carries another producer's batch for height H, whose block is valid. lokahi
// deletes the denial with a new epoch; the kona-node takes that epoch on by deriving again from
// its finalized head, ends on the valid block rather than the replacement, and lokahi verifies it
// on the new fork.
func TestUndoDerivesTheValidBlockOfTheNewL1Fork(gt *testing.T) {
	t := helpers.NewDefaultTesting(gt)
	sys := newSystem(t, 1)
	a := sys.chains[0]
	first, second := a.producer, a.newProducer("second-producer")

	// The common prefix: the first producer's blocks, batched on L1 block 2 and final on L1. Both
	// producers and the kona-node derive it.
	sys.miner.ActEmptyBlock(t)
	first.buildToL1Head()
	sys.mine(first.batch())
	prefix := first.derive()
	require.Equal(t, prefix, second.derive(), "both producers derive the common prefix")
	a.postUnsafe(first.payload(1))
	a.awaitLocalSafe(prefix.Number)
	sys.miner.ActL1Safe(t, 2)
	sys.miner.ActL1Finalize(t, 2)
	require.Eventually(t, func() bool { return a.syncStatus().FinalizedL2.Number >= prefix.Number },
		time.Minute, 500*time.Millisecond, "the kona-node must finalize the common prefix")

	// Each producer builds its own block at height H on the common prefix: the first one executes
	// a message that does not exist, the second one transfers ether.
	execTx := first.executingMessageTx(a.missingMessage())
	denied := first.build(execTx)
	require.Len(t, first.receipt(execTx).Logs, 1, "the executing message tx must emit its ExecutingMessage log")
	valid := second.build(second.transferTx())
	require.Equal(t, prefix.Number+1, denied.Number)
	require.Equal(t, denied.Number, valid.Number)
	require.NotEqual(t, denied.Hash, valid.Hash)
	height := denied.Number

	// The first fork: lokahi denies the block, decided on the L1 block carrying its batch, the
	// kona-node replaces it, and lokahi verifies the replacement.
	oldFork := sys.mine(first.batch())
	epochBefore := a.lokahiStatus().Epoch
	var replacement eth.BlockID
	require.Eventually(t, func() bool {
		status := a.lokahiStatus()
		if !slices.Contains(status.Denied, denied.ID()) || status.Verified == nil ||
			status.Verified.Block.Number < height {
			return false
		}
		replacement = status.Verified.Block
		return true
	}, 2*time.Minute, 500*time.Millisecond, "lokahi must deny block %s and verify its replacement", denied.ID())
	require.NotEqual(t, denied.ID(), replacement, "the denied block is replaced")
	require.NotEqual(t, valid.ID(), replacement)
	require.Equal(t, replacement, a.nodeBlockAt(height), "the kona-node holds the replacement")
	// So the undo that takes the next epoch on moves the local-safe head and the safe label back
	// from the replacement to the finalized head.
	before := a.syncStatus()
	require.Equal(t, prefix.ID(), before.FinalizedL2.ID(), "the common prefix is final")
	require.Equal(t, replacement, before.LocalSafeL2.ID(), "the replacement is local-safe")
	require.Equal(t, epochBefore, a.lokahiStatus().Epoch, "no denial was deleted yet")
	t.Logf("denied %s on L1 %s, replaced by %s", denied.ID(), eth.ToBlockID(oldFork), replacement)

	// The reorg: the L1 block the denial was decided on goes, and the new fork carries the second
	// producer's batch for height H.
	sys.miner.ActL1RewindToParent(t)
	newFork := sys.mine(second.batch())
	sys.mine()
	t.Logf("reorged L1 block %s out for %s", eth.ToBlockID(oldFork), eth.ToBlockID(newFork))

	var epochAfter common.Hash
	require.Eventually(t, func() bool {
		epochAfter = a.lokahiStatus().Epoch
		return epochAfter != epochBefore
	}, 2*time.Minute, 500*time.Millisecond, "lokahi must delete the denial with a new epoch")
	require.Eventually(t, func() bool {
		applied := a.appliedEpoch()
		return applied != nil && *applied == epochAfter
	}, 2*time.Minute, 500*time.Millisecond, "the kona-node must take the new epoch on")
	require.Eventually(t, func() bool {
		status := a.lokahiStatus()
		return status.Verified != nil && status.Verified.Block == valid.ID()
	}, 2*time.Minute, 500*time.Millisecond, "lokahi must verify the valid block %s", valid.ID())

	status := a.lokahiStatus()
	require.Equal(t, eth.ToBlockID(newFork), status.Verified.L1, "the verified head holds on the new fork")
	require.Empty(t, status.Denied, "lokahi denies nothing on the new fork")
	require.Equal(t, valid.ID(), a.nodeBlockAt(height), "the kona-node ends on the valid block")
	require.Eventually(t, func() bool {
		safe := a.syncStatus().SafeL2
		return safe.Number >= height && a.nodeBlockAt(height) == valid.ID()
	}, time.Minute, 500*time.Millisecond, "the kona-node's safe head must pass the valid block")
}
