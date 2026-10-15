package core

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	safety "github.com/ethereum-optimism/optimism/op-service/eth/safety"
	"github.com/ethereum-optimism/optimism/op-service/logpipe"
)

const (
	// postUnwindTrafficBlocks is how far past the sequencer's head the chain is driven once the
	// follower is back, so the sweep also covers blocks the store indexed live after the unwind.
	postUnwindTrafficBlocks = 10
	// unwindPersistenceThreshold is the one reth setting the smoke test runs under; the unwind
	// target matrix is covered by the CLI's own tests.
	unwindPersistenceThreshold = 7
	// unwoundBlocks is how far below the head the store had indexed before the stop the unwind
	// target sits; the store removes at least that many blocks, more if it indexed past the head
	// before the stop.
	unwoundBlocks = 5
	// executedBlockLogMessage is what the proofs store logs for every block it executes, which on
	// a restart is every block of the datadir above the store's tip.
	executedBlockLogMessage = "Block executed and trie updates buffered successfully"
)

// lowestExecutedBlock returns the lowest block the proofs store executed since executed was
// recorded, which after a restart is the block right above the store's tip.
func lowestExecutedBlock(t devtest.T, executed *logpipe.LogRecord) uint64 {
	entries := executed.Entries()
	t.Require().NotEmpty(entries, "the restarted store executed no block, so the unwind removed nothing")
	lowest := uint64(math.MaxUint64)
	for _, entry := range entries {
		number, err := blockNumberField(entry.FieldValue("block_number"))
		t.Require().NoErrorf(err, "block_number of %v", entry.LogFields())
		lowest = min(lowest, number)
	}
	return lowest
}

func blockNumberField(value any) (uint64, error) {
	switch v := value.(type) {
	case json.Number:
		return strconv.ParseUint(v.String(), 10, 64)
	case float64:
		return uint64(v), nil
	default:
		return strconv.ParseUint(fmt.Sprint(v), 10, 64)
	}
}

// TestProofsFollowerServesTheWindowAfterItsStoreIsUnwoundAndRestarted stops a proofs follower
// cleanly, removes the last blocks from its store with `op-reth proofs unwind`, the recovery for a
// store that got ahead of reth's datadir, and brings op-reth back. The store must resume right
// above the target and re-index the datadir up to its tip before the follower's CL is back, and
// every proof of the window, the re-indexed blocks and the blocks indexed live afterwards
// included, must verify against its header and match the sequencer's.
func TestProofsFollowerServesTheWindowAfterItsStoreIsUnwoundAndRestarted(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := newProofsFollowerSystem(t, engineArgs(unwindPersistenceThreshold),
		sysgo.OpRethWithProofsHistoryWindow(proofsHistoryWindow),
		sysgo.OpRethWithExtraArgs(engineArgs(unwindPersistenceThreshold)...),
	)
	follower, followerCL, reference := sys.L2ELValidator, sys.L2CLValidator, sys.L2ELSequencer
	w := newWorkload(t, sys)

	head := w.driveUntilBlock(w.setupBlock + unwoundBlocks)
	follower.Reached(eth.Unsafe, head, blockAttempts)
	follower.WaitForProofsStoreBlock(head, storeCatchUpTimeout)
	target := head - unwoundBlocks

	followerCL.Stop()
	follower.Stop()
	report := follower.UnwindProofsHistory(target)
	t.Logger().Info("Proofs unwind ran against the stopped follower's datadir", "target", target, "head_at_stop", head, "duration", report.Duration)

	executed := follower.RecordLogLines(executedBlockLogMessage)
	follower.Start()
	follower.WaitForOnline()
	tip := follower.BlockRefByLabel(eth.Unsafe)
	t.Require().GreaterOrEqualf(tip.Number, head, "the datadir tip %s must hold the head %d the store had indexed before the stop", tip, head)
	window := follower.WaitForProofsStoreBlock(tip.Number, storeCatchUpTimeout)
	t.Require().Equalf(tip.Number, *window.Latest, "the store must re-index the datadir up to its tip %s with nothing else feeding the node; window %s", tip, window)
	t.Require().Equalf(target+1, lowestExecutedBlock(t, executed),
		"the restarted store must resume right above the unwind target %d; window %s", target, window)
	follower.VerifyHistoricalProofsMatch(reference, func(opts *dsl.HistoricalProofsOpts) {
		opts.Accounts = w.accounts()
		opts.LastBlock = tip.Number
	})

	follower.PeerWith(reference)
	followerCL.Start()
	followerCL.ConnectPeer(sys.L2CLSequencer)
	follower.InSync(reference, safety.LocalUnsafe, syncAttempts)
	lastBlock := w.driveUntilBlock(reference.BlockRefByLabel(eth.Unsafe).Number + postUnwindTrafficBlocks)
	follower.VerifyHistoricalProofsMatch(reference, func(opts *dsl.HistoricalProofsOpts) {
		opts.Accounts = w.accounts()
		opts.LastBlock = lastBlock
	})
}
