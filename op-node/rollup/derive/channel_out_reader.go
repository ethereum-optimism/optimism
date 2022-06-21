package derive

import (
	"bytes"
	"compress/zlib"
	"context"
	"io"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
)

// channelOutReader is an io.Reader that produces the data for a channel,
// to be submitted to the data-availability layer in frames.
type channelOutReader struct {
	source   BlocksSource
	blocks   []eth.BlockID
	i        int
	compress *zlib.Writer
	buf      *bytes.Buffer
	closed   bool
	genesis  *rollup.Genesis
	ctx      context.Context
}

func newChannelOutReader(ctx context.Context, genesis *rollup.Genesis, source BlocksSource, blocks []eth.BlockID) (*channelOutReader, error) {
	buf := new(bytes.Buffer)
	w, err := zlib.NewWriterLevel(buf, zlib.BestCompression)
	if err != nil {
		return nil, err
	}
	return &channelOutReader{
		source:   source,
		blocks:   blocks,
		i:        0,
		compress: w,
		buf:      buf,
		genesis:  genesis,
		ctx:      ctx,
	}, nil
}

// TODO: it would be nice to re-use the channel reader for new channels,
// but only after the channel is removed and the old channel is no longer read
func (cr *channelOutReader) Reset(ctx context.Context, blocks []eth.BlockID) {
	cr.blocks = blocks
	cr.i = 0
	cr.buf.Reset()
	cr.compress.Reset(cr.buf)
	cr.ctx = ctx
	cr.closed = false
}

func (cr *channelOutReader) readPayload() (*eth.ExecutionPayload, error) {
	if len(cr.blocks) == 0 {
		return nil, io.EOF
	}
	payload, err := cr.source.PayloadByHash(cr.ctx, cr.blocks[0].Hash)
	if err != nil {
		return nil, err
	}
	cr.blocks = cr.blocks[1:]
	return payload, nil
}

func (cr *channelOutReader) readBatch() (*BatchData, error) {
	payload, err := cr.readPayload()
	if err != nil {
		return nil, err
	}
	ref, err := PayloadToBlockRef(payload, cr.genesis)
	if err != nil {
		return nil, err
	}
	var opaqueTxs []hexutil.Bytes
	for _, otx := range payload.Transactions {
		if otx[0] == types.DepositTxType {
			continue
		}
		opaqueTxs = append(opaqueTxs, otx)
	}
	return &BatchData{BatchV1: BatchV1{
		Epoch:        rollup.Epoch(ref.L1Origin.Number), // the L1 block number equals the L2 epoch.
		Timestamp:    uint64(payload.Timestamp),
		Transactions: opaqueTxs,
	}}, nil
}

func (cr *channelOutReader) encodeNext() error {
	batch, err := cr.readBatch()
	if err != nil {
		return err
	}
	return rlp.Encode(cr.compress, batch)
}

func (cr *channelOutReader) Read(p []byte) (n int, err error) {
	// try to empty the buffer first, we cannot write to it until we have read it all
	bufN, err := cr.buf.Read(p)
	if err != nil { // *bytes.Buffer.Read() only returns io.EOF errors, and only if the buffer is empty.
		if cr.closed {
			return 0, io.EOF
		}
		// if the buffer is empty, then encode the next block to it
		if err := cr.encodeNext(); err != nil {
			if err == io.EOF { // if there are no more blocks, close (includes flush) the compression stream
				if err := cr.compress.Close(); err != nil {
					return 0, err
				}
				cr.closed = true
				// read what remains (if any). May return an EOF if the flush left nothing.
				return cr.buf.Read(p)
			} else if err != nil {
				return 0, err
			}
		}
		// compression may have buffered new data, but it may not be flushed. Read more from buf next call.
		return 0, nil
	}
	return bufN, nil
}
