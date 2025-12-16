package driver

import (
	"time"

	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// DerivationMetricsImpl implements derive.DerivationMetrics using Prometheus
type DerivationMetricsImpl struct {
	// Stage processing time distribution
	stageProcessingDuration *prometheus.HistogramVec

	// Queue depths by stage
	stageQueueDepth *prometheus.GaugeVec

	// Items processed by stage
	stageItemsProcessed *prometheus.CounterVec

	// Bytes processed counters
	stageBytesProcessed *prometheus.CounterVec

	// Pipeline reset events
	pipelineResets *prometheus.CounterVec

	// End-to-end L1 block processing time
	l1BlockProcessingDuration prometheus.Histogram

	// Stage wait time distribution
	stageWaitDuration *prometheus.HistogramVec
}

var _ derive.DerivationMetrics = (*DerivationMetricsImpl)(nil)

// NewDerivationMetrics creates a new Prometheus-backed derivation metrics instance
func NewDerivationMetrics(registry *prometheus.Registry, namespace string) *DerivationMetricsImpl {
	if namespace == "" {
		namespace = "op_node"
	}

	factory := promauto.With(registry)

	return &DerivationMetricsImpl{
		stageProcessingDuration: factory.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: "derivation",
			Name:      "stage_duration_seconds",
			Help:      "Time spent processing in each derivation pipeline stage",
			Buckets:   []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
		}, []string{"stage", "result"}),

		stageQueueDepth: factory.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: "derivation",
			Name:      "stage_queue_depth",
			Help:      "Current number of items pending in each pipeline stage",
		}, []string{"stage"}),

		stageItemsProcessed: factory.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "derivation",
			Name:      "stage_items_processed_total",
			Help:      "Total number of items processed by each pipeline stage",
		}, []string{"stage", "result"}),

		stageBytesProcessed: factory.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "derivation",
			Name:      "stage_bytes_processed_total",
			Help:      "Total number of bytes processed by each pipeline stage",
		}, []string{"stage"}),

		pipelineResets: factory.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "derivation",
			Name:      "pipeline_resets_total",
			Help:      "Total number of times the derivation pipeline has been reset",
		}, []string{"reason"}),

		l1BlockProcessingDuration: factory.NewHistogram(prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: "derivation",
			Name:      "l1_block_processing_duration_seconds",
			Help:      "Time taken to process an L1 block from fetch to payload derivation",
			Buckets:   []float64{0.1, 0.25, 0.5, 1.0, 2.5, 5.0, 10.0, 30.0, 60.0, 120.0},
		}),

		stageWaitDuration: factory.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: "derivation",
			Name:      "stage_wait_duration_seconds",
			Help:      "Time spent waiting for data from outer stages in the derivation pipeline",
			Buckets:   []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
		}, []string{"stage"}),
	}
}

func (m *DerivationMetricsImpl) RecordStageProcessing(stage string, duration time.Duration, result string) {
	m.stageProcessingDuration.WithLabelValues(stage, result).Observe(duration.Seconds())
}

func (m *DerivationMetricsImpl) RecordStageQueueDepth(stage string, depth int) {
	m.stageQueueDepth.WithLabelValues(stage).Set(float64(depth))
}

func (m *DerivationMetricsImpl) RecordStageItemProcessed(stage string, result string) {
	m.stageItemsProcessed.WithLabelValues(stage, result).Inc()
}

func (m *DerivationMetricsImpl) RecordStageBytesProcessed(stage string, bytes int64) {
	m.stageBytesProcessed.WithLabelValues(stage).Add(float64(bytes))
}

func (m *DerivationMetricsImpl) RecordPipelineReset(reason string) {
	m.pipelineResets.WithLabelValues(reason).Inc()
}

func (m *DerivationMetricsImpl) RecordL1BlockProcessingTime(duration time.Duration) {
	m.l1BlockProcessingDuration.Observe(duration.Seconds())
}

func (m *DerivationMetricsImpl) RecordStageWaitTime(stage string, duration time.Duration) {
	m.stageWaitDuration.WithLabelValues(stage).Observe(duration.Seconds())
}

// Helper function to time a stage operation
func TimeStage(m derive.DerivationMetrics, stage string, fn func() (string, error)) error {
	start := time.Now()
	result, err := fn()

	if err != nil {
		m.RecordStageProcessing(stage, time.Since(start), derive.ResultError)
		m.RecordStageItemProcessed(stage, derive.ResultError)
		return err
	}

	m.RecordStageProcessing(stage, time.Since(start), result)
	m.RecordStageItemProcessed(stage, result)
	return nil
}

// Example of how to instrument a stage
/*
func (s *SomeStage) Step(ctx context.Context) error {
	return TimeStage(s.metrics, derive.StageSomeStage, func() (string, error) {
		// Record queue depth
		s.metrics.RecordStageQueueDepth(derive.StageSomeStage, s.queueSize())

		// Do the actual work
		data, err := s.processNext()
		if err != nil {
			return "", err
		}

		if data == nil {
			return derive.ResultEmpty, nil
		}

		// Record bytes processed
		s.metrics.RecordStageBytesProcessed(derive.StageSomeStage, int64(len(data)))

		return derive.ResultSuccess, nil
	})
}
*/
