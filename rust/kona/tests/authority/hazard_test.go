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

// TestUndoRestoresABlockWhoseDenialAReorgVoided checks, on a deterministic L1, the undo of a denial
// that only lokahi can see is void: the denied block's own batch stays on the canonical L1 chain,
// so nothing of its chain's derivation changes with the reorg.
//
// Chain B's block at height H executes a message that chain A's block at the same timestamp is
// to initiate, and B's batch lands on L1 block N. On the first fork, A's batch on L1 block N+1
// carries a block at H without that message, so lokahi denies B's block, decided on N+1, and B's
// node replaces it. A reorg then removes only N+1, and the new fork carries another producer's
// batch for A whose block at H does initiate the message. lokahi deletes the denial with a new
// epoch, B's node takes it on by deriving again from its finalized head, derives its original
// block at H again from the unchanged batch on N, and lokahi verifies it.
//
// B's node sees L1 only up to N until the reorg is done, as a node whose L1 view lags behind, so
// its derivation never reaches the N+1 the reorg removes and has no reorg of its own to handle.
// It is also cut off from lokahi until it has derived past the new N+1, and keeps the replacement
// all that time: nothing but the undo the new epoch brings restores its block.
func TestUndoRestoresABlockWhoseDenialAReorgVoided(gt *testing.T) {
	t := helpers.NewDefaultTesting(gt)
	sys := newSystem(t, 2)
	a, b := sys.chains[0], sys.chains[1]
	aFirst, aSecond := a.producer, a.newProducer("second-producer")

	// The common prefix on both chains, batched on L1 block 2 and final on L1.
	sys.miner.ActEmptyBlock(t)
	aFirst.buildToL1Head()
	b.producer.buildToL1Head()
	sys.mine(aFirst.batch(), b.producer.batch())
	prefix := aFirst.derive()
	require.Equal(t, prefix, aSecond.derive(), "both of A's producers derive the common prefix")
	require.Equal(t, prefix.Number, b.producer.derive().Number, "both chains derive to the same height")
	a.postUnsafe(aFirst.payload(1))
	b.postUnsafe(b.producer.payload(1))
	sys.miner.ActL1Safe(t, 2)
	sys.miner.ActL1Finalize(t, 2)
	for _, c := range sys.chains {
		require.Eventually(t, func() bool { return c.syncStatus().FinalizedL2.Number >= prefix.Number },
			time.Minute, 500*time.Millisecond, "chain %s must finalize the common prefix", c.name)
	}

	// At height H, A's second producer's block initiates a message, B's block executes it, and A's
	// first producer's block does not initiate it.
	initTx := aSecond.initiatingTx()
	initiating := aSecond.build(initTx)
	msg := a.messageOf(aSecond.receipt(initTx), initiating.Time)
	execTx := b.producer.executingMessageTx(msg)
	executing := b.producer.build(execTx)
	require.Len(t, b.producer.receipt(execTx).Logs, 1, "the executing message tx must emit its ExecutingMessage log")
	without := aFirst.build(aFirst.transferTx())
	height := executing.Number
	for _, block := range []eth.L2BlockRef{initiating, without} {
		require.Equal(t, height, block.Number)
		require.Equal(t, executing.Time, block.Time)
	}

	// The first fork: B's batch on L1 block N, A's batch without the message on N+1. lokahi
	// denies B's block, decided on N+1, and B's node replaces it.
	n := sys.mine(b.producer.batch())
	b.awaitLocalSafe(height)
	// B's node sees L1 only up to N until the reorg is done, as a node whose L1 view lags behind,
	// so its derivation never reaches N+1 and nothing of its own reacts to the reorg.
	b.l1.freeze(n.NumberU64())
	nPlus1 := sys.mine(aFirst.batch())
	epochBefore := b.lokahiStatus().Epoch
	var replacement eth.BlockID
	require.Eventually(t, func() bool {
		status := b.lokahiStatus()
		if !slices.Contains(status.Denied, executing.ID()) || status.Verified == nil ||
			status.Verified.Block.Number < height {
			return false
		}
		replacement = status.Verified.Block
		return true
	}, 2*time.Minute, 500*time.Millisecond, "lokahi must deny B's block %s and verify its replacement", executing.ID())
	require.NotEqual(t, executing.ID(), replacement, "B's denied block is replaced")
	require.Equal(t, eth.ToBlockID(nPlus1), b.lokahiStatus().Verified.L1, "the replacement holds on N+1")
	require.Equal(t, replacement, b.nodeBlockAt(height), "B's node holds the replacement")
	before := b.syncStatus()
	require.Equal(t, prefix.Number, before.FinalizedL2.Number, "B's common prefix is final")
	require.Equal(t, replacement, before.LocalSafeL2.ID(), "B's replacement is local-safe")
	t.Logf("B's %s on L1 %s denied on L1 %s, replaced by %s", executing.ID(), eth.ToBlockID(n), eth.ToBlockID(nPlus1), replacement)

	// The reorg removes only N+1; the new fork carries A's other batch, whose block initiates the
	// message.
	logMark := b.logSize()
	b.authority.disconnect()
	sys.miner.ActL1RewindToParent(t)
	newNPlus1 := sys.mine(aSecond.batch())
	sys.mine()
	b.l1.unfreeze()
	t.Logf("reorged L1 block %s out for %s", eth.ToBlockID(nPlus1), eth.ToBlockID(newNPlus1))

	// Until B's node hears lokahi's new epoch nothing restores its block: its derivation sees L1
	// move on past N without a reorg of its own, and keeps the replacement.
	var epochAfter common.Hash
	require.Eventually(t, func() bool {
		epochAfter = b.lokahiStatus().Epoch
		return epochAfter != epochBefore
	}, 2*time.Minute, 500*time.Millisecond, "lokahi must delete the denial with a new epoch")
	require.Eventually(t, func() bool { return b.syncStatus().CurrentL1.Number >= newNPlus1.NumberU64() },
		time.Minute, 500*time.Millisecond, "B's node must derive past the new N+1")
	require.Never(t, func() bool { return b.nodeBlockAt(height) != replacement },
		10*time.Second, 500*time.Millisecond, "without the new epoch B's node must keep the replacement")
	b.authority.reconnect()
	require.Eventually(t, func() bool {
		applied := b.appliedEpoch()
		return applied != nil && *applied == epochAfter
	}, 2*time.Minute, 500*time.Millisecond, "B's node must take the new epoch on")
	require.Eventually(t, func() bool {
		aStatus, bStatus := a.lokahiStatus(), b.lokahiStatus()
		return aStatus.Verified != nil && aStatus.Verified.Block == initiating.ID() &&
			bStatus.Verified != nil && bStatus.Verified.Block == executing.ID()
	}, 2*time.Minute, 500*time.Millisecond, "lokahi must verify A's initiating and B's original executing block")

	status := b.lokahiStatus()
	require.Equal(t, eth.ToBlockID(newNPlus1), status.Verified.L1, "B's verified head holds on the new fork")
	require.Empty(t, status.Denied, "lokahi denies nothing on the new fork")
	require.Equal(t, executing.ID(), b.nodeBlockAt(height), "B's node derives its original block again")
	require.Eventually(t, func() bool {
		return b.syncStatus().SafeL2.Number >= height && b.nodeBlockAt(height) == executing.ID()
	}, time.Minute, 500*time.Millisecond, "B's safe head must pass its original block")

	// It was the undo that moved B's labels back, and B's derivation saw no reorg of its own: its
	// batch stayed on N.
	undo := b.logSince(logMark, "Moved the local-safe head and the safe label back to the finalized head")
	require.NotEmpty(t, undo, "B's node must move its labels back to take the new epoch on")
	resets := b.logSince(logMark, "Derivation pipeline is being reset", "Received reset request")
	require.Empty(t, resets, "B's derivation must not reset on the L1 reorg of its own")
}
