package derive

import (
	"bytes"
	"compress/zlib"
	"context"
	"io"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"

	"github.com/ethereum-optimism/optimism/op-node/metrics"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/ptr"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

// fakeRawChannelProvider hands out pre-built channels, one per NextRawChannel call.
type fakeRawChannelProvider struct {
	origin   eth.L1BlockRef
	channels [][]byte
	flushed  int
}

var _ RawChannelProvider = (*fakeRawChannelProvider)(nil)

func (f *fakeRawChannelProvider) Origin() eth.L1BlockRef { return f.origin }

func (f *fakeRawChannelProvider) NextRawChannel(context.Context) ([]byte, error) {
	if len(f.channels) == 0 {
		return nil, io.EOF
	}
	ch := f.channels[0]
	f.channels = f.channels[1:]
	return ch, nil
}

func (f *fakeRawChannelProvider) Reset(context.Context, eth.L1BlockRef, eth.SystemConfig) error {
	return io.EOF
}

func (f *fakeRawChannelProvider) FlushChannel() { f.flushed++ }

// encodeChannel zlib-compresses the RLP stream of the given batches, the form BatchReader expects.
func encodeChannel(t *testing.T, batches ...*BatchData) []byte {
	t.Helper()
	var out bytes.Buffer
	zw := zlib.NewWriter(&out)
	for _, b := range batches {
		require.NoError(t, rlp.Encode(zw, b))
	}
	require.NoError(t, zw.Close())
	return out.Bytes()
}

func newTestChannelInReader(t *testing.T, channels ...[]byte) (*ChannelInReader, *fakeRawChannelProvider) {
	t.Helper()
	prev := &fakeRawChannelProvider{
		origin:   eth.L1BlockRef{Number: 100, Time: 5_000},
		channels: channels,
	}
	return NewChannelInReader(spanBatchV2RollupConfig(), testlog.Logger(t, log.LevelInfo), prev, metrics.NoopMetrics), prev
}

func spanBatchV2Data(t *testing.T, parentTimestamp uint64, elems [][2]uint64) *BatchData {
	t.Helper()
	raw, err := buildSpanBatchV2(t, parentTimestamp, elems).ToRawSpanBatch()
	require.NoError(t, err)
	return NewBatchData(raw)
}

// Batches decoded before the v2 span are unaffected: only the remainder of the channel is dropped.
func TestChannelInReader_SpanBatchV2DropsOnlyTheRemainder(t *testing.T) {
	channel := encodeChannel(t,
		NewBatchData(spanBatchV2Element(t, 1010, 1)),
		spanBatchV2Data(t, 1010, [][2]uint64{{1012, 1}}),
		NewBatchData(spanBatchV2Element(t, 1014, 1)),
	)
	cr, _ := newTestChannelInReader(t, channel)

	batch, err := cr.NextBatch(context.Background())
	require.NoError(t, err)
	require.Equal(t, SingularBatchType, batch.GetBatchType())
	require.Equal(t, uint64(1010), batch.GetTimestamp())

	batch, err = cr.NextBatch(context.Background())
	require.Nil(t, batch)
	require.ErrorIs(t, err, NotEnoughData)

	batch, err = cr.NextBatch(context.Background())
	require.Nil(t, batch)
	require.ErrorIs(t, err, io.EOF)
}

// A following channel is still read: the drop is scoped to the channel holding the v2 span.
func TestChannelInReader_SpanBatchV2KeepsNextChannel(t *testing.T) {
	cr, _ := newTestChannelInReader(t,
		encodeChannel(t,
			spanBatchV2Data(t, 1008, [][2]uint64{{1010, 1}}),
			NewBatchData(spanBatchV2Element(t, 1012, 1)),
		),
		encodeChannel(t, NewBatchData(spanBatchV2Element(t, 1014, 1))),
	)

	batch, err := cr.NextBatch(context.Background())
	require.Nil(t, batch)
	require.ErrorIs(t, err, NotEnoughData)

	batch, err = cr.NextBatch(context.Background())
	require.NoError(t, err)
	require.Equal(t, SingularBatchType, batch.GetBatchType())
	require.Equal(t, uint64(1014), batch.GetTimestamp())
}

// TestChannelInReaderDropsChannelWithSpanBatchV2 checks that op-node, which does not derive
// multi-blocks, discards the whole channel a span batch v2 sits in, so the batches behind it never
// reach the safe chain.
func TestChannelInReaderDropsChannelWithSpanBatchV2(t *testing.T) {
	base := rollupCfg.Genesis.L2Time + 420_000
	cfg, batches, l1Origin := multiBlockChannelBatches(t, []uint64{base, base, base + 2}, base-rollupCfg.BlockTime)
	l1Origin.Time = base
	cfg.DeltaTime = ptr.New(uint64(0))

	// one v2 span holding the group, then a v1 span with the trailing block
	cout, err := NewSpanChannelOut(100_000, Zlib, rollup.NewChainSpec(&cfg), WithMaxBlocksPerSpanBatch(2),
		WithParentTimestamp(base-rollupCfg.BlockTime))
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

	// NewChainSpec keeps the pointer, so flipping MultiBlockTime above moved the channel out from
	// v2 to v1 mid-channel; unset now, the reader sees a chain without multi-blocks at all
	cfg.MultiBlockTime = nil
	prev := &fakeRawChannelProvider{origin: l1Origin, channels: [][]byte{channelData}}
	cr := NewChannelInReader(&cfg, testlog.Logger(t, log.LevelError), prev, metrics.NoopMetrics)

	// the v2 span is dropped without failing the pipeline
	batch, err := cr.NextBatch(context.Background())
	require.ErrorIs(t, err, NotEnoughData)
	require.Nil(t, batch)

	// the v1 span that follows it in the same channel goes with it
	batch, err = cr.NextBatch(context.Background())
	require.ErrorIs(t, err, io.EOF)
	require.Nil(t, batch)
}
