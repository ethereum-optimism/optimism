package op_reth_stale_forkchoice

import (
	"fmt"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	nodeSync "github.com/ethereum-optimism/optimism/op-node/rollup/sync"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
)

const (
	// replacedPersistedTimeout bounds how long the frozen follower may take to report the replaced
	// block persisted: it already holds every block it needs, so reth only finishes the saves that
	// are due.
	replacedPersistedTimeout = 30 * time.Second
	// syncBackAttempts bounds, in 2s polls, how long the follower may take to download the head
	// of a forkchoice update it answered SYNCING from the sequencer and to make it canonical.
	syncBackAttempts = 30
)

// freezeFollower stops the batcher, so nothing above the sequencer's head at that moment can
// become safe, lets the follower get blocksAboveSafe blocks past that head, and stops its
// consensus client, so the follower's engine answers only the forkchoice updates the test sends.
// It returns the follower's frozen head.
func freezeFollower(t devtest.T, sys *staleForkchoiceSystem, blocksAboveSafe uint64) eth.L2BlockRef {
	sys.Batcher.Stop()
	safeBound := sys.Sequencer.EL.BlockRefByLabel(eth.Unsafe)
	sys.Follower.EL.Reached(eth.Unsafe, safeBound.Number+blocksAboveSafe, chainBuildUpAttempts)
	sys.Follower.CL.Stop()
	head := sys.Follower.EL.BlockRefByLabel(eth.Unsafe)
	t.Logger().Info("Follower frozen", "head", head, "safe", sys.Follower.EL.BlockRefByLabel(eth.Safe), "safe_bound", safeBound)
	return head
}

// TestForkchoiceUpdateNamingReorgedOutFinalizedIsRejected builds a sibling of the follower's
// head and sends a forkchoice update that names the head the sibling replaces as finalized, and
// requires the engine to refuse it with the engine API's -38002 "Invalid forkchoice state" error
// and to leave its head, safe and finalized blocks as they were: first while the update is the one
// that would reorg to the sibling, then again once the sibling is the head.
func TestForkchoiceUpdateNamingReorgedOutFinalizedIsRejected(gt *testing.T) {
	t := devtest.ParallelT(gt)
	skipUnsupported(t)
	sys := newStaleForkchoiceSystem(t, systemConfig{
		persistenceThreshold: defaultPersistenceThreshold,
		syncMode:             nodeSync.CLSync,
	})
	replaced := freezeFollower(t, sys, 1)
	sibling := sys.Follower.EL.BuildSiblingBlock(replaced.Number)
	siblingHash := sibling.ExecutionPayload.BlockHash
	sys.Follower.EL.NewPayloadRaw(sibling).IsValid()
	labels := readEngineLabels(sys.Follower.EL)
	t.Require().Less(labels.safe.Number, replaced.Number, "the safe head must be below the replaced block")
	requireReplacedRefusedAsFinalized(t, sys.Follower.EL, siblingHash, replaced, labels)

	sys.Follower.EL.ForkchoiceUpdateRaw(siblingHash, labels.safe.Hash, labels.finalized.Hash, nil).IsValid()
	labels = readEngineLabels(sys.Follower.EL)
	t.Require().Equal(siblingHash, labels.head.Hash, "the sibling must be the head")
	requireReplacedRefusedAsFinalized(t, sys.Follower.EL, siblingHash, replaced, labels)
}

// requireReplacedRefusedAsFinalized sends a forkchoice update to the sibling that names the block
// it replaced as finalized, and requires the engine to refuse it and to keep labels.
func requireReplacedRefusedAsFinalized(t devtest.T, el *dsl.L2ELNode, siblingHash common.Hash, replaced eth.L2BlockRef, labels engineLabels) {
	t.Logger().Info("Naming the replaced block as finalized", "head", labels.head, "safe", labels.safe, "finalized", labels.finalized, "replaced", replaced, "sibling", siblingHash)
	el.ForkchoiceUpdateRaw(siblingHash, labels.safe.Hash, replaced.Hash, nil).IsInvalidForkchoiceState()
	t.Require().Equal(labels, readEngineLabels(el), "a refused forkchoice update must leave the head, safe and finalized blocks as they were")
}

// persistedHeadCase is one cell of the matrix the follower is reorged below its in-memory
// window under.
type persistedHeadCase struct {
	persistenceThreshold uint64
}

func (c persistedHeadCase) String() string {
	return fmt.Sprintf("threshold_%d", c.persistenceThreshold)
}

// forkDepth is how far below the frozen head the replaced block sits. Reth persists once more
// than the threshold of blocks sit in memory and keeps the newest of them up to the memory block
// buffer target, so a block this deep is due for persistence without further blocks arriving.
func (c persistedHeadCase) forkDepth() uint64 {
	return c.persistenceThreshold + sysgo.OpRethPinnedMemoryBlockBufferTarget + 2
}

func persistedHeadMatrix() []persistedHeadCase {
	return []persistedHeadCase{{persistenceThreshold: 7}, {persistenceThreshold: 50}}
}

// TestForkchoiceUpdateToPersistedReorgedOutHeadIsAppliedOrSynced waits until the follower
// reports a block persisted, reorgs it to a sibling of that block, and sends a forkchoice update
// back to the replaced block as soon as the reorg is answered. The replaced block is not
// canonical, so the engine must not take it for the head it already has: it either reorgs back to
// it and answers VALID, or answers SYNCING and the follower downloads it from the sequencer, which
// still holds it. Either way the follower ends on the replaced block.
func TestForkchoiceUpdateToPersistedReorgedOutHeadIsAppliedOrSynced(gt *testing.T) {
	t := devtest.ParallelT(gt)
	devtest.RunParallel(t, persistedHeadMatrix(), func(t devtest.T, c persistedHeadCase) {
		skipUnsupported(t)
		sys := newStaleForkchoiceSystem(t, systemConfig{
			persistenceThreshold: c.persistenceThreshold,
			syncMode:             nodeSync.CLSync,
		})
		persistedBlocks := sys.Follower.EL.TrackPersistedBlock()
		head := freezeFollower(t, sys, c.forkDepth()+1)
		replaced := sys.Follower.EL.BlockRefByNumber(head.Number - c.forkDepth())
		persistedBlocks.AwaitPersisted(replaced.Number, replacedPersistedTimeout)
		sibling := sys.Follower.EL.BuildSiblingBlock(replaced.Number)
		siblingHash := sibling.ExecutionPayload.BlockHash
		sys.Follower.EL.NewPayloadRaw(sibling).IsValid()
		labels := readEngineLabels(sys.Follower.EL)
		t.Require().Less(labels.safe.Number, replaced.Number, "the safe head must be below the replaced block")

		t.Logger().Info("Reorging the follower below its persisted block", "head", head, "replaced", replaced, "sibling", siblingHash, "safe", labels.safe, "finalized", labels.finalized)
		reorged := sys.Follower.EL.ForkchoiceUpdateRaw(siblingHash, labels.safe.Hash, labels.finalized.Hash, nil).IsValid()
		t.Require().NotNil(reorged.Result.PayloadStatus.LatestValidHash, "a VALID answer to the reorg must name its latest valid block")
		t.Require().Equal(siblingHash, *reorged.Result.PayloadStatus.LatestValidHash, "the engine must have made the sibling the head")
		// Reth removes the replaced blocks from disk on its persistence thread after answering the
		// reorg and offers no way to hold that off. Only a VALID answer, which needs the update back
		// to arrive before that removal commits, checks that a persisted header counts as canonical
		// only when it matches the canonical hash (reth#27429); after the removal reth answers
		// SYNCING, which this test accepts, so a run can pass without exercising that check. No RPC
		// runs between the two updates, to keep the VALID window as wide as it can be.
		answer := sys.Follower.EL.ForkchoiceUpdateRaw(replaced.Hash, labels.safe.Hash, labels.finalized.Hash, nil)
		t.Require().NoError(answer.Err, "the update back to the replaced block must be answered")
		status := answer.Result.PayloadStatus.Status
		headAfter := sys.Follower.EL.BlockRefByLabel(eth.Unsafe)
		t.Logger().Info("Engine answered the update back to the replaced block", "status", status,
			"latest_valid_hash", answer.Result.PayloadStatus.LatestValidHash, "head", headAfter)
		t.Require().Contains([]eth.ExecutePayloadStatus{eth.ExecutionValid, eth.ExecutionSyncing}, status,
			"the replaced block is valid, so the engine either applies the update or syncs to it")
		if status == eth.ExecutionValid {
			t.Logger().Info("Engine reorged back to the persisted block itself, which exercises reth#27429")
			t.Require().Equal(replaced, headAfter, "a VALID answer must mean the head is the requested block")
		} else {
			t.Logger().Info("Engine answered SYNCING, so this run does not exercise reth#27429; the follower downloads the replaced block from the sequencer")
		}
		siblingRef, err := derive.PayloadToBlockRef(sys.L2Network.Escape().RollupConfig(), sibling.ExecutionPayload)
		t.Require().NoError(err, "the sibling must carry an L1 info deposit")
		// A SYNCING answer makes the replaced block a sync target the follower downloads from the sequencer.
		sys.Follower.EL.ReorgExact(siblingRef, syncBackAttempts)
		t.Require().Equal(replaced, sys.Follower.EL.BlockRefByLabel(eth.Unsafe), "the follower must be back on the replaced block")
	})
}
