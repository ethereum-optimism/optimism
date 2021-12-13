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

	// NumTxPerBatch tracks the number of L2 transactions in each batch
	// submission.
	NumTxPerBatch prometheus.Histogram

	// SubmissionGasUsed tracks the amount of gas used to submit each batch.
	SubmissionGasUsed prometheus.Histogram

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
}

func NewMetrics(subsystem string) *Metrics {
	return &Metrics{
		ETHBalance: promauto.NewGauge(prometheus.GaugeOpts{
			Name:      "batch_submitter_eth_balance",
			Help:      "ETH balance of the batch submitter",
			Subsystem: subsystem,
		}),
		BatchSizeInBytes: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:      "batch_submitter_batch_size_in_bytes",
			Help:      "Size of batches in bytes",
			Subsystem: subsystem,
		}),
		NumTxPerBatch: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:      "batch_submitter_num_txs_per_batch",
			Help:      "Number of transaction in each batch",
			Subsystem: subsystem,
		}),
		SubmissionGasUsed: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:      "batch_submitter_submission_gas_used",
			Help:      "Gas used to submit each batch",
			Subsystem: subsystem,
		}),
		BatchesSubmitted: promauto.NewCounter(prometheus.CounterOpts{
			Name:      "batch_submitter_batches_submitted",
			Help:      "Count of batches submitted",
			Subsystem: subsystem,
		}),
		FailedSubmissions: promauto.NewCounter(prometheus.CounterOpts{
			Name:      "batch_submitter_failed_submissions",
			Help:      "Count of failed batch submissions",
			Subsystem: subsystem,
		}),
		BatchTxBuildTime: promauto.NewGauge(prometheus.GaugeOpts{
			Name:      "batch_submitter_batch_tx_build_time",
			Help:      "Time to construct batch transactions",
			Subsystem: subsystem,
		}),
		BatchConfirmationTime: promauto.NewGauge(prometheus.GaugeOpts{
			Name:      "batch_submitter_batch_confirmation_time",
			Help:      "Time to confirm batch transactions",
			Subsystem: subsystem,
		}),
	}
}
