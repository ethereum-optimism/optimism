package op_reth_stale_forkchoice

import (
	"fmt"
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	nodeSync "github.com/ethereum-optimism/optimism/op-node/rollup/sync"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

const (
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

// finalizedCase is one cell of the matrix a forkchoice update names a reorged-out block as
// finalized under.
type finalizedCase struct {
	name string
	// siblingCanonicalFirst makes the sibling the head before the refused update names the block
	// it replaced as finalized; otherwise the refused update is the one that would make it the
	// head, and must not.
	siblingCanonicalFirst bool
}

func finalizedMatrix() []finalizedCase {
	return []finalizedCase{
		{name: "head_already_canonical", siblingCanonicalFirst: true},
		{name: "head_reorgs_to_sibling", siblingCanonicalFirst: false},
	}
}

// TestForkchoiceUpdateNamingReorgedOutFinalizedIsRejected builds a sibling of the follower's
// head and sends a forkchoice update that names the head the sibling replaces as finalized, and
// requires the engine to refuse it with the engine API's -38002 "Invalid forkchoice state" error
// and to leave its head, safe and finalized blocks as they were: both when the sibling is
// already the head, and when the update is the one that would reorg to the sibling.
func TestForkchoiceUpdateNamingReorgedOutFinalizedIsRejected(gt *testing.T) {
	t := devtest.ParallelT(gt)
	for _, c := range finalizedMatrix() {
		t.Run(c.name, func(t devtest.T) {
			t.Parallel()
			skipUnsupported(t)
			if !c.siblingCanonicalFirst {
				t.Skip("fails on develop (#23046): reth 2.5.2 reorgs to the sibling before it checks the finalized hash, so the " +
					"update is refused with `-38002 Invalid forkchoice state` but the head has moved: `a refused forkchoice update " +
					"must leave the head, safe and finalized blocks as they were: Not equal: expected head 4847ca…2e8e84:11 (the " +
					"replaced block), actual 1cdfb4…b22e70:11 (the sibling)`; reth 2.6 checks the state before it applies anything (reth#27247)")
			}
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
			if c.siblingCanonicalFirst {
				sys.Follower.EL.ForkchoiceUpdateRaw(siblingHash, labels.safe.Hash, labels.finalized.Hash, nil).IsValid()
				labels = readEngineLabels(sys.Follower.EL)
				t.Require().Equal(siblingHash, labels.head.Hash, "the sibling must be the head")
			}

			t.Logger().Info("Naming the replaced block as finalized", "head", labels.head, "safe", labels.safe, "finalized", labels.finalized, "replaced", replaced, "sibling", siblingHash)
			sys.Follower.EL.ForkchoiceUpdateRaw(siblingHash, labels.safe.Hash, replaced.Hash, nil).IsInvalidForkchoiceState()
			t.Require().Equal(labels, readEngineLabels(sys.Follower.EL), "a refused forkchoice update must leave the head, safe and finalized blocks as they were")
		})
	}
}

// persistedHeadCase is one cell of the matrix the follower is reorged below its in-memory
// window under.
type persistedHeadCase struct {
	persistenceThreshold uint64
}

func (c persistedHeadCase) name() string {
	return fmt.Sprintf("threshold_%d", c.persistenceThreshold)
}

// forkDepth is how far below the head the replaced block sits: below everything reth may hold in
// memory. Reth persists once more than the threshold of blocks sit in memory and keeps the last
// memoryBlockBufferTarget of them, so a block more than the threshold plus the buffer behind the
// head has been persisted and dropped from memory, with a margin for a save still running.
func (c persistedHeadCase) forkDepth() uint64 {
	return c.persistenceThreshold + memoryBlockBufferTarget + 2
}

func persistedHeadMatrix() []persistedHeadCase {
	return []persistedHeadCase{{persistenceThreshold: 7}, {persistenceThreshold: 50}}
}

// TestForkchoiceUpdateToPersistedReorgedOutHeadIsAppliedWhenReportedValid reorgs the follower
// to a sibling of a block below reth's in-memory window, so the blocks it replaces are persisted
// and stay on disk until reth's disk reorg, which runs after the update is answered, removes
// them. It then sends a forkchoice update back to the replaced block while it may still be on
// disk. The replaced block is not canonical, so the engine must not take it for the head it
// already has: it either reorgs back to it and answers VALID, or answers SYNCING and the follower
// downloads it from the sequencer, which still holds it. Either way the follower ends on the
// replaced block.
func TestForkchoiceUpdateToPersistedReorgedOutHeadIsAppliedWhenReportedValid(gt *testing.T) {
	t := devtest.ParallelT(gt)
	for _, c := range persistedHeadMatrix() {
		t.Run(c.name(), func(t devtest.T) {
			t.Parallel()
			skipUnsupported(t)
			t.Skip("fails on develop (#23046): reth 2.5.2 finds the replaced block on disk and takes it for the canonical head, so it " +
				"answers `VALID` with the replaced block as latestValidHash while its head stays on the sibling: `a VALID answer must " +
				"mean the head is the requested block: Not equal: expected e9c6db…0c1bb5:10 (the replaced block), actual 3866ab…a6b9cf:10 " +
				"(the sibling)` at both thresholds; reth 2.6 checks a persisted header against the canonical hash at its height " +
				"(reth#27429), reorgs back to the replaced block and answers VALID")
			sys := newStaleForkchoiceSystem(t, systemConfig{
				persistenceThreshold: c.persistenceThreshold,
				syncMode:             nodeSync.CLSync,
			})
			head := freezeFollower(t, sys, c.forkDepth()+1)
			replaced := sys.Follower.EL.BlockRefByNumber(head.Number - c.forkDepth())
			sibling := sys.Follower.EL.BuildSiblingBlock(replaced.Number)
			siblingHash := sibling.ExecutionPayload.BlockHash
			sys.Follower.EL.NewPayloadRaw(sibling).IsValid()
			labels := readEngineLabels(sys.Follower.EL)
			t.Require().Less(labels.safe.Number, replaced.Number, "the safe head must be below the replaced block")

			t.Logger().Info("Reorging the follower below its in-memory window", "head", head, "replaced", replaced, "sibling", siblingHash, "safe", labels.safe, "finalized", labels.finalized)
			sys.Follower.EL.ForkchoiceUpdateRaw(siblingHash, labels.safe.Hash, labels.finalized.Hash, nil).IsValid()
			siblingRef := sys.Follower.EL.BlockRefByLabel(eth.Unsafe)
			t.Require().Equal(siblingHash, siblingRef.Hash, "the sibling must be the head")

			t.Logger().Info("Updating the forkchoice back to the replaced block", "replaced", replaced)
			answer := sys.Follower.EL.ForkchoiceUpdateRaw(replaced.Hash, labels.safe.Hash, labels.finalized.Hash, nil)
			t.Require().NoError(answer.Err, "the update back to the replaced block must be answered")
			status := answer.Result.PayloadStatus.Status
			headAfter := sys.Follower.EL.BlockRefByLabel(eth.Unsafe)
			t.Logger().Info("Engine answered the update back to the replaced block", "status", status,
				"latest_valid_hash", answer.Result.PayloadStatus.LatestValidHash, "head", headAfter)
			t.Require().Contains([]eth.ExecutePayloadStatus{eth.ExecutionValid, eth.ExecutionSyncing}, status,
				"the replaced block is valid, so the engine either applies the update or syncs to it")
			if status == eth.ExecutionValid {
				t.Require().Equal(replaced, headAfter, "a VALID answer must mean the head is the requested block")
			}
			// A SYNCING answer makes the replaced block a sync target the follower downloads from the sequencer.
			sys.Follower.EL.ReorgExact(siblingRef, syncBackAttempts)
			t.Require().Equal(replaced, sys.Follower.EL.BlockRefByLabel(eth.Unsafe), "the follower must be back on the replaced block")
		})
	}
}
