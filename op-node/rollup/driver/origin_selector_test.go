package driver

import (
	"context"
	"testing"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/testlog"
	"github.com/ethereum-optimism/optimism/op-node/testutils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

// TestOriginSelectorAdvances ensures that the origin selector
// advances the origin
//
// There are 2 L1 blocks at time 20 & 25. The L2 Head is at time 24.
// The next L2 time is 26 which is after the next L1 block time. There
// is no conf depth to stop the origin selection so block `b` should
// be the next L1 origin
func TestOriginSelectorAdvances(t *testing.T) {
	log := testlog.Logger(t, log.LvlCrit)
	cfg := &rollup.Config{
		MaxSequencerDrift: 500,
		BlockTime:         2,
	}
	l1 := &testutils.MockL1Source{}
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

	l1.ExpectL1BlockRefByHash(a.Hash, a, nil)
	l1.ExpectL1BlockRefByNumber(b.Number, b, nil)

	s := NewL1OriginSelector(log, cfg, l1, 0)

	next, err := s.FindL1Origin(context.Background(), b, l2Head)
	require.Nil(t, err)
	require.Equal(t, b, next)
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
	log := testlog.Logger(t, log.LvlCrit)
	cfg := &rollup.Config{
		MaxSequencerDrift: 500,
		BlockTime:         2,
	}
	l1 := &testutils.MockL1Source{}
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

	l1.ExpectL1BlockRefByHash(a.Hash, a, nil)
	l1.ExpectL1BlockRefByNumber(b.Number, b, nil)

	s := NewL1OriginSelector(log, cfg, l1, 0)

	next, err := s.FindL1Origin(context.Background(), b, l2Head)
	require.Nil(t, err)
	require.Equal(t, a, next)
}

// TestOriginSelectorRespectsConfDepth ensures that the origin selector
// will respects the confirmation depth requirement
//
// There are 2 L1 blocks at time 20 & 25. The L2 Head is at time 27.
// The next L2 time is 29 which enough to normally select block `b`
// as the origin, however block `b` is the L1 Head & the sequencer
// needs to wait until that block is confirmed enough before advancing.
func TestOriginSelectorRespectsConfDepth(t *testing.T) {
	log := testlog.Logger(t, log.LvlCrit)
	cfg := &rollup.Config{
		MaxSequencerDrift: 500,
		BlockTime:         2,
	}
	l1 := &testutils.MockL1Source{}
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

	s := NewL1OriginSelector(log, cfg, l1, 10)

	next, err := s.FindL1Origin(context.Background(), b, l2Head)
	require.Nil(t, err)
	require.Equal(t, a, next)
}

// TestOriginSelectorRespectsMaxSeqDrift ensures that the origin selector
// will advance if the time delta between the current L1 origin and the next
// L2 block is greater than the sequencer drift. This needs to occur even
// if conf depth needs to be ignored
//
// There are 2 L1 blocks at time 20 & 25. The L2 Head is at time 27.
// The next L2 time is 29. The sequencer drift is 8 so the L2 head is
// valid with origin `a`, but the next L2 block is not valid with origin `b.`
// This is because 29 (next L2 time) > 20 (origin) + 8 (seq drift) => invalid block.
// Even though the LOS would normally refuse to advance because block `b` does not
// have enough confirmations, it should in this instance.
func TestOriginSelectorRespectsMaxSeqDrift(t *testing.T) {
	log := testlog.Logger(t, log.LvlCrit)
	cfg := &rollup.Config{
		MaxSequencerDrift: 8,
		BlockTime:         2,
	}
	l1 := &testutils.MockL1Source{}
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

	s := NewL1OriginSelector(log, cfg, l1, 10)

	next, err := s.FindL1Origin(context.Background(), b, l2Head)
	require.Nil(t, err)
	require.Equal(t, b, next)
}

// TestOriginSelectorSeqDriftRespectsNextOriginTime
//
// There are 2 L1 blocks at time 20 & 100. The L2 Head is at time 27.
// The next L2 time is 29. Even though the next L2 time is past the seq
// drift, the origin should remain on block `a` because the next origin's
// time is greater than the next L2 time.
func TestOriginSelectorSeqDriftRespectsNextOriginTime(t *testing.T) {
	log := testlog.Logger(t, log.LvlCrit)
	cfg := &rollup.Config{
		MaxSequencerDrift: 8,
		BlockTime:         2,
	}
	l1 := &testutils.MockL1Source{}
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

	l1.ExpectL1BlockRefByHash(a.Hash, a, nil)
	l1.ExpectL1BlockRefByNumber(b.Number, b, nil)

	s := NewL1OriginSelector(log, cfg, l1, 10)

	next, err := s.FindL1Origin(context.Background(), b, l2Head)
	require.Nil(t, err)
	require.Equal(t, a, next)
}
