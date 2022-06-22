package derive

import (
	"bytes"
	"compress/zlib"
	"context"
	"fmt"
	"github.com/ethereum/go-ethereum/log"
	"io"

	"github.com/ethereum/go-ethereum/rlp"

	"github.com/ethereum-optimism/optimism/op-node/eth"
)

// zlib returns an io.ReadCloser but explicitly documents it is also a zlib.Resetter, and we want to use it as such.
type zlibReader interface {
	io.ReadCloser
	zlib.Resetter
}

type BatchQueueStage interface {
	OriginStage
	AddBatch(batch *BatchData) error
}

type ChannelInReader struct {
	log log.Logger

	ready    bool
	r        *bytes.Reader
	readZlib zlibReader
	readRLP  *rlp.Stream

	currentOrigin eth.L1BlockRef
	originOpen    bool
	data          []byte

	next BatchQueueStage
}

var _ ChannelBankOutput = (*ChannelInReader)(nil)

// NewChannelInReader creates a ChannelInReader, which should be Reset(origin) before use.
func NewChannelInReader(log log.Logger, next BatchQueueStage) *ChannelInReader {
	return &ChannelInReader{log: log, next: next}
}

func (cr *ChannelInReader) OpenOrigin(origin eth.L1BlockRef) error {
	if cr.currentOrigin.Hash != origin.ParentHash {
		return fmt.Errorf("next origin %s does not build on top of current origin %s, but on %s", origin.ID(), cr.currentOrigin.ID(), origin.ParentID())
	}
	cr.currentOrigin = origin
	cr.originOpen = true
	return nil
}

// CurrentOrigin returns the L1 block that encodes the data that is currently being read.
// Batches should be filtered based on this source.
// Note that the source might not be canonical anymore by the time the data is processed.
func (cr *ChannelInReader) CurrentOrigin() eth.L1BlockRef {
	return cr.currentOrigin
}

func (cr *ChannelInReader) CloseOrigin() {
	cr.originOpen = false
}

func (cr *ChannelInReader) IsOriginOpen() bool {
	return cr.originOpen
}

func (cr *ChannelInReader) WriteChannel(data []byte) {
	cr.data = data
	cr.ready = false
}

// ReadBatch returns a decoded rollup batch, or an error:
// - io.EOF, if the ChannelInReader source needs more data, to be provided with NextL1Origin() and ResetChannel().
// - any other error (e.g. invalid compression or batch data):
//   the caller should ChannelInReader.NextChannel() before continuing reading the next batch.
//
// It's up to the caller to check CurrentL1Origin() before reading more information.
// The CurrentL1Origin() does not change until the first ReadBatch() after the old source has been completely exhausted.
func (cr *ChannelInReader) ReadBatch(dest *BatchData) error {
	// The channel reader may not be initialized yet,
	// and initializing involves reading (zlib header data), so we do that now.
	if !cr.ready {
		if cr.data == nil {
			return io.EOF
		}
		if cr.r == nil {
			cr.r = bytes.NewReader(cr.data)
		} else {
			cr.r.Reset(cr.data)
		}
		if cr.readZlib == nil {
			// creating a new zlib reader involves resetting it, which reads data, which may error
			zr, err := zlib.NewReader(cr.r)
			if err != nil {
				return err
			}
			cr.readZlib = zr.(zlibReader)
		} else {
			err := cr.readZlib.Reset(cr.r, nil)
			if err != nil {
				return err
			}
		}
		if cr.readRLP == nil {
			cr.readRLP = rlp.NewStream(cr.readZlib, 10_000_000)
		} else {
			cr.readRLP.Reset(cr.readZlib, 10_000_000)
		}
		cr.ready = true
	}
	return cr.readRLP.Decode(dest)
}

// NextChannel forces the next read to continue with the next channel,
// resetting any decoding/decompression state to a fresh start.
func (cr *ChannelInReader) NextChannel() {
	cr.ready = false
	cr.data = nil
}

func (cr *ChannelInReader) Step(ctx context.Context) error {
	// move forward the batch queue if the ch reader has new L1 data
	if cr.next.CurrentOrigin() != cr.CurrentOrigin() {
		return cr.next.OpenOrigin(cr.CurrentOrigin())
	}
	var batch BatchData
	if err := cr.ReadBatch(&batch); err == io.EOF {
		// mark the origin as ended if the ch reader marked it as ended
		if !cr.IsOriginOpen() {
			cr.next.CloseOrigin()
		}
		return io.EOF
	} else if err != nil {
		cr.log.Warn("failed to read batch from channel reader, skipping to next channel now", "err", err)
		cr.NextChannel()
		return nil
	}
	cr.log.Debug("reading channel", "batch_epoch", batch.Epoch, "batch_timestamp", batch.Timestamp, "txs", len(batch.Transactions))
	return cr.next.AddBatch(&batch)
}

func (cr *ChannelInReader) ResetStep(ctx context.Context, l1Fetcher L1Fetcher) error {
	cr.ready = false
	cr.currentOrigin = cr.next.CurrentOrigin()
	cr.originOpen = true
	return io.EOF
}
