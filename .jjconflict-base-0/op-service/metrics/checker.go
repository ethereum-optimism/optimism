package metrics

import (
	"encoding/json"

	"github.com/prometheus/client_golang/prometheus"
	gocl "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/require"
)

type MetricFamilyChecker struct {
	fam *gocl.MetricFamily
	t   require.TestingT
}

func hasLabel(labels []*gocl.LabelPair, k string, v string) bool {
	for _, lab := range labels {
		if lab.GetName() == k && lab.GetValue() == v {
			return true
		}
	}
	return false
}

func hasAllLabels(m *gocl.Metric, labels map[string]string) bool {
	for k, v := range labels {
		if !hasLabel(m.Label, k, v) {
			return false
		}
	}
	return true
}

// FindByLabels finds a metric that matches the given labels.
// If not found, it fails the test.
func (f *MetricFamilyChecker) FindByLabels(labels map[string]string) *gocl.Metric {
	var found *gocl.Metric
	for _, m := range f.fam.Metric {
		if hasAllLabels(m, labels) {
			require.Nil(f.t, found, "must not have already found another other metric with same labels")
			found = m
		}
	}
	require.NotNil(f.t, found, "cannot find metric with labels")
	return found
}

type MetricFamiliesChecker struct {
	families []*gocl.MetricFamily
	t        require.TestingT
}

// FindByName finds a metric family by name.
// If not found, it fails the test.
func (m *MetricFamiliesChecker) FindByName(name string) *MetricFamilyChecker {
	var found *gocl.MetricFamily
	for _, f := range m.families {
		if f.GetName() == name {
			require.Nil(m.t, found, "must not have already found another other metric family with same name")
			found = f
		}
	}
	require.NotNil(m.t, found, "cannot find metric with labels")
	return &MetricFamilyChecker{fam: found, t: m.t}
}

// Dump prints indented json-formatted metrics info, for easy debugging
func (m *MetricFamiliesChecker) Dump() string {
	outStr, _ := json.MarshalIndent(m.families, "  ", "  ")
	return string(outStr)
}

// NewMetricChecker is a util for testing of metrics,
// to gather and search the metrics for things needed in a test.
func NewMetricChecker(t require.TestingT, reg *prometheus.Registry) *MetricFamiliesChecker {
	families, err := reg.Gather()
	require.NoError(t, err, "must gather metrics")
	return &MetricFamiliesChecker{families: families, t: t}
}
