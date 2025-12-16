package driver

import (
	"errors"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
)

func TestDerivationMetrics(t *testing.T) {
	registry := prometheus.NewRegistry()
	metrics := NewDerivationMetrics(registry, "test_node")

	t.Run("RecordStageProcessing", func(t *testing.T) {
		metrics.RecordStageProcessing(derive.StageL1Traversal, 100*time.Millisecond, derive.ResultSuccess)
		metrics.RecordStageProcessing(derive.StageL1Traversal, 50*time.Millisecond, derive.ResultError)

		// Verify metrics were recorded
		metricFamilies, err := registry.Gather()
		require.NoError(t, err)

		found := false
		for _, mf := range metricFamilies {
			if mf.GetName() == "test_node_derivation_stage_duration_seconds" {
				found = true
				require.Len(t, mf.GetMetric(), 2, "should have 2 label combinations")
			}
		}
		require.True(t, found, "stage duration metric should exist")
	})

	t.Run("RecordStageQueueDepth", func(t *testing.T) {
		// Set different queue depths
		metrics.RecordStageQueueDepth(derive.StageFrameQueue, 10)
		metrics.RecordStageQueueDepth(derive.StageChannelBank, 5)

		metricFamilies, err := registry.Gather()
		require.NoError(t, err)

		found := false
		for _, mf := range metricFamilies {
			if mf.GetName() == "test_node_derivation_stage_queue_depth" {
				found = true
				require.Len(t, mf.GetMetric(), 2, "should have 2 stages tracked")
			}
		}
		require.True(t, found, "queue depth metric should exist")
	})

	t.Run("RecordStageItemProcessed", func(t *testing.T) {
		metrics.RecordStageItemProcessed(derive.StageBatchQueue, derive.ResultSuccess)
		metrics.RecordStageItemProcessed(derive.StageBatchQueue, derive.ResultSuccess)
		metrics.RecordStageItemProcessed(derive.StageBatchQueue, derive.ResultFiltered)

		metricFamilies, err := registry.Gather()
		require.NoError(t, err)

		found := false
		for _, mf := range metricFamilies {
			if mf.GetName() == "test_node_derivation_stage_items_processed_total" {
				found = true
				// Should have 2 counters: one for success, one for filtered
				require.Len(t, mf.GetMetric(), 2)
			}
		}
		require.True(t, found, "items processed metric should exist")
	})

	t.Run("RecordStageBytesProcessed", func(t *testing.T) {
		metrics.RecordStageBytesProcessed(derive.StageL1Retrieval, 1024)
		metrics.RecordStageBytesProcessed(derive.StageL1Retrieval, 2048)

		metricFamilies, err := registry.Gather()
		require.NoError(t, err)

		found := false
		var totalBytes float64
		for _, mf := range metricFamilies {
			if mf.GetName() == "test_node_derivation_stage_bytes_processed_total" {
				found = true
				for _, m := range mf.GetMetric() {
					totalBytes += m.GetCounter().GetValue()
				}
			}
		}
		require.True(t, found, "bytes processed metric should exist")
		require.Equal(t, 3072.0, totalBytes, "should sum to 3072 bytes")
	})

	t.Run("RecordPipelineReset", func(t *testing.T) {
		metrics.RecordPipelineReset(derive.ResetReasonL1Reorg)
		metrics.RecordPipelineReset(derive.ResetReasonEngineError)
		metrics.RecordPipelineReset(derive.ResetReasonL1Reorg)

		metricFamilies, err := registry.Gather()
		require.NoError(t, err)

		found := false
		for _, mf := range metricFamilies {
			if mf.GetName() == "test_node_derivation_pipeline_resets_total" {
				found = true
				// Should have 2 different reset reasons
				require.Len(t, mf.GetMetric(), 2)
			}
		}
		require.True(t, found, "pipeline resets metric should exist")
	})

	t.Run("RecordL1BlockProcessingTime", func(t *testing.T) {
		metrics.RecordL1BlockProcessingTime(1 * time.Second)
		metrics.RecordL1BlockProcessingTime(500 * time.Millisecond)

		metricFamilies, err := registry.Gather()
		require.NoError(t, err)

		found := false
		for _, mf := range metricFamilies {
			if mf.GetName() == "test_node_derivation_l1_block_processing_duration_seconds" {
				found = true
				histogram := mf.GetMetric()[0].GetHistogram()
				require.Equal(t, uint64(2), histogram.GetSampleCount())
			}
		}
		require.True(t, found, "L1 block processing time metric should exist")
	})

	t.Run("RecordStageWaitTime", func(t *testing.T) {
		metrics.RecordStageWaitTime(derive.StageEngineQueue, 50*time.Millisecond)

		metricFamilies, err := registry.Gather()
		require.NoError(t, err)

		found := false
		for _, mf := range metricFamilies {
			if mf.GetName() == "test_node_derivation_stage_wait_duration_seconds" {
				found = true
			}
		}
		require.True(t, found, "stage wait time metric should exist")
	})
}

func TestNoopDerivationMetrics(t *testing.T) {
	// Verify NoopDerivationMetrics doesn't panic
	m := derive.NoopDerivationMetrics{}

	m.RecordStageProcessing("test", time.Second, "success")
	m.RecordStageQueueDepth("test", 10)
	m.RecordStageItemProcessed("test", "success")
	m.RecordStageBytesProcessed("test", 1024)
	m.RecordPipelineReset("test")
	m.RecordL1BlockProcessingTime(time.Second)
	m.RecordStageWaitTime("test", time.Millisecond)

	// If we get here without panicking, the test passes
}

func TestTimeStageHelper(t *testing.T) {
	registry := prometheus.NewRegistry()
	metrics := NewDerivationMetrics(registry, "test_node")

	t.Run("SuccessfulOperation", func(t *testing.T) {
		err := TimeStage(metrics, derive.StageL1Traversal, func() (string, error) {
			time.Sleep(10 * time.Millisecond)
			return derive.ResultSuccess, nil
		})
		require.NoError(t, err)
	})

	t.Run("ErrorOperation", func(t *testing.T) {
		testErr := errors.New("test error")
		err := TimeStage(metrics, derive.StageL1Traversal, func() (string, error) {
			return "", testErr
		})
		require.Error(t, err)
	})
}

// BenchmarkMetricsRecording measures the overhead of recording metrics
func BenchmarkMetricsRecording(b *testing.B) {
	registry := prometheus.NewRegistry()
	metrics := NewDerivationMetrics(registry, "bench_node")

	b.Run("RecordStageProcessing", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			metrics.RecordStageProcessing(derive.StageL1Traversal, 100*time.Millisecond, derive.ResultSuccess)
		}
	})

	b.Run("RecordStageQueueDepth", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			metrics.RecordStageQueueDepth(derive.StageFrameQueue, 10)
		}
	})

	b.Run("RecordAllMetrics", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			metrics.RecordStageProcessing(derive.StageL1Traversal, 100*time.Millisecond, derive.ResultSuccess)
			metrics.RecordStageQueueDepth(derive.StageFrameQueue, 5)
			metrics.RecordStageItemProcessed(derive.StageBatchQueue, derive.ResultSuccess)
			metrics.RecordStageBytesProcessed(derive.StageL1Retrieval, 1024)
		}
	})
}
