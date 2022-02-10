package metrics

import (
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type Metrics struct {
	// ETHBalance tracks the amount of ETH in the submitter's account.
	ETHBalance prometheus.Gauge

	// BatchSizeInBytes tracks the size of batch submission transactions.
	BatchSizeInBytes prometheus.Summary

	// NumElementsPerBatch tracks the number of L2 transactions in each batch
	// submission.
	NumElementsPerBatch prometheus.Summary

	// SubmissionTimestamp tracks the time at which each batch was confirmed.
	SubmissionTimestamp prometheus.Gauge

	// SubmissionGasUsed tracks the amount of gas used to submit each batch.
	SubmissionGasUsed prometheus.Gauge

	// BatchsSubmitted tracks the total number of successful batch submissions.
	BatchesSubmitted prometheus.Counter

	// FailedSubmissions tracks the total number of failed batch submissions.
	FailedSubmissions prometheus.Counter

	// BatchTxBuildTime tracks the duration it takes to construct a batch
	// transaction.
	BatchTxBuildTime prometheus.Gauge

	// BatchConfirmationTime tracks the duration it takes to confirm a batch
	// transaction.
	BatchConfirmationTime prometheus.Gauge

	// BatchPruneCount tracks the number of times a batch of sequencer
	// transactions is pruned in order to meet the desired size requirements.
	//
	// NOTE: This is currently only active in the sequencer driver.
	BatchPruneCount prometheus.Gauge
}

func NewMetrics(subsystem string) *Metrics {
	subsystem = "batch_submitter_" + strings.ToLower(subsystem)
	return &Metrics{
		ETHBalance: promauto.NewGauge(prometheus.GaugeOpts{
			Name:      "balance_eth",
			Help:      "ETH balance of the batch submitter",
			Subsystem: subsystem,
		}),
		BatchSizeInBytes: promauto.NewSummary(prometheus.SummaryOpts{
			Name:       "batch_size_bytes",
			Help:       "Size of batches in bytes",
			Subsystem:  subsystem,
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		}),
		NumElementsPerBatch: promauto.NewSummary(prometheus.SummaryOpts{
			Name:       "num_elements_per_batch",
			Help:       "Number of elements in each batch",
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
			Subsystem:  subsystem,
		}),
		SubmissionTimestamp: promauto.NewGauge(prometheus.GaugeOpts{
			Name:      "submission_timestamp_ms",
			Help:      "Timestamp of last batch submitter submission",
			Subsystem: subsystem,
		}),
		SubmissionGasUsed: promauto.NewGauge(prometheus.GaugeOpts{
			Name:      "submission_gas_used_wei",
			Help:      "Gas used to submit each batch",
			Subsystem: subsystem,
		}),
		BatchesSubmitted: promauto.NewCounter(prometheus.CounterOpts{
			Name:      "batches_submitted",
			Help:      "Count of batches submitted",
			Subsystem: subsystem,
		}),
		FailedSubmissions: promauto.NewCounter(prometheus.CounterOpts{
			Name:      "failed_submissions",
			Help:      "Count of failed batch submissions",
			Subsystem: subsystem,
		}),
		BatchTxBuildTime: promauto.NewGauge(prometheus.GaugeOpts{
			Name:      "batch_tx_build_time_ms",
			Help:      "Time to construct batch transactions",
			Subsystem: subsystem,
		}),
		BatchConfirmationTime: promauto.NewGauge(prometheus.GaugeOpts{
			Name:      "batch_confirmation_time_ms",
			Help:      "Time to confirm batch transactions",
			Subsystem: subsystem,
		}),
		BatchPruneCount: promauto.NewGauge(prometheus.GaugeOpts{
			Name:      "batch_prune_count",
			Help:      "Number of times a batch is pruned",
			Subsystem: subsystem,
		}),
	}
}
