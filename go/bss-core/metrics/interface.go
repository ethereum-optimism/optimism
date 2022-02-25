package metrics

import "github.com/prometheus/client_golang/prometheus"

type Metrics interface {
	// BalanceETH tracks the amount of ETH in the submitter's account.
	BalanceETH() prometheus.Gauge

	// BatchSizeBytes tracks the size of batch submission transactions.
	BatchSizeBytes() prometheus.Summary

	// NumElementsPerBatch tracks the number of L2 transactions in each batch
	// submission.
	NumElementsPerBatch() prometheus.Summary

	// SubmissionTimestamp tracks the time at which each batch was confirmed.
	SubmissionTimestamp() prometheus.Gauge

	// SubmissionGasUsedWei tracks the amount of gas used to submit each batch.
	SubmissionGasUsedWei() prometheus.Gauge

	// BatchsSubmitted tracks the total number of successful batch submissions.
	BatchesSubmitted() prometheus.Counter

	// FailedSubmissions tracks the total number of failed batch submissions.
	FailedSubmissions() prometheus.Counter

	// BatchTxBuildTimeMs tracks the duration it takes to construct a batch
	// transaction.
	BatchTxBuildTimeMs() prometheus.Gauge

	// BatchConfirmationTimeMs tracks the duration it takes to confirm a batch
	// transaction.
	BatchConfirmationTimeMs() prometheus.Gauge

	// BatchPruneCount tracks the number of times a batch of sequencer
	// transactions is pruned in order to meet the desired size requirements.
	BatchPruneCount() prometheus.Gauge
}
