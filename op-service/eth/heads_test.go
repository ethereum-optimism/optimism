package eth

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/log"
)

type staticBlockRefsSource struct {
	ref   L1BlockRef
	calls atomic.Int32
}

func (s *staticBlockRefsSource) L1BlockRefByLabel(ctx context.Context, label BlockLabel) (L1BlockRef, error) {
	s.calls.Add(1)
	return s.ref, nil
}

// TestPollBlockChanges_PrimesImmediately asserts that PollBlockChanges polls
// once immediately on subscription instead of waiting a full interval for the
// first ticker fire. Without the priming poll, op-node's L1 safe/finalized
// subscriptions (L1EpochPollInterval, default 384s) leave a fresh (virtual)
// node with a zero FinalizedL1 for one whole interval — see issue #22127.
func TestPollBlockChanges_PrimesImmediately(t *testing.T) {
	t.Parallel()
	src := &staticBlockRefsSource{ref: L1BlockRef{Number: 11370960, Time: 1785271000}}
	signals := make(chan L1BlockRef, 4)
	fn := func(ctx context.Context, sig L1BlockRef) {
		signals <- sig
	}
	// Interval mirrors the production default (L1EpochPollInterval = 384s):
	// long enough that only a priming poll can deliver the first signal.
	sub := PollBlockChanges(log.NewLogger(log.DiscardHandler()), src, fn, Finalized,
		384*time.Second, 10*time.Second)
	defer sub.Unsubscribe()

	select {
	case sig := <-signals:
		if sig.Number != src.ref.Number {
			t.Fatalf("unexpected signal: %v", sig)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("PollBlockChanges delivered no %s signal within 2s of subscription; "+
			"the first poll must fire immediately, not after a full interval; "+
			"L1BlockRefByLabel calls=%d", Finalized, src.calls.Load())
	}
}
