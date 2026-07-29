package metrics

import (
	"testing"

	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/stretchr/testify/require"
)

func TestMetrics(test *testing.T) {
	procName := "acceptance_test"
	prefix := Namespace + "_" + procName + "_"

	expectedSequenceNumber := 1.0
	infoLabel := "test"
	expectedInfo := 1.0
	expectedUp := 1.0

	metrics := NewMetrics(procName)
	metrics.RecordL2Proposal(uint64(expectedSequenceNumber))
	metrics.RecordInfo(infoLabel)
	metrics.RecordUp()

	checker := opmetrics.NewMetricChecker(test, metrics.Registry())
	sequenceNumberMetric := checker.FindByName(prefix + "proposed_sequence_number").FindByLabels(nil).Gauge.GetValue()
	infoMetric := checker.FindByName(prefix + "info").FindByLabels(map[string]string{"version": infoLabel}).Gauge.GetValue()
	upMetric := checker.FindByName(prefix + "up").FindByLabels(nil).Gauge.GetValue()

	require.Equal(test, expectedSequenceNumber, sequenceNumberMetric)
	require.Equal(test, expectedInfo, infoMetric)
	require.Equal(test, expectedUp, upMetric)
}
