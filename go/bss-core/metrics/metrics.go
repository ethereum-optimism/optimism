package metrics

import (
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type Base struct {
	// balanceETH tracks the amount of ETH in the submitter's account.
	balanceETH prometheus.Gauge

	// batchSizeBytes tracks the size of batch submission transactions.
	batchSizeBytes prometheus.Summary

	// numElementsPerBatch tracks the number of L2 transactions in each batch
	// submission.
	numElementsPerBatch prometheus.Summary

	// submissionTimestamp tracks the time at which each batch was confirmed.
	submissionTimestamp prometheus.Gauge

	// submissionGasUsedWei tracks the amount of gas used to submit each batch.
	submissionGasUsedWei prometheus.Gauge

	// batchsSubmitted tracks the total number of successful batch submissions.
	batchesSubmitted prometheus.Counter

	// failedSubmissions tracks the total number of failed batch submissions.
	failedSubmissions prometheus.Counter

	// batchTxBuildTimeMs tracks the duration it takes to construct a batch
	// transaction.
	batchTxBuildTimeMs prometheus.Gauge

	// batchConfirmationTimeMs tracks the duration it takes to confirm a batch
	// transaction.
	batchConfirmationTimeMs prometheus.Gauge

	// batchPruneCount tracks the number of times a batch of sequencer
	// transactions is pruned in order to meet the desired size requirements.
	//
	// NOTE: This is currently only active in the sequencer driver.
	batchPruneCount prometheus.Gauge
}

func NewBase(subsystem string) *Base {
	subsystem = "batch_submitter_" + strings.ToLower(subsystem)
	return &Base{
		balanceETH: promauto.NewGauge(prometheus.GaugeOpts{
			Name:      "balance_eth",
			Help:      "ETH balance of the batch submitter",
			Subsystem: subsystem,
		}),
		batchSizeBytes: promauto.NewSummary(prometheus.SummaryOpts{
			Name:       "batch_size_bytes",
			Help:       "Size of batches in bytes",
			Subsystem:  subsystem,
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		}),
		numElementsPerBatch: promauto.NewSummary(prometheus.SummaryOpts{
			Name:       "num_elements_per_batch",
			Help:       "Number of elements in each batch",
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
			Subsystem:  subsystem,
		}),
		submissionTimestamp: promauto.NewGauge(prometheus.GaugeOpts{
			Name:      "submission_timestamp_ms",
			Help:      "Timestamp of last batch submitter submission",
			Subsystem: subsystem,
		}),
		submissionGasUsedWei: promauto.NewGauge(prometheus.GaugeOpts{
			Name:      "submission_gas_used_wei",
			Help:      "Gas used to submit each batch",
			Subsystem: subsystem,
		}),
		batchesSubmitted: promauto.NewCounter(prometheus.CounterOpts{
			Name:      "batches_submitted",
			Help:      "Count of batches submitted",
			Subsystem: subsystem,
		}),
		failedSubmissions: promauto.NewCounter(prometheus.CounterOpts{
			Name:      "failed_submissions",
			Help:      "Count of failed batch submissions",
			Subsystem: subsystem,
		}),
		batchTxBuildTimeMs: promauto.NewGauge(prometheus.GaugeOpts{
			Name:      "batch_tx_build_time_ms",
			Help:      "Time to construct batch transactions",
			Subsystem: subsystem,
		}),
		batchConfirmationTimeMs: promauto.NewGauge(prometheus.GaugeOpts{
			Name:      "batch_confirmation_time_ms",
			Help:      "Time to confirm batch transactions",
			Subsystem: subsystem,
		}),
		batchPruneCount: promauto.NewGauge(prometheus.GaugeOpts{
			Name:      "batch_prune_count",
			Help:      "Number of times a batch is pruned",
			Subsystem: subsystem,
		}),
	}
}

// BalanceETH tracks the amount of ETH in the submitter's account.
func (b *Base) BalanceETH() prometheus.Gauge {
	return b.balanceETH
}

// BatchSizeBytes tracks the size of batch submission transactions.
func (b *Base) BatchSizeBytes() prometheus.Summary {
	return b.batchSizeBytes
}

// NumElementsPerBatch tracks the number of L2 transactions in each batch
// submission.
func (b *Base) NumElementsPerBatch() prometheus.Summary {
	return b.numElementsPerBatch
}

// SubmissionTimestamp tracks the time at which each batch was confirmed.
func (b *Base) SubmissionTimestamp() prometheus.Gauge {
	return b.submissionTimestamp
}

// SubmissionGasUsedWei tracks the amount of gas used to submit each batch.
func (b *Base) SubmissionGasUsedWei() prometheus.Gauge {
	return b.submissionGasUsedWei
}

// BatchsSubmitted tracks the total number of successful batch submissions.
func (b *Base) BatchesSubmitted() prometheus.Counter {
	return b.batchesSubmitted
}

// FailedSubmissions tracks the total number of failed batch submissions.
func (b *Base) FailedSubmissions() prometheus.Counter {
	return b.failedSubmissions
}

// BatchTxBuildTimeMs tracks the duration it takes to construct a batch
// transaction.
func (b *Base) BatchTxBuildTimeMs() prometheus.Gauge {
	return b.batchTxBuildTimeMs
}

// BatchConfirmationTimeMs tracks the duration it takes to confirm a batch
// transaction.
func (b *Base) BatchConfirmationTimeMs() prometheus.Gauge {
	return b.batchConfirmationTimeMs
}

// BatchPruneCount tracks the number of times a batch of sequencer transactions
// is pruned in order to meet the desired size requirements.
//
// NOTE: This is currently only active in the sequencer driver.
func (b *Base) BatchPruneCount() prometheus.Gauge {
	return b.batchPruneCount
}
