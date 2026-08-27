package derive

import (
	"bytes"
	"context"
	"errors"
	"io"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-node/metrics"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/ptr"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

// multiBlockChannelBatches builds a run of singular batches with the given timestamps, all on one
// L1 origin, alongside a rollup config whose multi-blocks activation is `activation`.
func multiBlockChannelBatches(t *testing.T, timestamps []uint64, activation uint64) (rollup.Config, []*SingularBatch, eth.L1BlockRef) {
	t.Helper()
	cfg := rollupCfg
	cfg.MultiBlockTime = ptr.New(activation)
	cfg.MaxMultiBlocks = ptr.New(uint64(4))

	l1Origin := eth.L1BlockRef{Number: cfg.Genesis.L1.Number + 42_000, Hash: common.Hash{0xde, 0xad, 0x42}}
	rng := rand.New(rand.NewSource(0x9182736))
	batches := make([]*SingularBatch, 0, len(timestamps))
	for _, ts := range timestamps {
		b := RandomSingularBatch(rng, 1, cfg.L2ChainID)
		b.Timestamp = ts
		b.EpochNum = rollup.Epoch(l1Origin.Number)
		b.EpochHash = l1Origin.Hash
		batches = append(batches, b)
	}
	return cfg, batches, l1Origin
}

// readSpanBatches closes the channel out, reads it back as a single frame and returns the wire type
// and derived span batch of every span in it.
func readSpanBatches(t *testing.T, cfg *rollup.Config, cout *SpanChannelOut, l1Origin eth.L1BlockRef) ([]uint8, []*SpanBatch) {
	t.Helper()
	require.NoError(t, cout.Close())

	var frameBuf bytes.Buffer
	_, err := cout.OutputFrame(&frameBuf, 100_000)
	require.ErrorIs(t, err, io.EOF)

	var frame Frame
	require.NoError(t, frame.UnmarshalBinary(&frameBuf))
	require.True(t, frame.IsLast)
	ch := NewChannel(frame.ID, l1Origin, false)
	require.NoError(t, ch.AddFrame(frame, l1Origin))
	require.True(t, ch.IsReady())

	spec := rollup.NewChainSpec(cfg)
	br, err := BatchReader(ch.Reader(), spec.MaxRLPBytesPerChannel(0), true)
	require.NoError(t, err)

	var types []uint8
	var spans []*SpanBatch
	for {
		bd, err := br()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
		types = append(types, bd.GetBatchType())
		sb, err := DeriveSpanBatch(bd, cfg)
		require.NoError(t, err)
		spans = append(spans, sb)
	}
	return types, spans
}

func spanTimestamps(spans []*SpanBatch) []uint64 {
	var out []uint64
	for _, sb := range spans {
		for _, elem := range sb.Batches {
			out = append(out, elem.Timestamp)
		}
	}
	return out
}

// TestSpanChannelOut_MultiBlockActivation checks that the batcher switches wire versions at the
// multi-blocks activation, sealing the open v1 span rather than growing it into a v2 span.
func TestSpanChannelOut_MultiBlockActivation(t *testing.T) {
	base := rollupCfg.Genesis.L2Time + 420_000
	activation := base + 2*rollupCfg.BlockTime
	// two blocks before activation, then the activation block, then a group of two, then one more
	timestamps := []uint64{base, base + 2, activation, activation + 2, activation + 2, activation + 4}
	cfg, batches, l1Origin := multiBlockChannelBatches(t, timestamps, activation)

	cout, err := NewSpanChannelOut(100_000, Zlib, rollup.NewChainSpec(&cfg))
	require.NoError(t, err)
	for i, b := range batches {
		require.NoError(t, cout.addSingularBatch(b, uint64(i)))
	}

	types, spans := readSpanBatches(t, &cfg, cout, l1Origin)
	require.Equal(t, []uint8{SpanBatchType, SpanBatchV2Type}, types,
		"the pre-activation blocks must stay in a v1 span")
	require.Equal(t, timestamps[:2], spanTimestamps(spans[:1]))
	require.Equal(t, timestamps[2:], spanTimestamps(spans[1:]))
	require.Equal(t, uint(0), spans[1].sameTsBits.Bit(0), "the activation block is not a sibling")
	require.Equal(t, uint(1), spans[1].sameTsBits.Bit(2))
}

// TestSpanChannelOut_MultiBlockSpanBoundaryInsideGroup checks that a span boundary falling in the
// middle of a group of siblings is expressed by setting bit 0 of the next span, so the pair still
// decodes to the original block sequence.
func TestSpanChannelOut_MultiBlockSpanBoundaryInsideGroup(t *testing.T) {
	base := rollupCfg.Genesis.L2Time + 420_000
	activation := base - rollupCfg.BlockTime
	// a group of three, split by the two-blocks-per-span limit after the second block
	timestamps := []uint64{base, base, base, base + 2, base + 2, base + 4}
	cfg, batches, l1Origin := multiBlockChannelBatches(t, timestamps, activation)

	cout, err := NewSpanChannelOut(100_000, Zlib, rollup.NewChainSpec(&cfg), WithMaxBlocksPerSpanBatch(2))
	require.NoError(t, err)
	for i, b := range batches {
		require.NoError(t, cout.addSingularBatch(b, uint64(i)))
	}

	types, spans := readSpanBatches(t, &cfg, cout, l1Origin)
	require.Equal(t, []uint8{SpanBatchV2Type, SpanBatchV2Type, SpanBatchV2Type}, types)
	require.Equal(t, timestamps, spanTimestamps(spans))
	// spans hold [base, base], [base, base+2] and [base+2, base+4]: both boundaries fall inside a
	// group, so both later spans start on a sibling of their parent
	require.Equal(t, uint(0), spans[0].sameTsBits.Bit(0))
	require.Equal(t, uint(1), spans[1].sameTsBits.Bit(0))
	require.Equal(t, uint(1), spans[2].sameTsBits.Bit(0))
}

// TestSpanChannelOut_MultiBlockChannelBoundaryInsideGroup checks that a channel filling up in the
// middle of a group of siblings is expressed by setting bit 0 of the follow-on channel's first
// span, so the two channels still decode to the original block sequence. Channel boundaries are
// driven by the compressor, so they fall inside groups by chance rather than by design.
func TestSpanChannelOut_MultiBlockChannelBoundaryInsideGroup(t *testing.T) {
	base := rollupCfg.Genesis.L2Time + 420_000
	// one long group, so wherever the channel fills up the boundary is inside it
	timestamps := make([]uint64, 8)
	for i := range timestamps {
		timestamps[i] = base
	}
	cfg, batches, l1Origin := multiBlockChannelBatches(t, timestamps, base-rollupCfg.BlockTime)

	first, err := NewSpanChannelOut(300, Zlib, rollup.NewChainSpec(&cfg))
	require.NoError(t, err)
	added := 0
	for i, b := range batches {
		if err := first.addSingularBatch(b, uint64(i)); errors.Is(err, ErrCompressorFull) {
			break
		} else {
			require.NoError(t, err)
		}
		added++
	}
	require.Positive(t, added)
	require.Less(t, added, len(batches), "the channel must fill up before the group ends")

	// the batcher opens the follow-on channel on the last block the full one took
	second, err := NewSpanChannelOut(100_000, Zlib, rollup.NewChainSpec(&cfg),
		WithParentTimestamp(batches[added-1].Timestamp))
	require.NoError(t, err)
	for i, b := range batches[added:] {
		require.NoError(t, second.addSingularBatch(b, uint64(added+i)))
	}

	firstTypes, firstSpans := readSpanBatches(t, &cfg, first, l1Origin)
	secondTypes, secondSpans := readSpanBatches(t, &cfg, second, l1Origin)
	require.Equal(t, []uint8{SpanBatchV2Type}, firstTypes)
	require.Equal(t, []uint8{SpanBatchV2Type}, secondTypes)
	require.Equal(t, uint(0), firstSpans[0].sameTsBits.Bit(0))
	require.Equal(t, uint(1), secondSpans[0].sameTsBits.Bit(0),
		"the follow-on channel starts on a sibling of the previous channel's last block")
	require.Equal(t, timestamps, append(spanTimestamps(firstSpans), spanTimestamps(secondSpans)...))
}

// TestSpanChannelOut_MultiBlockResetKeepsParentTimestamp checks that resetting a channel keeps the
// parent it was opened on: a reset rebuilds the same channel from the same blocks, so its first
// block is still a sibling of that parent.
func TestSpanChannelOut_MultiBlockResetKeepsParentTimestamp(t *testing.T) {
	base := rollupCfg.Genesis.L2Time + 420_000
	cfg, batches, _ := multiBlockChannelBatches(t, []uint64{base, base}, base-rollupCfg.BlockTime)

	cout, err := NewSpanChannelOut(100_000, Zlib, rollup.NewChainSpec(&cfg), WithParentTimestamp(base))
	require.NoError(t, err)
	require.NoError(t, cout.addSingularBatch(batches[0], 0))
	require.NoError(t, cout.Reset())
	require.NoError(t, cout.addSingularBatch(batches[1], 1))

	require.Equal(t, uint(1), cout.spanBatch.sameTsBits.Bit(0))
}

// staticRawChannelProvider hands out one prepared channel and then reports EOF.
type staticRawChannelProvider struct {
	origin eth.L1BlockRef
	data   []byte
	served bool
}

func (p *staticRawChannelProvider) Origin() eth.L1BlockRef { return p.origin }
func (p *staticRawChannelProvider) FlushChannel()          {}

func (p *staticRawChannelProvider) Reset(context.Context, eth.L1BlockRef, eth.SystemConfig) error {
	return io.EOF
}

func (p *staticRawChannelProvider) NextRawChannel(context.Context) ([]byte, error) {
	if p.served {
		return nil, io.EOF
	}
	p.served = true
	return p.data, nil
}

// TestChannelInReaderDropsSpanBatchV2 checks that op-node, which does not derive multi-blocks,
// drops a span batch v2 as a missing-data condition rather than failing the pipeline, and keeps
// reading the batches that follow it in the channel.
func TestChannelInReaderDropsSpanBatchV2(t *testing.T) {
	base := rollupCfg.Genesis.L2Time + 420_000
	cfg, batches, l1Origin := multiBlockChannelBatches(t, []uint64{base, base, base + 2}, base-rollupCfg.BlockTime)
	l1Origin.Time = base
	cfg.DeltaTime = ptr.New(uint64(0))

	// one v2 span holding the group, then a v1 span with the trailing block
	cout, err := NewSpanChannelOut(100_000, Zlib, rollup.NewChainSpec(&cfg), WithMaxBlocksPerSpanBatch(2))
	require.NoError(t, err)
	for i, b := range batches[:2] {
		require.NoError(t, cout.addSingularBatch(b, uint64(i)))
	}
	cfg.MultiBlockTime = ptr.New(base + 1_000_000)
	require.NoError(t, cout.addSingularBatch(batches[2], 2))
	require.NoError(t, cout.Close())

	var frameBuf bytes.Buffer
	_, err = cout.OutputFrame(&frameBuf, 100_000)
	require.ErrorIs(t, err, io.EOF)
	var frame Frame
	require.NoError(t, frame.UnmarshalBinary(&frameBuf))
	ch := NewChannel(frame.ID, l1Origin, false)
	require.NoError(t, ch.AddFrame(frame, l1Origin))
	channelData, err := io.ReadAll(ch.Reader())
	require.NoError(t, err)

	cfg.MultiBlockTime = nil
	cr := NewChannelInReader(&cfg, testlog.Logger(t, log.LevelError),
		&staticRawChannelProvider{origin: l1Origin, data: channelData}, metrics.NoopMetrics)

	// the v2 span is dropped without failing the pipeline
	batch, err := cr.NextBatch(context.Background())
	require.ErrorIs(t, err, NotEnoughData)
	require.Nil(t, batch)

	// the v1 span that follows it in the same channel is still read
	batch, err = cr.NextBatch(context.Background())
	require.NoError(t, err)
	span, ok := batch.AsSpanBatch()
	require.True(t, ok)
	require.Equal(t, []uint64{base + 2}, spanTimestamps([]*SpanBatch{span}))
}
