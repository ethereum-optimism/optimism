package buidl

import (
	"bytes"
	"compress/zlib"
	"context"
	"io"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
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

func (cr *channelOutReader) readPayload() (*eth.ExecutionPayload, error) {
	if len(cr.blocks) == 0 {
		return nil, io.EOF
	}
	payload, err := cr.source.Block(cr.ctx, cr.blocks[0])
	if err == nil {
		cr.blocks = cr.blocks[1:]
	}
	return payload, err
}

func (cr *channelOutReader) readBatch() (*derive.BatchData, error) {
	payload, err := cr.readPayload()
	if err != nil {
		return nil, err
	}
	ref, err := derive.PayloadToBlockRef(payload, cr.genesis)
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
	return &derive.BatchData{BatchV1: derive.BatchV1{
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
	if err != nil {
		if err == io.EOF {
			err = nil
			if bufN != 0 {
				return bufN, nil
			}
			// if the buffer is empty, then encode the next block to it
			if err := cr.encodeNext(); err != nil {
				return 0, err
			}
			// and read from the new buffer
			return cr.buf.Read(p)
		} else {
			return 0, err
		}
	}
	return bufN, nil
}
