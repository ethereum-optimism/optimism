package derive

import (
	"bytes"
	"context"
	"io"

	"github.com/ethereum/go-ethereum/log"
)

// Channel In Reader reads a batch from the channel
// This does decompression and limits the max RLP size
// This is a pure function from the channel, but each channel (or channel fragment)
// must be tagged with an L1 inclusion block to be passed to the the batch queue.

type BatchQueueStage interface {
	StageProgress
	AddBatch(batch *BatchData)
}

type ChannelInReader struct {
	log log.Logger

	nextBatchFn func() (BatchWithL1InclusionBlock, error)

	progress Progress

	next BatchQueueStage
	prev *ChannelBank
}

var _ Stage = (*ChannelInReader)(nil)

// NewChannelInReader creates a ChannelInReader, which should be Reset(origin) before use.
func NewChannelInReader(log log.Logger, next BatchQueueStage, prev *ChannelBank) *ChannelInReader {
	return &ChannelInReader{
		log:  log,
		next: next,
		prev: prev,
	}
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
	// Close ourselves if required
	if cr.progress.Closed {
		if cr.progress.Origin != cr.prev.Origin() {
			cr.progress.Closed = false
			cr.progress.Origin = cr.prev.Origin()
			return nil
		}
	}

	if cr.nextBatchFn == nil {
		if data, err := cr.prev.NextData(ctx); err == io.EOF {
			if !cr.progress.Closed {
				cr.progress.Closed = true
				return nil
			} else {
				return io.EOF
			}
		} else if err != nil {
			return err
		} else {
			cr.WriteChannel(data)
			return nil
		}
	} else {
		// TODO: can batch be non nil while err == io.EOF
		// This depends on the behavior of rlp.Stream
		batch, err := cr.nextBatchFn()
		if err == io.EOF {
			cr.NextChannel()
			return io.EOF
		} else if err != nil {
			cr.log.Warn("failed to read batch from channel reader, skipping to next channel now", "err", err)
			cr.NextChannel()
			return nil
		}
		cr.next.AddBatch(batch.Batch)
		return nil
	}

}

func (cr *ChannelInReader) ResetStep(ctx context.Context, l1Fetcher L1Fetcher) error {
	cr.nextBatchFn = nil
	cr.progress = cr.next.Progress()
	return io.EOF
}
