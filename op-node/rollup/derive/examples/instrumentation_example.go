// Package examples contains example code showing how to instrument derivation pipeline stages.
// This is NOT compiled with the main derive package to avoid type conflicts.
package examples

import (
	"context"
	"time"

	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/log"
)

// L1TraversalExample is a minimal interface representing the L1 Traversal stage
// This is an EXAMPLE showing how to add metrics to an existing stage
type L1TraversalExample struct {
	log     log.Logger
	l1      L1Fetcher
	metrics derive.DerivationMetrics

	// Stage state
	currentL1Block eth.L1BlockRef
	// ... other fields
}

// NewL1TraversalExample creates a new L1 Traversal stage with metrics
func NewL1TraversalExample(log log.Logger, l1 L1Fetcher, metrics derive.DerivationMetrics) *L1TraversalExample {
	return &L1TraversalExample{
		log:     log,
		l1:      l1,
		metrics: metrics,
	}
}

// AdvanceL1Block fetches the next L1 block and advances the traversal stage
// BEFORE INSTRUMENTATION - what the code might have looked like:
/*
func (l1t *L1TraversalExample) AdvanceL1Block(ctx context.Context) (eth.L1BlockRef, error) {
	next := l1t.currentL1Block.Number + 1
	block, err := l1t.l1.L1BlockRefByNumber(ctx, next)
	if err != nil {
		return eth.L1BlockRef{}, err
	}
	l1t.currentL1Block = block
	return block, nil
}
*/

// AFTER INSTRUMENTATION - with comprehensive metrics:
func (l1t *L1TraversalExample) AdvanceL1Block(ctx context.Context) (eth.L1BlockRef, error) {
	start := time.Now()

	// Track result for metrics
	var result string
	defer func() {
		// Record processing time at the end
		l1t.metrics.RecordStageProcessing(derive.StageL1Traversal, time.Since(start), result)
		l1t.metrics.RecordStageItemProcessed(derive.StageL1Traversal, result)
	}()

	// Record queue depth (how many L1 blocks ahead we could process)
	// In L1Traversal this is typically 0 or 1 since we process sequentially
	l1t.metrics.RecordStageQueueDepth(derive.StageL1Traversal, 0)

	// Fetch next L1 block
	next := l1t.currentL1Block.Number + 1
	fetchStart := time.Now()
	block, err := l1t.l1.L1BlockRefByNumber(ctx, next)

	if err != nil {
		l1t.log.Debug("Failed to fetch L1 block", "number", next, "err", err)
		result = derive.ResultError
		return eth.L1BlockRef{}, err
	}

	// Record time spent waiting on L1 RPC
	l1t.metrics.RecordStageWaitTime(derive.StageL1Traversal, time.Since(fetchStart))

	l1t.currentL1Block = block
	result = derive.ResultSuccess

	l1t.log.Debug("Advanced L1 block",
		"number", block.Number,
		"hash", block.Hash,
		"fetch_time", time.Since(fetchStart))

	return block, nil
}

// Reset resets the L1 Traversal stage, typically due to a reorg
func (l1t *L1TraversalExample) Reset(_ context.Context, base eth.L1BlockRef, reason string) error {
	start := time.Now()
	defer func() {
		l1t.metrics.RecordStageProcessing(derive.StageL1Traversal, time.Since(start), "reset")
	}()

	// Record why we're resetting
	l1t.metrics.RecordPipelineReset(reason)

	l1t.log.Info("Resetting L1 Traversal", "base", base, "reason", reason)
	l1t.currentL1Block = base

	return nil
}

// FrameQueueExample shows a more complex stage with queue depth tracking
type FrameQueueExample struct {
	log     log.Logger
	metrics derive.DerivationMetrics

	// Queue of pending frames
	frames []Frame
	// ... other fields
}

func (fq *FrameQueueExample) Step(_ context.Context) error {
	start := time.Now()
	var result string

	defer func() {
		fq.metrics.RecordStageProcessing(derive.StageFrameQueue, time.Since(start), result)
		fq.metrics.RecordStageItemProcessed(derive.StageFrameQueue, result)
	}()

	// Record current queue depth
	queueDepth := len(fq.frames)
	fq.metrics.RecordStageQueueDepth(derive.StageFrameQueue, queueDepth)

	if queueDepth == 0 {
		result = derive.ResultEmpty
		return nil
	}

	// Process next frame
	frame := fq.frames[0]
	fq.frames = fq.frames[1:]

	// Decode and validate frame
	if err := fq.decodeFrame(frame); err != nil {
		fq.log.Warn("Invalid frame", "err", err)
		result = derive.ResultFiltered
		return nil
	}

	// Record bytes processed
	fq.metrics.RecordStageBytesProcessed(derive.StageFrameQueue, int64(len(frame.Data)))

	result = derive.ResultSuccess
	fq.log.Debug("Processed frame",
		"channel_id", frame.ChannelID,
		"frame_number", frame.FrameNumber,
		"bytes", len(frame.Data),
		"remaining_queue", len(fq.frames))

	return nil
}

// Helper types for the example
type Frame struct {
	ChannelID   [16]byte
	FrameNumber uint16
	Data        []byte
}

type L1Fetcher interface {
	L1BlockRefByNumber(ctx context.Context, number uint64) (eth.L1BlockRef, error)
}

func (fq *FrameQueueExample) decodeFrame(_ Frame) error {
	// Placeholder - returns nil to indicate successful decode
	return nil
}

