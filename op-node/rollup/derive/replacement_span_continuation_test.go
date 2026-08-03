package derive

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
)

// flushCountingProvider records FlushChannel calls so a test can assert whether
// the containing channel was discarded.
type flushCountingProvider struct {
	flushes int
}

func (p *flushCountingProvider) Reset(context.Context, eth.L1BlockRef, eth.SystemConfig) error {
	return nil
}
func (p *flushCountingProvider) FlushChannel()          { p.flushes++ }
func (p *flushCountingProvider) Origin() eth.L1BlockRef { return eth.L1BlockRef{} }
func (p *flushCountingProvider) NextBatch(context.Context, eth.L2BlockRef) (*SingularBatch, bool, error) {
	return nil, false, nil
}

// TestDepositsOnlyReplacementChannelFlush pins the lineage-selection switch:
// whether producing a deposits-only replacement discards the remainder of the
// channel that contained the replaced block.
//
// Flushing hands the lineage to whichever channel comes next on L1, which is how
// a from-genesis node lands on an abandoned lineage (devnet interop-reorg-5,
// chain 420120192: replaced 13460 -> channel A flushed -> channel B's 13461
// adopted). Keeping it re-applies the span's tail onto the replacement, which is
// the lineage the batcher continued.
func TestDepositsOnlyReplacementChannelFlush(t *testing.T) {
	cfg := &rollup.Config{BlockTime: 1}
	parent := eth.L2BlockRef{Number: 13459, Hash: testHash(0xc1)}
	derivedFrom := eth.L1BlockRef{Number: 11391112, Hash: testHash(0x25)}

	newQueue := func(t *testing.T, opts ...AttributesQueueOption) (*AttributesQueue, *flushCountingProvider) {
		prev := &flushCountingProvider{}
		aq := NewAttributesQueue(testlog.Logger(t, log.LevelInfo), cfg, nil, prev, opts...)
		// A replacement is only requested for attributes the queue just produced,
		// so seed the state the real pipeline would be in.
		aq.lastAttribs = &AttributesWithParent{
			Attributes:  &eth.PayloadAttributes{Transactions: []eth.Data{{0x7e}, {0x02}}},
			Parent:      parent,
			DerivedFrom: derivedFrom,
		}
		return aq, prev
	}

	t.Run("flushes by default", func(t *testing.T) {
		aq, prev := newQueue(t)
		attrs, err := aq.DepositsOnlyAttributes(parent.ID(), derivedFrom)
		require.NoError(t, err)
		require.True(t, attrs.Attributes.IsDepositsOnly())
		require.Equal(t, 1, prev.flushes, "default behavior must discard the containing channel")
	})

	t.Run("keeps channel when continuing the span", func(t *testing.T) {
		aq, prev := newQueue(t, WithReplacementContinuesSpan(true))
		attrs, err := aq.DepositsOnlyAttributes(parent.ID(), derivedFrom)
		require.NoError(t, err)
		require.True(t, attrs.Attributes.IsDepositsOnly(),
			"the replacement itself must still be deposits-only")
		require.Zero(t, prev.flushes,
			"the containing channel must stay buffered so the span tail re-applies")
	})
}

func testHash(b byte) (h common.Hash) {
	h[0] = b
	return
}

// TestAnchorBetweenReplacementsWedges pins the case that discriminates the
// finality-pinned-anchor model (interop-reorg-5 investigation, update 9): a walk
// anchored BETWEEN two replacements.
//
// On the live cluster the reset anchor landed exactly ON the first replacement,
// because finality was pinned there and the replacement is the last block the
// pre- and post-invalidation branches share. That anchor past-skips the denied
// block, so the deny never fires, nothing is flushed, and the containing span's
// tail re-applies — which is why the live node ended on the batcher's lineage.
//
// Had finality been further along — above the first replacement but below the
// second — the walk would have re-proposed the second denied block, flushed its
// channel, and then found nothing on L1 that chains onto the resulting
// replacement: the later re-batch is built on the other lineage. That is a wedge
// on a node that never wiped anything, and it is the prediction this test checks.
//
// Layout (numbers scaled down from 13460 / 14245):
//
//	4      common ancestor
//	5 = R1 first replacement (already applied)
//	6      anchor — above R1, below the second denied block
//	7      denied on this lineage; replaced by R2
//	8,9    the containing span's tail
//	       re-batch span starts at 8 but is built on the OTHER lineage
func TestAnchorBetweenReplacementsWedges(t *testing.T) {
	zero := uint64(0)
	cfg := &rollup.Config{
		Genesis:           rollup.Genesis{L2Time: 1000},
		BlockTime:         1,
		SeqWindowSize:     3600,
		MaxSequencerDrift: 600,
		DeltaTime:         &zero,
		HoloceneTime:      &zero,
	}
	chainID := big.NewInt(420)
	logger := testlog.Logger(t, log.LevelError)
	ctx := context.Background()

	l1A := eth.L1BlockRef{Hash: common.HexToHash("0xaaaa01"), Number: 100, Time: 1002}
	l1B := eth.L1BlockRef{Hash: common.HexToHash("0xaaaa02"), Number: 101, ParentHash: l1A.Hash, Time: 1014}
	l1Blocks := []eth.L1BlockRef{l1A, l1B}

	r1 := eth.L2BlockRef{ // 5: the first replacement, on both branches
		Hash: testHash(0xd1), Number: 5, Time: 1005,
		L1Origin: eth.BlockID{Hash: l1A.Hash, Number: l1A.Number},
	}
	anchor := eth.L2BlockRef{ // 6: between the replacements
		Hash: testHash(0xf6), Number: 6, ParentHash: r1.Hash, Time: 1006,
		L1Origin: eth.BlockID{Hash: l1A.Hash, Number: l1A.Number},
	}
	r2 := eth.L2BlockRef{ // 7': deposits-only replacement of the denied block 7
		Hash: testHash(0xd2), Number: 7, ParentHash: anchor.Hash, Time: 1007,
		L1Origin: eth.BlockID{Hash: l1A.Hash, Number: l1A.Number},
	}

	l2Client := testutils.MockL2Client{}
	var nilErr error
	l2Client.Mock.On("L2BlockRefByNumber", r1.Number).Return(r1, &nilErr)
	l2Client.Mock.On("L2BlockRefByNumber", anchor.Number).Return(anchor, &nilErr)

	// The channel containing the denied block: 6..9, parent_check = hash(5).
	var containing []*SingularBatch
	for i := uint64(6); i <= 9; i++ {
		containing = append(containing, &SingularBatch{
			ParentHash: r1.Hash, EpochNum: rollup.Epoch(l1A.Number), EpochHash: l1A.Hash,
			Timestamp: 1000 + i,
		})
	}
	containingSpan := initializedSpanBatch(containing, cfg.Genesis.L2Time, chainID)

	// The later re-batch, built on the OTHER lineage: starts at 8, and its
	// parent_check names that lineage's block 7, which this node does not have.
	otherLineage7 := testHash(0xa7)
	var rebatch []*SingularBatch
	for i := uint64(8); i <= 9; i++ {
		rebatch = append(rebatch, &SingularBatch{
			ParentHash: otherLineage7, EpochNum: rollup.Epoch(l1A.Number), EpochHash: l1A.Hash,
			Timestamp: 1000 + i,
		})
	}
	rebatchSpan := initializedSpanBatch(rebatch, cfg.Genesis.L2Time, chainID)

	t.Run("anchor between replacements re-proposes the denied block", func(t *testing.T) {
		validity, _ := checkSpanBatchPrefix(ctx, cfg, logger, l1Blocks, anchor, containingSpan, l1B, &l2Client)
		require.EqualValues(t, BatchAccept, validity)

		got, err := containingSpan.GetSingularBatches(l1Blocks, anchor)
		require.NoError(t, err)
		var ts []uint64
		for _, sb := range got {
			ts = append(ts, sb.Timestamp)
		}
		require.Equal(t, []uint64{1007, 1008, 1009}, ts,
			"walking from between the replacements must re-propose block 7, triggering the deny")
	})

	t.Run("nothing on L1 chains onto the replacement once the channel is flushed", func(t *testing.T) {
		// Today's behavior: the deny-triggered replacement flushes the containing
		// channel, so the re-batch is the next candidate — and it is built on the
		// other lineage, so it drops. With no further batches this is the wedge.
		validity, _ := checkSpanBatchPrefix(ctx, cfg, logger, l1Blocks, r2, rebatchSpan, l1B, &l2Client)
		require.EqualValues(t, BatchDrop, validity,
			"the re-batch must not chain onto the replacement: this is the permanent wedge")
	})

	t.Run("keeping the channel lets the span tail continue on the replacement", func(t *testing.T) {
		// With --rollup.replacement-continues-span the containing channel is still
		// buffered, so its tail is extracted against the replacement instead.
		validity, _ := checkSpanBatchPrefix(ctx, cfg, logger, l1Blocks, r2, containingSpan, l1B, &l2Client)
		require.EqualValues(t, BatchAccept, validity)

		got, err := containingSpan.GetSingularBatches(l1Blocks, r2)
		require.NoError(t, err)
		var ts []uint64
		for _, sb := range got {
			ts = append(ts, sb.Timestamp)
		}
		require.Equal(t, []uint64{1008, 1009}, ts,
			"the replaced block is past-skipped and the tail re-applies on the replacement")
	})
}
