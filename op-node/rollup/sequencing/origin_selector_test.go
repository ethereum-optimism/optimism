package sequencing

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"testing"
	"time"

	optypes "github.com/ethereum-optimism/optimism/op-core/types"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/confdepth"
	"github.com/ethereum-optimism/optimism/op-node/rollup/engine"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// TestOriginSelectorFetchCurrentError ensures that the origin selector
// returns an error when it cannot fetch the current origin and has no
// internal cached state.
func TestOriginSelectorFetchCurrentError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	log := testlog.Logger(t, log.LevelCrit)
	cfg := &rollup.Config{
		MaxSequencerDrift: 500,
		BlockTime:         2,
	}
	l1 := &testutils.MockL1Source{}
	defer l1.AssertExpectations(t)
	a := eth.L1BlockRef{
		Hash:   common.Hash{'a'},
		Number: 10,
		Time:   20,
	}
	b := eth.L1BlockRef{
		Hash:       common.Hash{'b'},
		Number:     11,
		Time:       25,
		ParentHash: a.Hash,
	}
	l2Head := eth.L2BlockRef{
		L1Origin: a.ID(),
		Time:     24,
	}

	l1.ExpectL1BlockRefByHash(a.Hash, eth.L1BlockRef{}, errors.New("test error"))

	s := newTestSelector(ctx, log, cfg, l1)

	_, err := s.FindL1Origin(ctx, l2Head)
	require.ErrorContains(t, err, "test error")

	// The same outcome occurs when the cached origin is different from that of the L2 head.
	l1.ExpectL1BlockRefByHash(a.Hash, eth.L1BlockRef{}, errors.New("test error"))

	s = newTestSelector(ctx, log, cfg, l1)
	s.currentOrigin = b

	_, err = s.FindL1Origin(ctx, l2Head)
	require.ErrorContains(t, err, "test error")
}

// TestOriginSelectorFetchNextError ensures that the origin selector
// gracefully handles an error when fetching the next origin from the
// forkchoice update event.
func TestOriginSelectorFetchNextError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	log := testlog.Logger(t, log.LevelCrit)
	cfg := &rollup.Config{
		MaxSequencerDrift: 500,
		BlockTime:         2,
	}
	l1 := &testutils.MockL1Source{}
	defer l1.AssertExpectations(t)
	a := eth.L1BlockRef{
		Hash:   common.Hash{'a'},
		Number: 10,
		Time:   20,
	}
	b := eth.L1BlockRef{
		Hash:   common.Hash{'b'},
		Number: 11,
	}
	l2Head := eth.L2BlockRef{
		L1Origin: a.ID(),
		Time:     24,
	}

	s := newTestSelector(ctx, log, cfg, l1)
	s.currentOrigin = a

	next, err := s.FindL1Origin(ctx, l2Head)
	require.Nil(t, err)
	require.Equal(t, a, next)

	l1.ExpectL1BlockRefByNumber(b.Number, eth.L1BlockRef{}, ethereum.NotFound)

	handled := s.OnEvent(context.Background(), engine.ForkchoiceUpdateEvent{UnsafeL2Head: l2Head})
	require.True(t, handled)

	l1.ExpectL1BlockRefByNumber(b.Number, eth.L1BlockRef{}, errors.New("test error"))

	handled = s.OnEvent(context.Background(), engine.ForkchoiceUpdateEvent{UnsafeL2Head: l2Head})
	require.True(t, handled)

	// The next origin should still be `a` because the fetch failed.
	next, err = s.FindL1Origin(ctx, l2Head)
	require.Nil(t, err)
	require.Equal(t, a, next)
}

// TestOriginSelectorAdvances ensures that the origin selector
// advances the origin with the internal cache
//
// There are 3 L1 blocks at times 20, 22, 24. The L2 Head is at time 24.
// The next L2 time is 26 which is after the next L1 block time. There
// is no conf depth to stop the origin selection so block `b` should
// be the next L1 origin, and then block `c` is the subsequent L1 origin.
func TestOriginSelectorAdvances(t *testing.T) {
	testOriginSelectorAdvances := func(t *testing.T, recoverMode bool) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		log := testlog.Logger(t, log.LevelCrit)
		cfg := &rollup.Config{
			MaxSequencerDrift: 500,
			BlockTime:         2,
		}
		l1 := &testutils.MockL1Source{}
		defer l1.AssertExpectations(t)
		a := eth.L1BlockRef{
			Hash:   common.Hash{'a'},
			Number: 10,
			Time:   20,
		}
		b := eth.L1BlockRef{
			Hash:       common.Hash{'b'},
			Number:     11,
			Time:       22,
			ParentHash: a.Hash,
		}
		c := eth.L1BlockRef{
			Hash:       common.Hash{'c'},
			Number:     12,
			Time:       24,
			ParentHash: b.Hash,
		}
		d := eth.L1BlockRef{
			Hash:       common.Hash{'d'},
			Number:     13,
			Time:       36,
			ParentHash: c.Hash,
		}
		l2Head := eth.L2BlockRef{
			L1Origin: a.ID(),
			Time:     24,
		}

		s := newTestSelector(ctx, log, cfg, l1)

		requireL1OriginAt := func(l2Head eth.L2BlockRef, want eth.L1BlockRef) {
			got, err := s.FindL1Origin(ctx, l2Head)
			require.Nil(t, err)
			require.Equal(t, want, got)
		}

		s.currentOrigin = a
		s.nextOrigin = b

		// Trigger the background fetch via a forkchoice update.
		// The next origin is already cached, so this only prefetches its successor.
		l1.ExpectL1BlockRefByNumber(c.Number, c, nil)
		handled := s.OnEvent(context.Background(), engine.ForkchoiceUpdateEvent{UnsafeL2Head: l2Head})
		require.True(t, handled)

		requireL1OriginAt(l2Head, b)

		l2Head = eth.L2BlockRef{
			L1Origin: b.ID(),
			Time:     26,
		}

		// Adopting `b` promoted the prefetched `c` to next origin, so the origin
		// advances again without waiting for another forkchoice update.
		requireL1OriginAt(l2Head, c)

		// The forkchoice update now looks further ahead, but `d` is not available yet.
		l1.ExpectL1BlockRefByNumber(d.Number, eth.BlockRef{}, ethereum.NotFound)
		handled = s.OnEvent(context.Background(), engine.ForkchoiceUpdateEvent{UnsafeL2Head: l2Head})
		require.True(t, handled)

		requireL1OriginAt(l2Head, c)

		// Now force the retrieval of the next L1 origin
		s.recoverMode.Store(recoverMode)

		l2Head = eth.L2BlockRef{
			L1Origin: c.ID(),
			Time:     d.Time + 4,
		}

		if recoverMode {
			// In recovery mode (only) we make an RPC call to find the next origin.
			// First, cover the case where the nextOrigin
			// is not ready yet by simulating a NotFound error.
			l1.ExpectL1BlockRefByNumber(d.Number, eth.BlockRef{}, ethereum.NotFound)
			requireL1OriginAt(l2Head, c)

			// Now, simulate the block being ready, and ensure
			// that the origin advances to the next block.
			l1.ExpectL1BlockRefByNumber(d.Number, d, nil)
			requireL1OriginAt(l2Head, d)
		} else {
			requireL1OriginAt(l2Head, c)
		}
	}

	t.Run("normal", func(t *testing.T) { testOriginSelectorAdvances(t, false) })
	t.Run("recover_mode", func(t *testing.T) { testOriginSelectorAdvances(t, true) })
}

// TestOriginSelectorHandlesReset ensures that the origin selector
// resets its internal cached state on derivation pipeline resets.
func TestOriginSelectorHandlesReset(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	log := testlog.Logger(t, log.LevelCrit)
	cfg := &rollup.Config{
		MaxSequencerDrift: 500,
		BlockTime:         2,
	}
	l1 := &testutils.MockL1Source{}
	defer l1.AssertExpectations(t)
	a := eth.L1BlockRef{
		Hash:   common.Hash{'a'},
		Number: 10,
		Time:   20,
	}
	b := eth.L1BlockRef{
		Hash:       common.Hash{'b'},
		Number:     11,
		Time:       25,
		ParentHash: a.Hash,
	}
	l2Head := eth.L2BlockRef{
		L1Origin: a.ID(),
		Time:     24,
	}

	s := newTestSelector(ctx, log, cfg, l1)
	s.currentOrigin = a
	s.nextOrigin = b

	next, err := s.FindL1Origin(ctx, l2Head)
	require.Nil(t, err)
	require.Equal(t, b, next)

	// Trigger the pipeline reset
	handled := s.OnEvent(context.Background(), rollup.ResetEvent{})
	require.True(t, handled)

	// The next origin should be `a` now, but we need to fetch it
	// because the internal cache was reset.
	l1.ExpectL1BlockRefByHash(a.Hash, a, nil)

	next, err = s.FindL1Origin(ctx, l2Head)
	require.Nil(t, err)
	require.Equal(t, a, next)
}

// TestOriginSelectorFetchesNextOrigin ensures that the origin selector
// fetches the next origin when a fcu is received and the internal cache is empty
//
// The next L2 time is 26 which is after the next L1 block time. There
// is no conf depth to stop the origin selection so block `b` will
// be the next L1 origin as soon as it is fetched.
func TestOriginSelectorFetchesNextOrigin(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	log := testlog.Logger(t, log.LevelCrit)
	cfg := &rollup.Config{
		MaxSequencerDrift: 500,
		BlockTime:         2,
	}
	l1 := &testutils.MockL1Source{}
	defer l1.AssertExpectations(t)
	a := eth.L1BlockRef{
		Hash:   common.Hash{'a'},
		Number: 10,
		Time:   20,
	}
	b := eth.L1BlockRef{
		Hash:       common.Hash{'b'},
		Number:     11,
		Time:       25,
		ParentHash: a.Hash,
	}
	l2Head := eth.L2BlockRef{
		L1Origin: a.ID(),
		Time:     24,
	}

	// These are called as part of the background prefetch job
	l1.ExpectL1BlockRefByNumber(b.Number, b, nil)
	l1.ExpectL1BlockRefByNumber(b.Number+1, eth.L1BlockRef{}, ethereum.NotFound)

	s := newTestSelector(ctx, log, cfg, l1)
	s.currentOrigin = a

	next, err := s.FindL1Origin(ctx, l2Head)
	require.Nil(t, err)
	require.Equal(t, a, next)

	// Selection is stable until the next origin is fetched
	next, err = s.FindL1Origin(ctx, l2Head)
	require.Nil(t, err)
	require.Equal(t, a, next)

	// Trigger the background fetch via a forkchoice update
	handled := s.OnEvent(context.Background(), engine.ForkchoiceUpdateEvent{UnsafeL2Head: l2Head})
	require.True(t, handled)

	// The next origin should be `b` now.
	next, err = s.FindL1Origin(ctx, l2Head)
	require.Nil(t, err)
	require.Equal(t, b, next)
}

// TestOriginSelectorHandlesReorg ensures that the origin selector
// can handle the current origin being reorged out
//
// There are 3 blocks [a, b, c]. After advancing to b, a reorg is simulated
// where b is reorged and replaced by providing a `c` next that has a different parent hash.
// A sentinel error should be returned.
func TestOriginSelectorHandlesReorg(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	log := testlog.Logger(t, log.LevelDebug)
	cfg := &rollup.Config{
		MaxSequencerDrift: 500,
		BlockTime:         2,
	}
	l1 := &testutils.MockL1Source{}
	defer l1.AssertExpectations(t)
	a := eth.L1BlockRef{
		Hash:   common.Hash{'a'},
		Number: 10,
		Time:   20,
	}
	b := eth.L1BlockRef{
		Hash:       common.Hash{'b'},
		Number:     11,
		Time:       22,
		ParentHash: a.Hash,
	}
	l2Head := eth.L2BlockRef{
		L1Origin: a.ID(),
		Time:     24,
	}

	// A reorg happens and `b` is replaced by a block with a different hash,
	// which the canonical `c` builds on.
	c := eth.L1BlockRef{
		Hash:       common.Hash{'c'},
		Number:     12,
		Time:       24,
		ParentHash: common.Hash{'b', '2'},
	}

	// These are called as part of the background prefetch job. The lookahead
	// already sees the reorged chain.
	l1.ExpectL1BlockRefByNumber(b.Number, b, nil)
	l1.ExpectL1BlockRefByNumber(c.Number, c, nil)

	s := newTestSelector(ctx, log, cfg, l1)
	s.currentOrigin = a

	requireFindl1OriginEqual := func(l1ref eth.L1BlockRef) {
		next, err := s.FindL1Origin(ctx, l2Head)
		require.NoError(t, err)
		require.Equal(t, l1ref, next)
	}

	requireFindL1OriginError := func(e error) {
		_, err := s.FindL1Origin(ctx, l2Head)
		require.ErrorIs(t, err, e)
	}

	requireFindl1OriginEqual(a)

	// Selection is stable until the next origin is fetched
	requireFindl1OriginEqual(a)

	// Trigger the background fetch via a forkchoice update
	handled := s.OnEvent(context.Background(), engine.ForkchoiceUpdateEvent{UnsafeL2Head: l2Head})
	require.True(t, handled)

	// The next origin should be `b` now.
	requireFindl1OriginEqual(b)

	l2Head = eth.L2BlockRef{
		L1Origin: b.ID(),
		Time:     26,
	}

	// The lookahead `c` does not extend `b`, so it is not promoted along with `b`:
	// the forkchoice update fetches the next origin again.
	l1.ExpectL1BlockRefByNumber(c.Number, c, nil)
	l1.ExpectL1BlockRefByNumber(c.Number+1, eth.L1BlockRef{}, ethereum.NotFound)
	handled = s.OnEvent(context.Background(), engine.ForkchoiceUpdateEvent{UnsafeL2Head: l2Head})
	require.True(t, handled)

	// We shuold get a sentinel error
	requireFindL1OriginError(ErrNextL1OriginOrphaned)
}

// TestOriginSelectorRespectsOriginTiming ensures that the origin selector
// does not pick an origin that is ahead of the next L2 block time
//
// There are 2 L1 blocks at time 20 & 25. The L2 Head is at time 22.
// The next L2 time is 24 which is before the next L1 block time. There
// is no conf depth to stop the LOS from potentially selecting block `b`
// but it should select block `a` because the L2 block time must be ahead
// of the the timestamp of it's L1 origin.
func TestOriginSelectorRespectsOriginTiming(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	log := testlog.Logger(t, log.LevelCrit)
	cfg := &rollup.Config{
		MaxSequencerDrift: 500,
		BlockTime:         2,
	}
	l1 := &testutils.MockL1Source{}
	defer l1.AssertExpectations(t)
	a := eth.L1BlockRef{
		Hash:   common.Hash{'a'},
		Number: 10,
		Time:   20,
	}
	b := eth.L1BlockRef{
		Hash:       common.Hash{'b'},
		Number:     11,
		Time:       25,
		ParentHash: a.Hash,
	}
	l2Head := eth.L2BlockRef{
		L1Origin: a.ID(),
		Time:     22,
	}

	s := newTestSelector(ctx, log, cfg, l1)
	s.currentOrigin = a
	s.nextOrigin = b

	next, err := s.FindL1Origin(ctx, l2Head)
	require.Nil(t, err)
	require.Equal(t, a, next)
}

// TestOriginSelectorRespectsSeqDrift
//
// There are 2 L1 blocks at time 20 & 25. The L2 Head is at time 27.
// The next L2 time is 29. The sequencer drift is 8 so the L2 head is
// valid with origin `a`, but the next L2 block is not valid with origin `b.`
// This is because 29 (next L2 time) > 20 (origin) + 8 (seq drift) => invalid block.
// The origin selector does not yet know about block `b` so it should wait for the
// background fetch to complete synchronously.
func TestOriginSelectorRespectsSeqDrift(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	log := testlog.Logger(t, log.LevelCrit)
	cfg := &rollup.Config{
		MaxSequencerDrift: 8,
		BlockTime:         2,
	}
	l1 := &testutils.MockL1Source{}
	defer l1.AssertExpectations(t)
	a := eth.L1BlockRef{
		Hash:   common.Hash{'a'},
		Number: 10,
		Time:   20,
	}
	b := eth.L1BlockRef{
		Hash:       common.Hash{'b'},
		Number:     11,
		Time:       25,
		ParentHash: a.Hash,
	}
	l2Head := eth.L2BlockRef{
		L1Origin: a.ID(),
		Time:     27,
	}

	l1.ExpectL1BlockRefByHash(a.Hash, a, nil)

	l1.ExpectL1BlockRefByNumber(b.Number, b, nil)

	s := newTestSelector(ctx, log, cfg, l1)

	next, err := s.FindL1Origin(ctx, l2Head)
	require.NoError(t, err)
	require.Equal(t, b, next)
}

// TestOriginSelectorRespectsConfDepth ensures that the origin selector
// will respect the confirmation depth requirement
//
// There are 2 L1 blocks at time 20 & 25. The L2 Head is at time 27.
// The next L2 time is 29 which enough to normally select block `b`
// as the origin, however block `b` is the L1 Head & the sequencer
// needs to wait until that block is confirmed enough before advancing.
func TestOriginSelectorRespectsConfDepth(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	log := testlog.Logger(t, log.LevelCrit)
	cfg := &rollup.Config{
		MaxSequencerDrift: 500,
		BlockTime:         2,
	}
	l1 := &testutils.MockL1Source{}
	defer l1.AssertExpectations(t)
	a := eth.L1BlockRef{
		Hash:   common.Hash{'a'},
		Number: 10,
		Time:   20,
	}
	b := eth.L1BlockRef{
		Hash:       common.Hash{'b'},
		Number:     11,
		Time:       25,
		ParentHash: a.Hash,
	}
	l2Head := eth.L2BlockRef{
		L1Origin: a.ID(),
		Time:     27,
	}

	confDepthL1 := confdepth.NewConfDepth(10, func() eth.L1BlockRef { return b }, l1)
	s := newTestSelector(ctx, log, cfg, confDepthL1)
	s.currentOrigin = a

	next, err := s.FindL1Origin(ctx, l2Head)
	require.Nil(t, err)
	require.Equal(t, a, next)
}

// TestOriginSelectorStrictConfDepth ensures that the origin selector will maintain the sequencer conf depth,
// even while the time delta between the current L1 origin and the next
// L2 block is greater than the sequencer drift.
// It's more important to maintain safety with an empty block than to maintain liveness with poor conf depth.
//
// There are 2 L1 blocks at time 20 & 25. The L2 Head is at time 27.
// The next L2 time is 29. The sequencer drift is 8 so the L2 head is
// valid with origin `a`, but the next L2 block is not valid with origin `b.`
// This is because 29 (next L2 time) > 20 (origin) + 8 (seq drift) => invalid block.
// We maintain confirmation distance, even though we would shift to the next origin if we could.
func TestOriginSelectorStrictConfDepth(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	log := testlog.Logger(t, log.LevelCrit)
	cfg := &rollup.Config{
		MaxSequencerDrift: 8,
		BlockTime:         2,
	}
	l1 := &testutils.MockL1Source{}
	defer l1.AssertExpectations(t)
	a := eth.L1BlockRef{
		Hash:   common.Hash{'a'},
		Number: 10,
		Time:   20,
	}
	b := eth.L1BlockRef{
		Hash:       common.Hash{'b'},
		Number:     11,
		Time:       25,
		ParentHash: a.Hash,
	}
	l2Head := eth.L2BlockRef{
		L1Origin: a.ID(),
		Time:     27,
	}

	l1.ExpectL1BlockRefByHash(a.Hash, a, nil)
	confDepthL1 := confdepth.NewConfDepth(10, func() eth.L1BlockRef { return b }, l1)
	s := newTestSelector(ctx, log, cfg, confDepthL1)

	_, err := s.FindL1Origin(ctx, l2Head)
	require.ErrorIs(t, err, ErrNextL1OriginRequired)
}

func u64ptr(n uint64) *uint64 {
	return &n
}

// TestOriginSelector_FjordSeqDrift has a similar setup to the previous test
// TestOriginSelectorStrictConfDepth but with Fjord activated at the l1 origin.
// This time the same L1 origin is returned if no new L1 head is seen, instead of an error,
// because the Fjord max sequencer drift is higher.
func TestOriginSelector_FjordSeqDrift(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	log := testlog.Logger(t, log.LevelCrit)
	cfg := &rollup.Config{
		MaxSequencerDrift: 8,
		BlockTime:         2,
		FjordTime:         u64ptr(20), // a's timestamp
	}
	l1 := &testutils.MockL1Source{}
	defer l1.AssertExpectations(t)
	a := eth.L1BlockRef{
		Hash:   common.Hash{'a'},
		Number: 10,
		Time:   20,
	}
	l2Head := eth.L2BlockRef{
		L1Origin: a.ID(),
		Time:     27, // next L2 block time would be past pre-Fjord seq drift
	}

	s := newTestSelector(ctx, log, cfg, l1)
	s.currentOrigin = a

	next, err := s.FindL1Origin(ctx, l2Head)
	require.NoError(t, err, "with Fjord activated, have increased max seq drift")
	require.Equal(t, a, next)
}

// TestOriginSelectorSeqDriftRespectsNextOriginTime
//
// There are 2 L1 blocks at time 20 & 100. The L2 Head is at time 27.
// The next L2 time is 29. Even though the next L2 time is past the seq
// drift, the origin should remain on block `a` because the next origin's
// time is greater than the next L2 time.
func TestOriginSelectorSeqDriftRespectsNextOriginTime(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	log := testlog.Logger(t, log.LevelCrit)
	cfg := &rollup.Config{
		MaxSequencerDrift: 8,
		BlockTime:         2,
	}
	l1 := &testutils.MockL1Source{}
	defer l1.AssertExpectations(t)
	a := eth.L1BlockRef{
		Hash:   common.Hash{'a'},
		Number: 10,
		Time:   20,
	}
	b := eth.L1BlockRef{
		Hash:       common.Hash{'b'},
		Number:     11,
		Time:       100,
		ParentHash: a.Hash,
	}
	l2Head := eth.L2BlockRef{
		L1Origin: a.ID(),
		Time:     27,
	}

	s := newTestSelector(ctx, log, cfg, l1)
	s.currentOrigin = a
	s.nextOrigin = b

	next, err := s.FindL1Origin(ctx, l2Head)
	require.Nil(t, err)
	require.Equal(t, a, next)
}

// TestOriginSelectorSeqDriftRespectsNextOriginTimeNoCache
//
// There are 2 L1 blocks at time 20 & 100. The L2 Head is at time 27.
// The next L2 time is 29. Even though the next L2 time is past the seq
// drift, the origin should remain on block `a` because the next origin's
// time is greater than the next L2 time.
// The L1OriginSelector does not have the next origin cached, and must fetch it
// because the max sequencer drift has been exceeded.
func TestOriginSelectorSeqDriftRespectsNextOriginTimeNoCache(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	log := testlog.Logger(t, log.LevelCrit)
	cfg := &rollup.Config{
		MaxSequencerDrift: 8,
		BlockTime:         2,
	}
	l1 := &testutils.MockL1Source{}
	defer l1.AssertExpectations(t)
	a := eth.L1BlockRef{
		Hash:   common.Hash{'a'},
		Number: 10,
		Time:   20,
	}
	b := eth.L1BlockRef{
		Hash:       common.Hash{'b'},
		Number:     11,
		Time:       100,
		ParentHash: a.Hash,
	}
	l2Head := eth.L2BlockRef{
		L1Origin: a.ID(),
		Time:     27,
	}

	l1.ExpectL1BlockRefByNumber(b.Number, b, nil)

	s := newTestSelector(ctx, log, cfg, l1)
	s.currentOrigin = a

	next, err := s.FindL1Origin(ctx, l2Head)
	require.Nil(t, err)
	require.Equal(t, a, next)
}

// TestOriginSelectorHandlesLateL1Blocks tests the forced repeat of the previous origin,
// but with a conf depth that first prevents it from learning about the need to repeat.
//
// There are 2 L1 blocks at time 20 & 100. The L2 Head is at time 27.
// The next L2 time is 29. Even though the next L2 time is past the seq
// drift, the origin should remain on block `a` because the next origin's
// time is greater than the next L2 time.
// Due to a conf depth of 2, block `b` is not immediately visible,
// and the origin selection should fail until it is visible, by waiting for block `c`.
func TestOriginSelectorHandlesLateL1Blocks(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	log := testlog.Logger(t, log.LevelCrit)
	cfg := &rollup.Config{
		MaxSequencerDrift: 8,
		BlockTime:         2,
	}
	l1 := &testutils.MockL1Source{}
	defer l1.AssertExpectations(t)
	a := eth.L1BlockRef{
		Hash:   common.Hash{'a'},
		Number: 10,
		Time:   20,
	}
	b := eth.L1BlockRef{
		Hash:       common.Hash{'b'},
		Number:     11,
		Time:       100,
		ParentHash: a.Hash,
	}
	c := eth.L1BlockRef{
		Hash:       common.Hash{'c'},
		Number:     12,
		Time:       150,
		ParentHash: b.Hash,
	}
	d := eth.L1BlockRef{
		Hash:       common.Hash{'d'},
		Number:     13,
		Time:       200,
		ParentHash: c.Hash,
	}
	l2Head := eth.L2BlockRef{
		L1Origin: a.ID(),
		Time:     27,
	}

	// l2 head does not change, so we start at the same origin again and again until we meet the conf depth
	l1.ExpectL1BlockRefByHash(a.Hash, a, nil)

	l1.ExpectL1BlockRefByNumber(b.Number, b, nil)

	l1Head := b
	confDepthL1 := confdepth.NewConfDepth(2, func() eth.L1BlockRef { return l1Head }, l1)
	s := newTestSelector(ctx, log, cfg, confDepthL1)

	_, err := s.FindL1Origin(ctx, l2Head)
	require.ErrorIs(t, err, ErrNextL1OriginRequired)

	l1Head = c
	_, err = s.FindL1Origin(ctx, l2Head)
	require.ErrorIs(t, err, ErrNextL1OriginRequired)

	l1Head = d
	next, err := s.FindL1Origin(ctx, l2Head)
	require.Nil(t, err)
	require.Equal(t, a, next, "must stay on a because the L1 time may not be higher than the L2 time")
}

// TestOriginSelectorMiscEvent ensures that the origin selector ignores miscellaneous events,
// but instead returns false to indicate that the event was not handled.
func TestOriginSelectorMiscEvent(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	log := testlog.Logger(t, log.LevelCrit)
	cfg := &rollup.Config{
		MaxSequencerDrift: 8,
		BlockTime:         2,
	}
	l1 := &testutils.MockL1Source{}
	defer l1.AssertExpectations(t)

	s := newTestSelector(ctx, log, cfg, l1)

	// This event is not handled
	handled := s.OnEvent(context.Background(), rollup.L1TemporaryErrorEvent{})
	require.False(t, handled)
}

func TestFindL1OriginOfNextL2Block(t *testing.T) {
	cfg := &rollup.Config{
		MaxSequencerDrift: 1800, // Use Fjord constant value
		BlockTime:         2,
	}

	los := newTestSelector(context.Background(), testlog.Logger(t, log.LevelDebug), cfg, &testutils.MockL1Source{})

	require.Panics(t, func() {
		_, _ = los.findL1OriginOfNextL2Block(
			eth.L2BlockRef{},
			eth.L1BlockRef{},
			eth.L1BlockRef{},
			false)
	})

	type testCase struct {
		name                string
		l2Head              eth.L2BlockRef
		currentL1Origin     eth.L1BlockRef
		nextL1Origin        eth.L1BlockRef
		matchAutoderivation bool
		expectedResult      eth.L1BlockRef
		expectedError       error
	}

	tcs := []testCase{}

	// Scenarios with valid data, no drift concerns
	// but availability of next l1 origin is modulated.
	//
	// L1 chain: a100(1200) <- [ a101(1212) ]
	//            /\
	// L2 chain    \_ b1000(1220)
	a100 := eth.L1BlockRef{
		Number: 100,
		Hash:   common.Hash{'a', '1', '0', '0'},
		Time:   1200,
	}
	a101 := eth.L1BlockRef{
		Number:     101,
		ParentHash: a100.Hash,
		Hash:       common.Hash{'a', '0', '0'},
		Time:       1212,
	}
	b1000 := eth.L2BlockRef{
		Number:   1000,
		Hash:     common.Hash{'b', '1', '0', '0', '0'},
		L1Origin: a100.ID(),
		Time:     1220,
	}

	tcs = append(tcs,
		testCase{
			name:            "normal operation, progress because we can",
			l2Head:          b1000,
			currentL1Origin: a100,
			nextL1Origin:    a101,
			expectedResult:  a101,
		},
		testCase{
			name:                "recover mode, progress because we can",
			l2Head:              b1000,
			currentL1Origin:     a100,
			nextL1Origin:        a101,
			expectedResult:      a101,
			matchAutoderivation: true,
		},
		testCase{
			name:            "normal operation, don't need to progress",
			l2Head:          b1000,
			currentL1Origin: a100,
			expectedResult:  a100,
		},
		testCase{
			name:                "recover mode, need to progress but can't",
			l2Head:              b1000,
			currentL1Origin:     a100,
			matchAutoderivation: true,
			expectedError:       ErrNextL1OriginRequired,
		},
	)

	// Bad input data / reorg scenarios
	// L1 chain: c100(1200) <-[x]- c101(1212)
	//            /\
	// L2 chain    \_[x]- d/e1000(1220)
	c100 := eth.L1BlockRef{
		Number: 100,
		Hash:   common.Hash{'a', '1', '0', '0'},
		Time:   1200,
	}
	c101 := eth.L1BlockRef{
		Number:     101,
		ParentHash: common.Hash{}, // does not point to c100
		Hash:       common.Hash{'a', '0', '0'},
		Time:       1212,
	}
	d1000 := eth.L2BlockRef{
		Number:   1000,
		L1Origin: c100.ID(),
		Hash:     common.Hash{'d', '1', '0', '0', '0'},
		Time:     1220,
	}
	e1000 := eth.L2BlockRef{
		Number:   1000,
		L1Origin: eth.BlockID{}, // does not point to c100
		Hash:     common.Hash{'d', '1', '0', '0', '0'},
		Time:     1220,
	}
	tcs = append(tcs,
		testCase{
			name:            "L1 reorg",
			currentL1Origin: c100,
			nextL1Origin:    c101,
			l2Head:          d1000,
			expectedResult:  c101,
			expectedError:   ErrNextL1OriginOrphaned,
		},
		testCase{
			name:            "Invalid l1 origin",
			currentL1Origin: c100,
			nextL1Origin:    c101,
			l2Head:          e1000,
			expectedResult:  c100,
			expectedError:   ErrInvalidL1Origin,
		})

	// Drift at maximum,
	// L1 chain: a100(1200) <- [ a101(1212) ]
	//            /\
	// L2 chain    \_ f1000(3000)
	f1000 := eth.L2BlockRef{
		Number:   1000,
		L1Origin: a100.ID(),
		Hash:     common.Hash{'f', '1', '0', '0', '0'},
		Time:     3000,
	}
	tcs = append(tcs,
		testCase{
			name:            "Drift at maximum, nextL1Origin available",
			currentL1Origin: a100,
			nextL1Origin:    a101,
			l2Head:          f1000,
			expectedResult:  a101,
		},
		testCase{
			name:            "Drift at maximum, nextL1Origin unavailable",
			currentL1Origin: a100,
			l2Head:          f1000,
			expectedError:   ErrNextL1OriginRequired,
		})

	// Negative drift,
	// L1 chain: a100(1200) <- a101(1212)
	//            /\
	// L2 chain    \_ g1000(1200)
	// Current drift is 0
	// adopting the nextLOrigin would make it negative (add 2 subtract 12)
	g1000 := eth.L2BlockRef{
		Number:   1000,
		L1Origin: a100.ID(),
		Hash:     common.Hash{'g', '1', '0', '0', '0'},
		Time:     1200,
	}
	tcs = append(tcs,
		testCase{
			name:            "Negative drift",
			currentL1Origin: a100,
			nextL1Origin:    a101,
			l2Head:          g1000,
			expectedResult:  a100,
		})

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			result, err := los.findL1OriginOfNextL2Block(
				tc.l2Head,
				tc.currentL1Origin,
				tc.nextL1Origin,
				tc.matchAutoderivation)
			if tc.expectedError != nil {
				require.ErrorIs(t, err, tc.expectedError)
			} else {
				require.NoError(t, err)
			}
			if result != tc.expectedResult {
				t.Errorf("expected result %v, got %v", tc.expectedResult, result)
			}
		})
	}
}

// simulatedL1 serves a canonical L1 chain with one block every 12s, hiding blocks
// fewer than confDepth blocks behind the head implied by now.
type simulatedL1 struct {
	// The selector only looks up block refs; anything else panics on the nil embed.
	L1Blocks
	now       uint64
	confDepth uint64
	// byHashLookups counts lookups by hash, which the selector only makes when its cache is reset.
	byHashLookups int
	// receiptsFetched records the blocks whose receipts were fetched.
	receiptsFetched map[common.Hash]bool
}

func (s *simulatedL1) FetchReceipts(_ context.Context, h common.Hash) (eth.BlockInfo, optypes.Receipts, error) {
	s.receiptsFetched[h] = true
	return nil, nil, nil
}

func simulatedL1Hash(n uint64) (h common.Hash) {
	binary.BigEndian.PutUint64(h[24:], n)
	return h
}

func simulatedL1Block(n uint64) eth.L1BlockRef {
	return eth.L1BlockRef{
		Hash:       simulatedL1Hash(n),
		Number:     n,
		Time:       n * 12,
		ParentHash: simulatedL1Hash(n - 1),
	}
}

func (s *simulatedL1) L1BlockRefByNumber(_ context.Context, n uint64) (eth.L1BlockRef, error) {
	if n+s.confDepth > s.now/12 {
		return eth.L1BlockRef{}, ethereum.NotFound
	}
	return simulatedL1Block(n), nil
}

func (s *simulatedL1) L1BlockRefByHash(_ context.Context, h common.Hash) (eth.L1BlockRef, error) {
	s.byHashLookups++
	return simulatedL1Block(binary.BigEndian.Uint64(h[24:])), nil
}

// TestOriginSelectorKeepsUpWithL1 ensures the L1 origin keeps pace with L1 however late the
// previous block's forkchoice update reaches the selector relative to the next build. A 10s
// block time falls behind L1 if the origin can only advance every other L2 block.
func TestOriginSelectorKeepsUpWithL1(t *testing.T) {
	const confDepth = 15
	// fcuDelay is the number of builds the forkchoice update of a block trails the build on
	// top of it by: -1 delivers it before that build, 2 models an event loop lagging behind.
	for _, fcuDelay := range []int{-1, 0, 2} {
		for _, blockTime := range []uint64{2, 10} {
			t.Run(fmt.Sprintf("fcuDelay=%d/blockTime=%d", fcuDelay, blockTime), func(t *testing.T) {
				ctx := context.Background()
				l1 := &simulatedL1{now: 10_000, confDepth: confDepth, receiptsFetched: make(map[common.Hash]bool)}
				cfg := &rollup.Config{BlockTime: blockTime, MaxSequencerDrift: 1800}
				s := NewL1OriginSelector(ctx, testlog.Logger(t, log.LevelCrit), cfg, l1).WithInlinePrefetch()

				l2Head := eth.L2BlockRef{Time: l1.now, L1Origin: simulatedL1Block(l1.now/12 - confDepth).ID()}
				initialOrigin := l2Head.L1Origin
				var built []eth.L2BlockRef
				var maxLag uint64
				// Long enough for a lagging origin to drift all the way to the limit.
				for l2Head.Time < 10_000+3*cfg.MaxSequencerDrift {
					l1.now = l2Head.Time
					if fcuDelay < 0 {
						s.OnEvent(ctx, engine.ForkchoiceUpdateEvent{UnsafeL2Head: l2Head})
					}
					origin, err := s.FindL1Origin(ctx, l2Head)
					require.NoError(t, err)
					if origin.ID() != initialOrigin {
						require.True(t, l1.receiptsFetched[origin.Hash], "receipts of origin %d were not prefetched", origin.Number)
					}
					built = append(built, l2Head)
					if fcuDelay >= 0 && len(built) > fcuDelay {
						s.OnEvent(ctx, engine.ForkchoiceUpdateEvent{UnsafeL2Head: built[len(built)-1-fcuDelay]})
					}
					l2Head = eth.L2BlockRef{Number: l2Head.Number + 1, Time: l2Head.Time + blockTime, L1Origin: origin.ID()}
					// Past the first drift window, a lagging origin would have drifted to the limit.
					if l2Head.Time > 10_000+cfg.MaxSequencerDrift {
						maxLag = max(maxLag, l2Head.Time/12-l2Head.L1Origin.Number)
					}
				}

				require.LessOrEqual(t, maxLag, uint64(confDepth+2),
					"L1 origin fell behind the sequencer confirmation depth")
				require.Equal(t, 1, l1.byHashLookups, "only the initial head should need a lookup by hash")
			})
		}
	}
}

// TestOriginSelectorIgnoresLateForkchoiceUpdate ensures that a forkchoice update for a head
// behind the current origin, which the event loop delivers after the sequencer has moved on,
// leaves the cached origins alone instead of resetting them.
func TestOriginSelectorIgnoresLateForkchoiceUpdate(t *testing.T) {
	ctx := context.Background()
	l1 := &testutils.MockL1Source{}
	defer l1.AssertExpectations(t)
	cfg := &rollup.Config{MaxSequencerDrift: 500, BlockTime: 10}
	a := eth.L1BlockRef{Hash: common.Hash{'a'}, Number: 10, Time: 20}
	b := eth.L1BlockRef{Hash: common.Hash{'b'}, Number: 11, Time: 32, ParentHash: a.Hash}
	c := eth.L1BlockRef{Hash: common.Hash{'c'}, Number: 12, Time: 44, ParentHash: b.Hash}
	d := eth.L1BlockRef{Hash: common.Hash{'d'}, Number: 13, Time: 56, ParentHash: c.Hash}

	s := newTestSelector(ctx, testlog.Logger(t, log.LevelCrit), cfg, l1)
	s.currentOrigin = b
	s.nextOrigin = c
	s.lookahead = d

	// The late update names a head on origin `a`. Neither a lookup nor a fetch may happen.
	handled := s.OnEvent(ctx, engine.ForkchoiceUpdateEvent{UnsafeL2Head: eth.L2BlockRef{L1Origin: a.ID(), Time: 40}})
	require.True(t, handled)

	origin, err := s.FindL1Origin(ctx, eth.L2BlockRef{L1Origin: b.ID(), Time: 50})
	require.NoError(t, err)
	require.Equal(t, c, origin)
	origin, err = s.FindL1Origin(ctx, eth.L2BlockRef{L1Origin: c.ID(), Time: 60})
	require.NoError(t, err)
	require.Equal(t, d, origin)

	// Had the unsafe head really been rewound to origin `a`, the build on top of it repairs
	// the cache by looking `a` up.
	l1.ExpectL1BlockRefByHash(a.Hash, a, nil)
	origin, err = s.FindL1Origin(ctx, eth.L2BlockRef{L1Origin: a.ID(), Time: 40})
	require.NoError(t, err)
	require.Equal(t, a, origin)
}

// TestOriginSelectorDropsFetchStraddlingReset ensures that a block fetched before a reset is
// not cached after it, since the reset may have abandoned the chain the block was fetched from.
func TestOriginSelectorDropsFetchStraddlingReset(t *testing.T) {
	ctx := context.Background()
	l1 := &testutils.MockL1Source{}
	defer l1.AssertExpectations(t)
	cfg := &rollup.Config{MaxSequencerDrift: 500, BlockTime: 10}
	a := eth.L1BlockRef{Hash: common.Hash{'a'}, Number: 10, Time: 20}
	orphan := eth.L1BlockRef{Hash: common.Hash{'b', '2'}, Number: 11, Time: 32, ParentHash: common.Hash{'a', '2'}}
	l2Head := eth.L2BlockRef{L1Origin: a.ID(), Time: 40}

	s := newTestSelector(ctx, testlog.Logger(t, log.LevelCrit), cfg, l1)
	s.currentOrigin = a

	// While the forkchoice update fetches the next origin, a reset happens and the build path
	// restores the current origin.
	l1.Mock.On("L1BlockRefByNumber", orphan.Number).Once().Run(func(mock.Arguments) {
		s.OnEvent(ctx, rollup.ResetEvent{})
		l1.ExpectL1BlockRefByHash(a.Hash, a, nil)
		_, _, err := s.CurrentAndNextOrigin(ctx, l2Head)
		require.NoError(t, err)
	}).Return(orphan, nil)
	l1.ExpectL1BlockRefByNumber(orphan.Number+1, eth.L1BlockRef{}, ethereum.NotFound)
	s.OnEvent(ctx, engine.ForkchoiceUpdateEvent{UnsafeL2Head: l2Head})

	_, next, err := s.CurrentAndNextOrigin(ctx, l2Head)
	require.NoError(t, err)
	require.Equal(t, eth.L1BlockRef{}, next)
}

// TestOriginSelectorPrefetchesInBackground ensures that the default prefetch runs off the
// caller's goroutine and still fills the cache.
func TestOriginSelectorPrefetchesInBackground(t *testing.T) {
	ctx := context.Background()
	l1 := &testutils.MockL1Source{}
	cfg := &rollup.Config{MaxSequencerDrift: 500, BlockTime: 2}
	a := eth.L1BlockRef{Hash: common.Hash{'a'}, Number: 10, Time: 20}
	b := eth.L1BlockRef{Hash: common.Hash{'b'}, Number: 11, Time: 22, ParentHash: a.Hash}
	c := eth.L1BlockRef{Hash: common.Hash{'c'}, Number: 12, Time: 24, ParentHash: b.Hash}

	s := NewL1OriginSelector(ctx, testlog.Logger(t, log.LevelCrit), cfg, l1)
	s.currentOrigin = a
	s.nextOrigin = b

	l1.ExpectL1BlockRefByNumber(c.Number, c, nil)
	l1.ExpectFetchReceipts(c.Hash, nil, nil, nil)
	origin, err := s.FindL1Origin(ctx, eth.L2BlockRef{L1Origin: a.ID(), Time: 24})
	require.NoError(t, err)
	require.Equal(t, b, origin)

	require.Eventually(t, func() bool {
		return s.isCached(c.Number) && l1.AssertNumberOfCalls(quietT{}, "FetchReceipts", 1)
	}, 5*time.Second, 10*time.Millisecond)
}

// quietT lets a mock assertion be polled without failing the test on early misses.
type quietT struct{}

func (quietT) Logf(string, ...any)   {}
func (quietT) Errorf(string, ...any) {}
func (quietT) FailNow()              {}

// TestOriginSelectorPrefetchesOnAdoption ensures that adopting the next origin fetches its
// successor and that successor's receipts, without waiting for a forkchoice update.
func TestOriginSelectorPrefetchesOnAdoption(t *testing.T) {
	ctx := context.Background()
	l1 := &testutils.MockL1Source{}
	defer l1.AssertExpectations(t)
	cfg := &rollup.Config{MaxSequencerDrift: 500, BlockTime: 10}
	a := eth.L1BlockRef{Hash: common.Hash{'a'}, Number: 10, Time: 20}
	b := eth.L1BlockRef{Hash: common.Hash{'b'}, Number: 11, Time: 32, ParentHash: a.Hash}
	c := eth.L1BlockRef{Hash: common.Hash{'c'}, Number: 12, Time: 44, ParentHash: b.Hash}

	s := NewL1OriginSelector(ctx, testlog.Logger(t, log.LevelCrit), cfg, l1).WithInlinePrefetch()
	s.currentOrigin = a
	s.nextOrigin = b

	l1.ExpectL1BlockRefByNumber(c.Number, c, nil)
	l1.ExpectFetchReceipts(c.Hash, nil, nil, nil)
	origin, err := s.FindL1Origin(ctx, eth.L2BlockRef{L1Origin: a.ID(), Time: 40})
	require.NoError(t, err)
	require.Equal(t, b, origin)

	// Adopting `c` in turn looks for its successor, which does not exist yet.
	l1.ExpectL1BlockRefByNumber(c.Number+1, eth.L1BlockRef{}, ethereum.NotFound)
	origin, err = s.FindL1Origin(ctx, eth.L2BlockRef{L1Origin: b.ID(), Time: 50})
	require.NoError(t, err)
	require.Equal(t, c, origin)
}

// TestOriginSelectorPrefetchesNextOriginReceipts ensures that caching a next origin on a
// forkchoice update also warms its receipts, which the first build on it needs.
func TestOriginSelectorPrefetchesNextOriginReceipts(t *testing.T) {
	ctx := context.Background()
	l1 := &testutils.MockL1Source{}
	defer l1.AssertExpectations(t)
	cfg := &rollup.Config{MaxSequencerDrift: 500, BlockTime: 10}
	a := eth.L1BlockRef{Hash: common.Hash{'a'}, Number: 10, Time: 20}
	b := eth.L1BlockRef{Hash: common.Hash{'b'}, Number: 11, Time: 32, ParentHash: a.Hash}

	s := NewL1OriginSelector(ctx, testlog.Logger(t, log.LevelCrit), cfg, l1).WithInlinePrefetch()
	s.currentOrigin = a

	l1.ExpectL1BlockRefByNumber(b.Number, b, nil)
	l1.ExpectFetchReceipts(b.Hash, nil, nil, nil)
	l1.ExpectL1BlockRefByNumber(b.Number+1, eth.L1BlockRef{}, ethereum.NotFound)
	handled := s.OnEvent(ctx, engine.ForkchoiceUpdateEvent{UnsafeL2Head: eth.L2BlockRef{L1Origin: a.ID(), Time: 40}})
	require.True(t, handled)
}

// newTestSelector returns a selector that drops background prefetches, so that tests with
// strict L1 mocks only see the fetches they set up. Tests of prefetching run them inline.
func newTestSelector(ctx context.Context, log log.Logger, cfg *rollup.Config, l1 L1Blocks) *L1OriginSelector {
	s := NewL1OriginSelector(ctx, log, cfg, l1)
	s.runPrefetch = func(func()) {}
	return s
}
