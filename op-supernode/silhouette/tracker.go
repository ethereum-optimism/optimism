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

// ProvenHeadTracker walks L1 through the acceptance path to keep the standalone Silhouette EL's fact
// store current. The supernode has its own stock derivation pipeline and does not use this walker.
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
	// start is the earliest L1 block this verifier is allowed to replay.
	start uint64

	// next and processed are checkpointed atomically with the fact store. A restart continues from
	// the exact L1 boundary whose facts are durable instead of replaying from the configured start.
	next uint64
	// processed records the canonical hash observed at every processed L1 height. It lets a
	// caught-up walker detect an L1 reorg that made the chain shorter: fetching next returns
	// NotFound on both an ordinary tip and a shortened fork, so the previous hashes disambiguate
	// those cases.
	processed map[uint64]common.Hash
	// interval is how long to wait after catching up with L1 before looking again.
	interval time.Duration
}

// NewProvenHeadTracker builds a tracker starting at the configured L1 start block.
func NewProvenHeadTracker(logger log.Logger, src *DataSource, l1 L1Headers, startBlock uint64, interval time.Duration) *ProvenHeadTracker {
	if interval <= 0 {
		interval = 2 * time.Second
	}
	state := trackerState{Initialized: true, Start: startBlock, Next: startBlock, Processed: make(map[uint64]common.Hash)}
	if src != nil {
		state = src.facts.trackerState(startBlock)
	}
	tracker := &ProvenHeadTracker{
		log: logger, src: src, l1: l1, start: startBlock, next: state.Next,
		processed: state.Processed, interval: interval,
	}
	if src != nil && tracker.next > tracker.start {
		if hash, ok := tracker.processed[tracker.next-1]; ok {
			src.restoreL1Cursor(tracker.next-1, hash)
		}
	}
	return tracker
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
			t.log.Warn("proof walker step failed; retrying", "l1", t.next, "err", err)
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
		return t.reconcileShortenedL1(ctx)
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
	oldNext := t.next
	t.processed[oldNext] = ref.Hash
	t.next = oldNext + 1
	if err := t.checkpoint(); err != nil {
		delete(t.processed, oldNext)
		t.next = oldNext
		t.src.facts.setTrackerState(t.state())
		return false, err
	}
	return true, nil
}

// reconcileShortenedL1 distinguishes an ordinary caught-up cursor from an L1 reorg whose new tip
// is below that cursor. It searches backward for the newest height whose hash is still canonical,
// then replays from its child. DataSource.OpenData performs the corresponding fact/log rollback
// when that child is opened.
func (t *ProvenHeadTracker) reconcileShortenedL1(ctx context.Context) (bool, error) {
	if t.next <= t.start || len(t.processed) == 0 {
		return false, nil
	}
	newNext := t.start
	for n := t.next; n > t.start; {
		n--
		want, ok := t.processed[n]
		if !ok {
			continue
		}
		got, err := t.l1.L1BlockRefByNumber(ctx, n)
		if errors.Is(err, ethereum.NotFound) {
			continue
		}
		if err != nil {
			return false, fmt.Errorf("check canonical L1 block %d: %w", n, err)
		}
		if got.Hash == want {
			newNext = n + 1
			break
		}
	}
	if newNext == t.next {
		return false, nil
	}
	oldNext := t.next
	for n := newNext; n < oldNext; n++ {
		delete(t.processed, n)
	}
	t.next = newNext
	if err := t.checkpoint(); err != nil {
		return false, err
	}
	t.log.Warn("L1 shortened or reorged behind proof cursor; replaying from canonical ancestor",
		"old_next", oldNext, "new_next", newNext)
	return true, nil
}

// Cursor is the next L1 block the tracker will read.
func (t *ProvenHeadTracker) Cursor() uint64 { return t.next }

func (t *ProvenHeadTracker) state() trackerState {
	return trackerState{Initialized: true, Start: t.start, Next: t.next, Processed: t.processed}
}

func (t *ProvenHeadTracker) checkpoint() error {
	if t.src == nil {
		return nil
	}
	t.src.facts.setTrackerState(t.state())
	if err := t.src.facts.Flush(); err != nil {
		return fmt.Errorf("checkpoint proof walker at L1 block %d: %w", t.next, err)
	}
	return nil
}
