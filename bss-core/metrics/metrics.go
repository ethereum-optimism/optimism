package metrics

import (
	"fmt"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type Base struct {
	// subsystemName stores the name that will prefix all metrics. This can be
	// used by drivers to further extend the core metrics and ensure they use the
	// same prefix.
	subsystemName string

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
}

func NewBase(serviceName, subServiceName string) *Base {
	subsystem := MakeSubsystemName(serviceName, subServiceName)
	return &Base{
		subsystemName: subsystem,
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
	}
}

// SubsystemName returns the subsystem name for the metrics group.
func (b *Base) SubsystemName() string {
	return b.subsystemName
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

// MakeSubsystemName builds the subsystem name for a group of metrics, which
// prometheus will use to prefix all metrics in the group. If two non-empty
// strings are provided, they are joined with an underscore. If only one
// non-empty string is provided, that name will be used alone. Otherwise an
// empty string is returned after converting the characters to lower case.
//
// NOTE: This method panics if spaces are included in either string.
func MakeSubsystemName(serviceName string, subServiceName string) string {
	var subsystem string
	switch {
	case serviceName != "" && subServiceName != "":
		subsystem = fmt.Sprintf("%s_%s", serviceName, subServiceName)
	case serviceName != "":
		subsystem = serviceName
	default:
		subsystem = subServiceName
	}

	if strings.ContainsAny(subsystem, " ") {
		panic(fmt.Sprintf("metrics name \"%s\"cannot have spaces", subsystem))
	}

	return strings.ToLower(subsystem)
}
