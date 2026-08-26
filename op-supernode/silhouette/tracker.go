package silhouette

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

// ProvenHeadTracker walks L1 through the acceptance path to keep a fact store current on a node
// that does NOT derive P.
//
// In the verifier posture nothing needs this: the derivation pipeline calls OpenData as it
// traverses L1, so acceptance happens as a side effect of deriving the chain, and the fact store
// fills itself. The SEQUENCER posture has no such pipeline — P's container there fronts the real
// execution client, which is producing blocks from the private sequencer rather than deriving them
// from anything — so the walk has to be driven explicitly.
//
// It drives the SAME DataSource, which is the point. Acceptance rules, chaining, the forced
// extension, the rendered origins and the log sink are one implementation with one set of tests,
// and the sequencer's view of what P has proven is therefore the same view every verifier has,
// arrived at by the same code. A second, simpler "just read the head" path would be a second
// implementation of acceptance, and the first time the two disagreed the disagreement would be
// between a chain's sequencer and its verifiers.
type ProvenHeadTracker struct {
	log log.Logger
	src *DataSource
	l1  L1Headers

	// next is the L1 block to read. Acceptance is a pure function of L1 (G2 D5), so this cursor is
	// the only state the tracker has, and a restart from the configured start block re-derives
	// identical facts.
	next uint64
	// interval is how long to wait after catching up with L1 before looking again.
	interval time.Duration
}

// NewProvenHeadTracker builds a tracker starting at the configured L1 start block.
func NewProvenHeadTracker(logger log.Logger, src *DataSource, l1 L1Headers, startBlock uint64, interval time.Duration) *ProvenHeadTracker {
	if interval <= 0 {
		interval = 2 * time.Second
	}
	return &ProvenHeadTracker{log: logger, src: src, l1: l1, next: startBlock, interval: interval}
}

// Run drives the tracker until ctx is cancelled. It is expected to be called in its own goroutine.
func (t *ProvenHeadTracker) Run(ctx context.Context) {
	for {
		if err := ctx.Err(); err != nil {
			return
		}
		advanced, err := t.Step(ctx)
		switch {
		case err != nil:
			t.log.Warn("proven-head tracker step failed; retrying", "l1", t.next, "err", err)
		case advanced:
			continue // more L1 to read, no reason to wait
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(t.interval):
		}
	}
}

// Step reads at most one L1 block through the acceptance path. It reports whether it advanced,
// so a caller catching up does not sleep between blocks.
//
// Returning after ONE block rather than looping to the head is what makes this testable as a
// stepper and cancellable as a loop, and it costs nothing: Run continues immediately when a step
// advanced.
func (t *ProvenHeadTracker) Step(ctx context.Context) (bool, error) {
	ref, err := t.l1.L1BlockRefByNumber(ctx, t.next)
	if errors.Is(err, ethereum.NotFound) {
		return false, nil // caught up with L1
	}
	if err != nil {
		return false, fmt.Errorf("fetch L1 block %d: %w", t.next, err)
	}
	iter, err := t.src.OpenData(ctx, ref, common.Address{})
	if err != nil {
		return false, fmt.Errorf("open L1 block %d: %w", t.next, err)
	}
	// Drain the iterator. Acceptance happens inside Next: a proof batch found in this L1 block is
	// verified, chained, and — through the sink — sealed into the interop log database, all before
	// the cursor moves. The frames it yields are what a derivation pipeline would consume; here
	// there is no pipeline, and discarding them is correct rather than lossy, because the facts and
	// the messages are the things this node needed and they are already recorded.
	for {
		if _, err := iter.Next(ctx); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return false, fmt.Errorf("read L1 block %d: %w", t.next, err)
		}
	}
	t.next++
	return true, nil
}

// Cursor is the next L1 block the tracker will read.
func (t *ProvenHeadTracker) Cursor() uint64 { return t.next }
