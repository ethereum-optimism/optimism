package metrics

import (
	"testing"

	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/stretchr/testify/require"
)

func TestSyncTesterMetrics(t *testing.T) {
	m := NewMetrics("")

	require.NotEmpty(t, m.Document(), "sanity check there are generated metrics docs")

	version := "v3.4.5"
	m.RecordInfo(version)
	m.RecordUp()

	c := opmetrics.NewMetricChecker(t, m.Registry())

	prefix := Namespace + "_default_"

	record := c.FindByName(prefix + "up").FindByLabels(nil)
	require.Equal(t, 1.0, record.Gauge.GetValue())

	record = c.FindByName(prefix + "info").FindByLabels(map[string]string{"version": version})
	require.Equal(t, 1.0, record.Gauge.GetValue())
}
