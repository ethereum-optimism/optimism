package metrics

import (
	"testing"

	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
)

func TestMetrics(test *testing.T) {
	procName := "acceptance_test"
	prefix := Namespace + "_" + procName + "_"

	metrics := NewMetrics(procName)
	metrics.RecordL2Proposal(1)
	metrics.RecordInfo("test")
	metrics.RecordUp()

	checker := opmetrics.NewMetricChecker(test, metrics.Registry())
	checker.FindByName(prefix + "proposed_sequence_number")
	checker.FindByName(prefix + "info")
	checker.FindByName(prefix + "up")
}
