package derive

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/ethereum/go-ethereum/log"
)

// Channel In Reader reads a batch from the channel
// This does decompression and limits the max RLP size
// This is a pure function from the channel, but each channel (or channel fragment)
// must be tagged with an L1 inclusion block to be passed to the the batch queue.

type ChannelInReader struct {
	log log.Logger

	nextBatchFn func() (BatchWithL1InclusionBlock, error)
	progress    Progress
	prev        *ChannelBank
}

var _ PullStage = (*ChannelInReader)(nil)

// NewChannelInReader creates a ChannelInReader, which should be Reset(origin) before use.
func NewChannelInReader(log log.Logger, prev *ChannelBank) *ChannelInReader {
	return &ChannelInReader{log: log, prev: prev}
}

func (cr *ChannelInReader) Progress() Progress {
	return cr.progress
}

func (cr *ChannelInReader) NextBatch(ctx context.Context) (*BatchData, error) {
	// Try to load up more data if needed
	if cr.nextBatchFn == nil {
		if data, err := cr.prev.NextData(ctx); err == io.EOF {
			return nil, io.EOF
		} else if err != nil {
			if f, err := BatchReader(bytes.NewBuffer(data), cr.progress.Origin); err != nil {
				cr.log.Error("Error creating batch reader from channel data", "err", err)
				return nil, NewTemporaryError(fmt.Errorf("failed to create batch reader: %w", err))
			} else {
				cr.nextBatchFn = f
			}
		} else {
			return nil, fmt.Errorf("failed to read from channel bank: %w", err)
		}
	}

	// Return the cached data if we have it
	if batch, err := cr.nextBatchFn(); err == io.EOF {
		cr.nextBatchFn = nil
		return nil, io.EOF
	} else if err != nil {
		cr.log.Warn("failed to read batch from channel reader, skipping to next channel now", "err", err)
		return nil, err
	} else {
		return batch.Batch, nil
	}
}

func (cr *ChannelInReader) Reset(ctx context.Context, inner Progress) error {
	cr.nextBatchFn = nil
	cr.progress = inner
	return io.EOF
}
