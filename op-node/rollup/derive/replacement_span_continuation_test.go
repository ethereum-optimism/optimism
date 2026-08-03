package derive

import (
	"context"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
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
