package derive

import (
	"time"
)

// DerivationMetrics provides observability into the derivation pipeline's performance.
// Each stage of the pipeline reports timing, throughput, and queue depth metrics
// To enable identification of bottlenecks and performance optimization opportunities.
type DerivationMetrics interface {
	// RecordStageProcessing records the time taken to process a single step in a pipeline stage.
	// Parameters:
	// - stage: name of the pipeline stage
	// - duration: time spent processing
	// - result: outcome of processing (success, empty, error, filtered)
	RecordStageProcessing(stage string, duration time.Duration, result string)

	// RecordStageQueueDepth records the current depth of a stage's queue.
	// Parameters:
	// - stage: name of the pipeline stage
	// - depth: current number of items waiting to be processed
	RecordStageQueueDepth(stage string, depth int)

	// RecordStageItemProcessed increments the counter of items processed by a stage.
	// Parameters:
	//   - stage: name of the pipeline stage
	//   - result: outcome ("success", "filtered", "invalid", "error")
	RecordStageItemProcessed(stage string, result string)

	// RecordStageBytesProcessed increments the counter of bytes processed by a stage.
	// Useful for understanding data flow through the pipeline.
	// Parameters:
	//   - stage: name of the pipeline stage
	//   - bytes: number of bytes processed
	RecordStageBytesProcessed(stage string, bytes int64)

	// RecordPipelineReset records when the pipeline is reset and why.
	// Pipeline resets are expensive operations that indicate issues.
	// Parameters:
	//   - reason: categorized reason for reset (e.g., "l1_reorg", "engine_error", "missing_data")
	RecordPipelineReset(reason string)

	// RecordL1BlockProcessingTime records end-to-end time from seeing an L1 block
	// to completing its processing through all derivation stages.
	// Parameters:
	//   - duration: total time from L1 block fetch to payload derivation
	RecordL1BlockProcessingTime(duration time.Duration)

	// RecordStageWaitTime records time spent waiting for data from outer stages.
	// This helps distinguish processing time from coordination/wait time.
	// Parameters:
	//   - stage: name of the pipeline stage
	//   - duration: time spent waiting
	RecordStageWaitTime(stage string, duration time.Duration)
}

// NoopDerivationMetrics is a metrics implementation that does nothing\
// Useful for testing or when metrics are disabled
type NoopDerivationMetrics struct{}

func (NoopDerivationMetrics) RecordStageProcessing(stage string, duration time.Duration, result string) {
}
func (NoopDerivationMetrics) RecordStageQueueDepth(stage string, depth int)            {}
func (NoopDerivationMetrics) RecordStageItemProcessed(stage string, result string)     {}
func (NoopDerivationMetrics) RecordStageBytesProcessed(stage string, bytes int64)      {}
func (NoopDerivationMetrics) RecordPipelineReset(reason string)                        {}
func (NoopDerivationMetrics) RecordL1BlockProcessingTime(duration time.Duration)       {}
func (NoopDerivationMetrics) RecordStageWaitTime(stage string, duration time.Duration) {}

// Pipeline stage name constants for consistency
const (
	StageL1Traversal     = "l1_traversal"
	StageL1Retrieval     = "l1_retrieval"
	StageFrameQueue      = "frame_queue"
	StageChannelBank     = "channel_bank"
	StageChannelReader   = "channel_reader"
	StageBatchQueue      = "batch_queue"
	StageAttributesQueue = "attributes_queue"
	StageEngineQueue     = "engine_queue"
)

// Result type constants for consistency
const (
	ResultSuccess  = "success"  // Item processed successfully
	ResultEmpty    = "empty"    // No data available to process
	ResultError    = "error"    // Error during processing
	ResultFiltered = "filtered" // Item filtered out as invalid
	ResultCached   = "cached"   // Item served from cache
)

// Reset reason constants
const (
	ResetReasonL1Reorg        = "l1_reorg"
	ResetReasonEngineError    = "engine_error"
	ResetReasonMissingData    = "missing_data"
	ResetReasonInvalidBatch   = "invalid_batch"
	ResetReasonChannelTimeout = "channel_timeout"
	ResetReasonManual         = "manual"
)
