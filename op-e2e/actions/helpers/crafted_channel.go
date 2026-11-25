package helpers

import (
	"bytes"
	"compress/zlib"
	"crypto/rand"

	"github.com/andybalholm/brotli"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/stretchr/testify/require"

	optypes "github.com/ethereum-optimism/optimism/op-core/types"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	derive_params "github.com/ethereum-optimism/optimism/op-node/rollup/derive/params"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// CraftedChannel assembles a channel batch by batch, for tests that need a channel the batcher's
// channel outs do not build: a span batch next to other batches in the same channel, a batch that
// no client can derive in front of valid ones, or a compressed stream that breaks at a chosen
// point. It holds each batch in its channel encoding, an RLP string of the batch type byte and the
// batch.
//
// Submit the channel with L2Batcher.ActL2BatchSubmitRaw(t, c.Frame(t, c.Zlib(t))).
type CraftedChannel struct {
	id      derive.ChannelID
	batches [][]byte
}

// NewCraftedChannel returns an empty channel with a random ID.
func NewCraftedChannel(t Testing) *CraftedChannel {
	c := &CraftedChannel{}
	_, err := rand.Read(c.id[:])
	require.NoError(t, err)
	return c
}

// AddSingularBatch adds the block as a singular batch.
func (c *CraftedChannel) AddSingularBatch(t Testing, rollupCfg *rollup.Config, payload *eth.ExecutionPayload) {
	batch, _, err := derive.PayloadToSingularBatch(rollupCfg, payload)
	require.NoError(t, err)
	c.batches = append(c.batches, encodeBatch(t, derive.NewBatchData(batch)))
}

// AddSpanBatch adds the blocks, which must be consecutive, as one span batch. The block's
// transactions go into the batch as they are, so a test can pass blocks whose transactions it
// changed.
func (c *CraftedChannel) AddSpanBatch(t Testing, rollupCfg *rollup.Config, payloads ...*eth.ExecutionPayload) {
	c.batches = append(c.batches, encodeSpanBatch(t, rollupCfg, payloads...))
}

// AddDepositTypedSpanBatch adds a span batch for the block in which the block's user transactions
// are replaced by one transaction typed as a deposit (0x7E). The batch decodes, but a deposit is
// never a valid span batch transaction, so op-node and kona both reject the batch when they derive
// the span batch from its decoded fields.
//
// The batcher's span batch encoder refuses a deposit, so the batch is encoded with a dynamic fee
// transaction of known span batch encoding in its place, and that transaction's type byte is then
// replaced.
func (c *CraftedChannel) AddDepositTypedSpanBatch(t Testing, rollupCfg *rollup.Config, payload *eth.ExecutionPayload) {
	marker := bytes.Repeat([]byte{0x7e, 0xde}, 16)
	placeholder := types.NewTx(&types.DynamicFeeTx{ChainID: rollupCfg.L2ChainID, Data: marker})
	block := *payload
	block.Transactions = nil
	for _, tx := range payload.Transactions {
		if len(tx) > 0 && tx[0] == optypes.DepositTxType {
			block.Transactions = append(block.Transactions, tx)
		}
	}
	block.Transactions = append(block.Transactions, EncodeTx(t, placeholder))
	batch := encodeSpanBatch(t, rollupCfg, &block)

	// A span batch carries a dynamic fee transaction's fields as its type byte followed by the
	// RLP list [value, max_priority_fee_per_gas, max_fee_per_gas, data, access_list].
	fields, err := rlp.EncodeToBytes([]any{uint64(0), uint64(0), uint64(0), marker, []any{}})
	require.NoError(t, err)
	txData := append([]byte{types.DynamicFeeTxType}, fields...)
	require.Equal(t, 1, bytes.Count(batch, txData), "the placeholder transaction's span batch encoding must occur once")
	batch[bytes.Index(batch, txData)] = optypes.DepositTxType

	var decoded derive.BatchData
	require.NoError(t, rlp.DecodeBytes(batch, &decoded), "the retyped span batch must still decode")
	_, err = derive.DeriveSpanBatch(&decoded, rollupCfg)
	require.Error(t, err, "deriving the retyped span batch must fail")
	c.batches = append(c.batches, batch)
}

func encodeSpanBatch(t Testing, rollupCfg *rollup.Config, payloads ...*eth.ExecutionPayload) []byte {
	require.NotEmpty(t, payloads)
	spanBatch := derive.NewSpanBatch(rollupCfg.Genesis.L2Time, rollupCfg.L2ChainID)
	for _, payload := range payloads {
		batch, l1Info, err := derive.PayloadToSingularBatch(rollupCfg, payload)
		require.NoError(t, err)
		require.NoError(t, spanBatch.AppendSingularBatch(batch, l1Info.SequenceNumber))
	}
	rawSpanBatch, err := spanBatch.ToRawSpanBatch()
	require.NoError(t, err)
	return encodeBatch(t, derive.NewBatchData(rawSpanBatch))
}

func encodeBatch(t Testing, batch *derive.BatchData) []byte {
	encoded, err := rlp.EncodeToBytes(batch)
	require.NoError(t, err)
	return encoded
}

// Zlib returns the channel's batches compressed with zlib.
func (c *CraftedChannel) Zlib(t Testing) []byte {
	var buf bytes.Buffer
	w, err := zlib.NewWriterLevel(&buf, zlib.BestCompression)
	require.NoError(t, err)
	for _, batch := range c.batches {
		_, err := w.Write(batch)
		require.NoError(t, err)
	}
	require.NoError(t, w.Close())
	return buf.Bytes()
}

// BrotliFlushedTo returns the channel's batches compressed with brotli, behind the brotli channel
// version byte. The stream is flushed often enough before byte breakAt of the channel to put a
// meta-block boundary at or just after it. BrotliWithFormatError breaks the same stream there.
func (c *CraftedChannel) BrotliFlushedTo(t Testing, breakAt int) []byte {
	out, _ := c.brotliFlushedTo(t, breakAt)
	return out
}

// BrotliWithFormatError returns the stream of BrotliFlushedTo with a format error at its
// meta-block boundary at or after byte breakAt: the header of the meta-block that starts there gets
// its reserved bit set. Everything before the error decodes, and nothing after it does.
func (c *CraftedChannel) BrotliWithFormatError(t Testing, breakAt int) []byte {
	out, brk := c.brotliFlushedTo(t, breakAt)
	// Bit 0 (ISLAST) is 0, bits 1-2 (MNIBBLES) are 3, which makes this a metadata block header,
	// and bit 3 is the reserved bit that must be 0.
	out[brk] = 0x0e
	return out
}

// brotliFlushedTo returns the stream and the offset of its first meta-block boundary at or after
// breakAt.
func (c *CraftedChannel) brotliFlushedTo(t Testing, breakAt int) ([]byte, int) {
	require.Positive(t, breakAt)
	var buf bytes.Buffer
	buf.WriteByte(derive.ChannelVersionBrotli)
	w := brotli.NewWriterLevel(&buf, derive.GetBrotliLevel(derive.Brotli10))
	data := bytes.Join(c.batches, nil)
	brk := -1
	const chunk = 64
	for len(data) > 0 {
		n := len(data)
		if brk < 0 {
			n = min(chunk, n)
		}
		_, err := w.Write(data[:n])
		require.NoError(t, err)
		data = data[n:]
		if brk < 0 {
			// A flush ends the meta-block and pads the stream to a byte boundary, so the next
			// meta-block header starts at the next byte written.
			require.NoError(t, w.Flush())
			if buf.Len() >= breakAt {
				brk = buf.Len()
			}
		}
	}
	require.NoError(t, w.Close())
	out := buf.Bytes()
	require.True(t, brk >= 0 && brk < len(out), "the channel must extend past byte %d to break it there", breakAt)
	return out, brk
}

// Frame returns data as the only frame of the channel, behind the derivation version byte, as
// L2Batcher.ActL2BatchSubmitRaw takes it.
func (c *CraftedChannel) Frame(t Testing, data []byte) []byte {
	var buf bytes.Buffer
	buf.WriteByte(derive_params.DerivationVersion0)
	frame := derive.Frame{ID: c.id, FrameNumber: 0, Data: data, IsLast: true}
	require.NoError(t, frame.MarshalBinary(&buf))
	return buf.Bytes()
}
