package derive

import (
	"bytes"
	"compress/zlib"
	"context"
	"io"

	"github.com/ethereum/go-ethereum/log"
)

// Channel In Reader reads a batch from the channel
// This does decompression and limits the max RLP size
// This is a pure function from the channel, but each channel (or channel fragment)
// must be tagged with an L1 inclusion block to be passed to the the batch queue.

// zlib returns an io.ReadCloser but explicitly documents it is also a zlib.Resetter, and we want to use it as such.
type zlibReader interface {
	io.ReadCloser
	zlib.Resetter
}

type BatchQueueStage interface {
	StageProgress
	AddBatch(batch *BatchData)
}

type ChannelInReader struct {
	log log.Logger

	nextBatchFn func() (BatchWithL1InclusionBlock, error)

	progress Progress

	next BatchQueueStage
}

var _ ChannelBankOutput = (*ChannelInReader)(nil)

// NewChannelInReader creates a ChannelInReader, which should be Reset(origin) before use.
func NewChannelInReader(log log.Logger, next BatchQueueStage) *ChannelInReader {
	return &ChannelInReader{log: log, next: next}
}

func (cr *ChannelInReader) Progress() Progress {
	return cr.progress
}

// TODO: Take full channel for better logging
func (cr *ChannelInReader) WriteChannel(data []byte) {
	if cr.progress.Closed {
		panic("write channel while closed")
	}
	if f, err := BatchReader(bytes.NewBuffer(data), cr.progress.Origin); err == nil {
		cr.nextBatchFn = f
	} else {
		cr.log.Error("Error creating batch reader from channel data", "err", err)
	}
}

// NextChannel forces the next read to continue with the next channel,
// resetting any decoding/decompression state to a fresh start.
func (cr *ChannelInReader) NextChannel() {
	cr.nextBatchFn = nil
}

func (cr *ChannelInReader) Step(ctx context.Context, outer Progress) error {
	if changed, err := cr.progress.Update(outer); err != nil || changed {
		return err
	}

	// TODO: can batch be non nil while err == io.EOF
	// This depends on the behavior of rlp.Stream
	batch, err := cr.nextBatchFn()

	if err == io.EOF {
		return io.EOF
	} else if err != nil {
		cr.log.Warn("failed to read batch from channel reader, skipping to next channel now", "err", err)
		cr.NextChannel()
		return nil
	}
	cr.next.AddBatch(batch.Batch)
	return nil
}

func (cr *ChannelInReader) ResetStep(ctx context.Context, l1Fetcher L1Fetcher) error {
	cr.nextBatchFn = nil
	cr.progress = cr.next.Progress()
	return io.EOF
}
