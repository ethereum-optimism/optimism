package sequencing

import (
	"context"
	"errors"
	"testing"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/confdepth"
	"github.com/ethereum-optimism/optimism/op-node/rollup/engine"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
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

	s := NewL1OriginSelector(ctx, log, cfg, l1)

	_, err := s.FindL1Origin(ctx, l2Head)
	require.ErrorContains(t, err, "test error")

	// The same outcome occurs when the cached origin is different from that of the L2 head.
	l1.ExpectL1BlockRefByHash(a.Hash, eth.L1BlockRef{}, errors.New("test error"))

	s = NewL1OriginSelector(ctx, log, cfg, l1)
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

	s := NewL1OriginSelector(ctx, log, cfg, l1)
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

		s := NewL1OriginSelector(ctx, log, cfg, l1)

		requireL1OriginAt := func(l2Head eth.L2BlockRef, want eth.L1BlockRef) {
			got, err := s.FindL1Origin(ctx, l2Head)
			require.Nil(t, err)
			require.Equal(t, want, got)
		}

		s.currentOrigin = a
		s.nextOrigin = b

		// Trigger the background fetch via a forkchoice update.
		// This should be a no-op because the next origin is already cached.
		handled := s.OnEvent(context.Background(), engine.ForkchoiceUpdateEvent{UnsafeL2Head: l2Head})
		require.True(t, handled)

		requireL1OriginAt(l2Head, b)

		l2Head = eth.L2BlockRef{
			L1Origin: b.ID(),
			Time:     26,
		}

		// The origin is still `b` because the next origin has not been fetched yet.
		requireL1OriginAt(l2Head, b)

		l1.ExpectL1BlockRefByNumber(c.Number, c, nil)

		// Trigger the background fetch via a forkchoice update.
		// This will actually fetch the next origin because the internal cache is empty.
		handled = s.OnEvent(context.Background(), engine.ForkchoiceUpdateEvent{UnsafeL2Head: l2Head})
		require.True(t, handled)

		// The next origin should be `c` now.
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

	s := NewL1OriginSelector(ctx, log, cfg, l1)
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

	// This is called as part of the background prefetch job
	l1.ExpectL1BlockRefByNumber(b.Number, b, nil)

	s := NewL1OriginSelector(ctx, log, cfg, l1)
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

	// This is called as part of the background prefetch job
	l1.ExpectL1BlockRefByNumber(b.Number, b, nil)

	s := NewL1OriginSelector(ctx, log, cfg, l1)
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

	// A reorg happens and `b` is replaced by a block with a different hash
	c := eth.L1BlockRef{
		Hash:       common.Hash{'c'},
		Number:     12,
		Time:       24,
		ParentHash: common.Hash{'b', '2'},
	}
	l1.ExpectL1BlockRefByNumber(c.Number, c, nil)
	l2Head = eth.L2BlockRef{
		L1Origin: b.ID(),
		Time:     26,
	}

	// Trigger the background fetch via a forkchoice update
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

	s := NewL1OriginSelector(ctx, log, cfg, l1)
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

	s := NewL1OriginSelector(ctx, log, cfg, l1)

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
	s := NewL1OriginSelector(ctx, log, cfg, confDepthL1)
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
	s := NewL1OriginSelector(ctx, log, cfg, confDepthL1)

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

	s := NewL1OriginSelector(ctx, log, cfg, l1)
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

	s := NewL1OriginSelector(ctx, log, cfg, l1)
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

	s := NewL1OriginSelector(ctx, log, cfg, l1)
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
	s := NewL1OriginSelector(ctx, log, cfg, confDepthL1)

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

	s := NewL1OriginSelector(ctx, log, cfg, l1)

	// This event is not handled
	handled := s.OnEvent(context.Background(), rollup.L1TemporaryErrorEvent{})
	require.False(t, handled)
}

func TestFindL1OriginOfNextL2Block(t *testing.T) {
	cfg := &rollup.Config{
		MaxSequencerDrift: 1800, // Use Fjord constant value
		BlockTime:         2,
	}

	los := NewL1OriginSelector(context.Background(), testlog.Logger(t, log.LevelDebug), cfg, &testutils.MockL1Source{})

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
