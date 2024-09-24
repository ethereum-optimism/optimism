package sequencing

import (
	"context"
	"errors"
	"testing"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/confdepth"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
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

// TestOriginSelectorAdvances ensures that the origin selector
// advances the origin
//
// There are 2 L1 blocks at time 20 & 25. The L2 Head is at time 24.
// The next L2 time is 26 which is after the next L1 block time. There
// is no conf depth to stop the origin selection so block `b` should
// be the next L1 origin
func TestOriginSelectorAdvances(t *testing.T) {
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

	c := make(chan eth.L1BlockRef, 1)
	next, err := s.findL1Origin(ctx, l2Head, c)
	require.Nil(t, err)
	require.Equal(t, b, next)

	// Wait for the origin selector's background fetch to finish.
	// This fetch should not be triggered because the next origin is already known.
	select {
	case _, ok := <-c:
		require.False(t, ok)
	default:
		t.Fatal("expected the background fetch to have not run")
	}
}

// TestOriginSelectorNextOrigin ensures that the origin selector
// handles the case where the L2 Head is based on the internal next origin.
//
// There are 2 L1 blocks at time 20 & 25. The L2 Head is at time 24.
// The next L2 time is 26 which is after the next L1 block time. There
// is no conf depth to stop the origin selection so block `b` should
// be the next L1 origin
func TestOriginSelectorAdvancesFromCache(t *testing.T) {
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
	s.nextOrigin = a

	c := make(chan eth.L1BlockRef, 1)
	next, err := s.findL1Origin(ctx, l2Head, c)
	require.Nil(t, err)
	require.Equal(t, a, next)

	// Wait for the origin selector's background fetch to finish.
	// This fetch should be triggered because the next origin is not already known.
	next, ok := <-c
	require.True(t, ok)
	require.Equal(t, b, next)
}

// TestOriginSelectorPrefetchesNextOrigin ensures that the origin selector
// prefetches the next origin when it can.
//
// The next L2 time is 26 which is after the next L1 block time. There
// is no conf depth to stop the origin selection so block `b` will
// be the next L1 origin as soon as it is fetched.
func TestOriginSelectorPrefetchesNextOrigin(t *testing.T) {
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

	c := make(chan eth.L1BlockRef, 1)
	next, err := s.findL1Origin(ctx, l2Head, c)
	require.Nil(t, err)
	require.Equal(t, a, next)

	// Wait for the origin selector's background fetch to finish.
	// This fetch should be triggered because the next origin is not already known.
	next, ok := <-c
	require.True(t, ok)
	require.Equal(t, b, next)

	// The next origin should be `b` now.
	c = make(chan eth.L1BlockRef, 1)
	next, err = s.findL1Origin(ctx, l2Head, c)
	require.Nil(t, err)
	require.Equal(t, b, next)

	// Wait for the origin selector's background fetch to finish.
	// This fetch should not be triggered because the next origin is already known.
	select {
	case _, ok := <-c:
		require.False(t, ok)
	default:
		t.Fatal("expected the background fetch to have not run")
	}
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

	c := make(chan eth.L1BlockRef, 1)
	next, err := s.findL1Origin(ctx, l2Head, c)
	require.Nil(t, err)
	require.Equal(t, a, next)

	// Wait for the origin selector's background fetch to finish.
	// This fetch should not be triggered because the next origin is already known.
	select {
	case _, ok := <-c:
		require.False(t, ok)
	default:
		t.Fatal("expected the background fetch to have not run")
	}
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

	c := make(chan eth.L1BlockRef, 1)
	next, err := s.findL1Origin(ctx, l2Head, c)
	require.NoError(t, err)
	require.Equal(t, b, next)

	// Wait for the origin selector's background fetch to finish.
	// This fetch should already be completed because findL1Origin would have waited for it.
	select {
	case _, ok := <-c:
		require.False(t, ok)
	default:
		t.Fatal("expected the background fetch to have already completed")
	}
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

	// l1.ExpectL1BlockRefByHash(a.Hash, a, nil)
	confDepthL1 := confdepth.NewConfDepth(10, func() eth.L1BlockRef { return b }, l1)
	s := NewL1OriginSelector(ctx, log, cfg, confDepthL1)
	s.currentOrigin = a

	c := make(chan eth.L1BlockRef, 1)
	next, err := s.findL1Origin(ctx, l2Head, c)
	require.Nil(t, err)
	require.Equal(t, a, next)

	// Wait for the origin selector's background fetch to finish.
	// This fetch should not return a new origin because the conf depth has not been met.
	_, ok := <-c
	require.False(t, ok)
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

	c := make(chan eth.L1BlockRef, 1)
	_, err := s.findL1Origin(ctx, l2Head, c)
	require.ErrorContains(t, err, "sequencer time drift")

	// Wait for the origin selector's background fetch to finish.
	// This fetch should already be completed because findL1Origin would have waited for it.
	select {
	case _, ok := <-c:
		require.False(t, ok)
	default:
		t.Fatal("expected the background fetch to have already completed")
	}
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
	b := eth.L1BlockRef{
		Hash:   common.Hash{'b'},
		Number: 11,
		Time:   22,
	}
	l2Head := eth.L2BlockRef{
		L1Origin: a.ID(),
		Time:     27, // next L2 block time would be past pre-Fjord seq drift
	}

	// This is called as part of the background prefetch job
	l1.ExpectL1BlockRefByNumber(a.Number+1, b, nil)

	s := NewL1OriginSelector(ctx, log, cfg, l1)
	s.currentOrigin = a

	c := make(chan eth.L1BlockRef, 1)
	l1O, err := s.findL1Origin(ctx, l2Head, c)
	require.NoError(t, err, "with Fjord activated, have increased max seq drift")
	require.Equal(t, a, l1O)

	// Wait for the origin selector's background fetch to finish.
	// This fetch should be triggered because the next origin is not already known.
	next, ok := <-c
	require.True(t, ok)
	require.Equal(t, b, next)
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

	c := make(chan eth.L1BlockRef, 1)
	next, err := s.findL1Origin(ctx, l2Head, c)
	require.Nil(t, err)
	require.Equal(t, a, next)

	// Wait for the origin selector's background fetch to finish.
	// This fetch should not be triggered because the next origin is already known.
	select {
	case _, ok := <-c:
		require.False(t, ok)
	default:
		t.Fatal("expected the background fetch to have not run")
	}
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

	c := make(chan eth.L1BlockRef, 1)
	next, err := s.findL1Origin(ctx, l2Head, c)
	require.Nil(t, err)
	require.Equal(t, a, next)

	// Wait for the origin selector's background fetch to finish.
	// This fetch should already be completed because findL1Origin would have waited for it.
	select {
	case _, ok := <-c:
		require.False(t, ok)
	default:
		t.Fatal("expected the background fetch to have already completed")
	}
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

	ch := make(chan eth.L1BlockRef, 1)
	_, err := s.findL1Origin(ctx, l2Head, ch)
	require.ErrorContains(t, err, "sequencer time drift")

	// Wait for the origin selector's background fetch to finish.
	// This fetch should already be completed because findL1Origin would have waited for it.
	select {
	case _, ok := <-ch:
		require.False(t, ok)
	default:
		t.Fatal("expected the background fetch to have already completed")
	}

	l1Head = c
	ch = make(chan eth.L1BlockRef, 1)
	_, err = s.findL1Origin(ctx, l2Head, ch)
	require.ErrorContains(t, err, "sequencer time drift")

	// Wait for the origin selector's background fetch to finish.
	// This fetch should already be completed because findL1Origin would have waited for it.
	select {
	case _, ok := <-ch:
		require.False(t, ok)
	default:
		t.Fatal("expected the background fetch to have already completed")
	}

	l1Head = d
	ch = make(chan eth.L1BlockRef, 1)
	next, err := s.findL1Origin(ctx, l2Head, ch)
	require.Nil(t, err)
	require.Equal(t, a, next, "must stay on a because the L1 time may not be higher than the L2 time")

	// Wait for the origin selector's background fetch to finish.
	// This fetch should already be completed because findL1Origin would have waited for it.
	select {
	case _, ok := <-ch:
		require.False(t, ok)
	default:
		t.Fatal("expected the background fetch to have already completed")
	}
}
