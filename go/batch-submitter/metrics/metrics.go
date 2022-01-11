package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type Metrics struct {
	// ETHBalance tracks the amount of ETH in the submitter's account.
	ETHBalance prometheus.Gauge

	// BatchSizeInBytes tracks the size of batch submission transactions.
	BatchSizeInBytes prometheus.Histogram

	// NumElementsPerBatch tracks the number of L2 transactions in each batch
	// submission.
	NumElementsPerBatch prometheus.Histogram

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
	return &Metrics{
		ETHBalance: promauto.NewGauge(prometheus.GaugeOpts{
			Name:      "batch_submitter_eth_balance",
			Help:      "ETH balance of the batch submitter",
			Subsystem: subsystem,
		}),
		BatchSizeInBytes: promauto.NewSummary(prometheus.SummaryOpts{
			Name:       "batch_size_bytes",
			Help:       "Size of batches in bytes",
			Subsystem:  subsystem,
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		}),
		NumElementsPerBatch: promauto.NewHistogram(prometheus.HistogramOpts{
			Name: "num_elements_per_batch",
			Help: "Number of transaction in each batch",
			Buckets: []float64{
				250,
				500,
				750,
				1000,
				1250,
				1500,
				1750,
				2000,
				2250,
				2500,
				2750,
				3000,
			},
			Subsystem: subsystem,
		}),
		SubmissionTimestamp: promauto.NewGauge(prometheus.GaugeOpts{
			Name:      "submission_timestamp",
			Help:      "Timestamp of last batch submitter submission",
			Subsystem: subsystem,
		}),
		SubmissionGasUsed: promauto.NewGauge(prometheus.GaugeOpts{
			Name:      "submission_gas_used",
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
			Name:      "batch_submitter_batch_confirmation_time_ms",
			Help:      "Time to confirm batch transactions",
			Subsystem: subsystem,
		}),
		BatchPruneCount: promauto.NewGauge(prometheus.GaugeOpts{
			Name:      "batch_submitter_batch_prune_count",
			Help:      "Number of times a batch is pruned",
			Subsystem: subsystem,
		}),
	}
}
