package core

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/wait"
	"github.com/ethereum-optimism/optimism/op-service/logpipe"
	"github.com/ethereum-optimism/optimism/rust/op-reth/tests/proofs/utils"
	"github.com/ethereum/go-ethereum/common"
)

const (
	// historyPruneBefore is the block before which the follower's reth prunes its account and
	// storage history, once its persisted tip is historyPruneMinimumDistance past it. A fixed
	// block rather than a distance: reth's config refuses a history distance below its
	// unwind-safe minimum of 10064 blocks, but takes any block for `before`.
	historyPruneBefore = 30
	// historyPruneMinimumDistance replaces reth's unwind-safe minimum in the follower's config,
	// so the pruner acts on historyPruneBefore while the chain is short.
	historyPruneMinimumDistance = 5
	// historyPruneBlockInterval is how many persisted blocks reth waits between pruner runs.
	historyPruneBlockInterval = 5
	// historyPrunedProofsWindow covers the whole chain of the run, so the store serves the blocks
	// reth pruned its own history of.
	historyPrunedProofsWindow = 1_000
	// historyPruneTimeout bounds the wait for reth's pruner to pass the checked block after the
	// last persistence before the chain stopped.
	historyPruneTimeout = 60 * time.Second

	rethHistoryPruned = "pruned"
	rethHistoryServed = "served"
)

// rethHistoryPruneConfig writes the config file that makes the follower's reth prune its account
// and storage history before historyPruneBefore. The prune modes go through the file rather than
// --prune.* flags, since flags replace the file's prune section and with it the lowered minimum.
func rethHistoryPruneConfig(t devtest.T) string {
	path := filepath.Join(t.TempDir(), "reth.toml")
	content := fmt.Sprintf(`[prune]
block_interval = %d
minimum_pruning_distance = %d

[prune.segments]
account_history = { before = %d }
storage_history = { before = %d }
`, historyPruneBlockInterval, historyPruneMinimumDistance, historyPruneBefore, historyPruneBefore)
	t.Require().NoError(os.WriteFile(path, []byte(content), 0o644), "must write the reth prune config")
	return path
}

// historyPrunedCase is one cell of the matrix the history-pruned follower runs under.
type historyPrunedCase struct {
	persistenceThreshold uint64
}

func (c historyPrunedCase) name() string {
	return fmt.Sprintf("threshold_%d", c.persistenceThreshold)
}

// headWithPrunedHistory is a chain height by which reth's pruner has run past historyPruneBefore:
// the pruner sees reth's persisted tip, which trails the head by the blocks reth holds in memory,
// and runs once that tip is the minimum distance plus a block interval past the prune block.
func (c historyPrunedCase) headWithPrunedHistory() uint64 {
	return c.persistenceThreshold + memoryBlockBufferTarget + historyPruneBefore + historyPruneMinimumDistance + historyPruneBlockInterval + windowMargin
}

func (c historyPrunedCase) followerArgs(t devtest.T) []string {
	return append(engineArgs(c.persistenceThreshold),
		"--config="+rethHistoryPruneConfig(t),
		// reth's pruner reports what it does, and why it holds back, only at debug; the lines
		// reach the test through log records, not the test log.
		"--log.stdout.filter=info,pruner=debug,engine::persistence=debug",
		"--proofs-history.verification-interval=0",
	)
}

func historyPrunedMatrix() []historyPrunedCase {
	var cases []historyPrunedCase
	for _, persistenceThreshold := range persistenceThresholds {
		cases = append(cases, historyPrunedCase{persistenceThreshold: persistenceThreshold})
	}
	return cases
}

func newHistoryPrunedSystem(t devtest.T, c historyPrunedCase) *utils.MixedOpProofPreset {
	return newProofsFollowerSystem(t, engineArgs(c.persistenceThreshold),
		sysgo.OpRethWithProofsHistoryWindow(historyPrunedProofsWindow),
		sysgo.OpRethWithExtraArgs(c.followerArgs(t)...),
	)
}

// rethPrunerLogs are the follower's reth pruner lines the test reports when the prune checkpoint
// does not move: the engine deciding to run the pruner after a persistence, the pruner holding
// back until every ExEx acknowledged a height, the height it prunes up to, and each segment's
// outcome.
type rethPrunerLogs struct {
	adjustedToExExHeight *logpipe.LogRecord
	notReady             *logpipe.LogRecord
	running              *logpipe.LogRecord
	segmentFinished      *logpipe.LogRecord
}

func recordRethPrunerLogs(follower *dsl.L2ELNode) rethPrunerLogs {
	return rethPrunerLogs{
		adjustedToExExHeight: follower.RecordLogLines("Adjusting tip block number to the finished ExEx height"),
		notReady:             follower.RecordLogLines("Not all ExExs have emitted a `FinishedHeight` event yet"),
		running:              follower.RecordLogLines("Running pruner"),
		segmentFinished:      follower.RecordLogLines("Segment pruning finished"),
	}
}

func (l rethPrunerLogs) String() string {
	return fmt.Sprintf("running=%d adjusted_to_exex_height=%d not_ready=%d segments_finished=%d; last adjustment: %s; last segment: %s",
		l.running.Count(), l.adjustedToExExHeight.Count(), l.notReady.Count(), l.segmentFinished.Count(),
		lastLogLine(l.adjustedToExExHeight), lastLogLine(l.segmentFinished))
}

func lastLogLine(record *logpipe.LogRecord) string {
	entries := record.Entries()
	if len(entries) == 0 {
		return "none"
	}
	last := entries[len(entries)-1]
	return fmt.Sprintf("%s %v", last.LogMessage(), last.LogFields())
}

// rethHistoryStatus reports, for each block, whether reth's own historical state is still served,
// gone, or failing for another reason.
func rethHistoryStatus(ctx context.Context, node *dsl.L2ELNode, account common.Address, blocks []uint64) map[uint64]string {
	status := make(map[uint64]string, len(blocks))
	for _, block := range blocks {
		_, err := node.EthClient().GetStorageAt(ctx, account, recorderNumberSlot, blockNumberTag(block))
		switch {
		case err == nil:
			status[block] = rethHistoryServed
		case strings.Contains(err.Error(), rethHistoryPruned):
			status[block] = rethHistoryPruned
		default:
			status[block] = err.Error()
		}
	}
	return status
}

// waitForRethHistoryPrunedAt waits until reth's own historical state at block is gone on node,
// which eth_getStorageAt reports as pruned. The pruner runs after reth persists, so the wait only
// covers the run that may still be in flight once the chain stopped. Blocks further below are
// probed too, so a failure shows where reth's history actually ends.
func waitForRethHistoryPrunedAt(t devtest.T, node *dsl.L2ELNode, account common.Address, logs rethPrunerLogs, block uint64) {
	ctx, cancel := context.WithTimeout(t.Ctx(), historyPruneTimeout)
	defer cancel()
	probes := []uint64{block, block - 1, block / 2, 1}
	var status map[uint64]string
	err := wait.For(ctx, time.Second, func() (bool, error) {
		status = rethHistoryStatus(ctx, node, account, probes)
		if status[block] == rethHistoryPruned {
			return true, nil
		}
		t.Logger().Info("reth still serves its own state at block; waiting for the pruner", "block", block, "history", status, "pruner", logs.String())
		return false, nil
	})
	t.Require().NoErrorf(err, "reth's history prune checkpoint must pass block %d within %s for the block to be served from the proofs store alone; reth history by block: %v; pruner: %s",
		block, historyPruneTimeout, status, logs)
	t.Logger().Info("reth's own history at block is pruned", "block", block, "history", status, "pruner", logs.String())
}

// TestProofsFollowerServesBlocksBelowRethHistoryPruneCheckpoint runs a proofs follower whose reth
// prunes its account and storage history before a fixed block, with a proofs window that covers
// the whole chain. Once reth's own state at the highest block it refuses is gone, eth_getProof
// there must still verify and match the sequencer's, and debug_executionWitness for the block
// after it must still execute: the proofs store is what serves the window, not reth's history.
func TestProofsFollowerServesBlocksBelowRethHistoryPruneCheckpoint(gt *testing.T) {
	t := devtest.ParallelT(gt)
	for _, c := range historyPrunedMatrix() {
		t.Run(c.name(), func(t devtest.T) {
			t.Parallel()
			sys := newHistoryPrunedSystem(t, c)
			follower, reference := sys.L2ELValidator, sys.L2ELSequencer
			prunerLogs := recordRethPrunerLogs(follower)
			w := newWorkload(t, sys.FunderL2)
			// reth still answers for the prune target block itself, whose state only needs the
			// changesets above it; the block below it is the first one reth refuses.
			checked := uint64(historyPruneBefore - 2)
			t.Require().Lessf(w.setupBlock, checked, "the workload's accounts and contract (block %d) must exist at the checked block %d", w.setupBlock, checked)

			w.driveUntilBlock(c.headWithPrunedHistory())
			_, window := stopChainAndSettleStore(t, sys)
			t.Require().LessOrEqualf(*window.Earliest, checked, "the store must hold the checked block %d; window %s", checked, window)
			waitForRethHistoryPrunedAt(t, follower, w.recorder, prunerLogs, checked)

			follower.VerifyProofsMatchAt(reference, checked, w.accounts()...)
			witness, err := executionWitness(t, follower, checked+1)
			t.Require().NoErrorf(err, "debug_executionWitness for block %d, whose parent's state reth no longer holds, must be served from the proofs store", checked+1)
			requireWitnessNonEmpty(t, witness, checked+1)
		})
	}
}
