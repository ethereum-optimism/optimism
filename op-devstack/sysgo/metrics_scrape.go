package sysgo

import (
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/prometheus/common/expfmt"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
)

// ScrapeCounterSum reads one Prometheus counter family off a metrics endpoint and returns the sum
// over its label series.
//
// The endpoint MUST answer and the payload MUST parse — a scrape that quietly returned zero on a
// connection error would turn "this counter is zero" into "we could not tell", which is the whole
// value of asserting on a counter in the first place. An ABSENT family, by contrast, really is zero:
// a prometheus CounterVec emits no series until something calls WithLabelValues on it, so a counter
// that has never been touched has no line in the payload.
func ScrapeCounterSum(t devtest.T, metricsURL string, name string) float64 {
	require := t.Require()
	require.NotEmpty(metricsURL, "no metrics endpoint to scrape %s from", name)

	url := strings.TrimSuffix(metricsURL, "/") + "/metrics"
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	require.NoErrorf(err, "scrape %s", url)
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	require.NoErrorf(err, "read %s", url)
	require.Equalf(http.StatusOK, resp.StatusCode, "scrape %s returned %s: %s", url, resp.Status, body)

	families, err := (&expfmt.TextParser{}).TextToMetricFamilies(strings.NewReader(string(body)))
	require.NoErrorf(err, "parse metrics from %s", url)

	family, ok := families[name]
	if !ok {
		return 0
	}
	var sum float64
	for _, m := range family.Metric {
		if c := m.GetCounter(); c != nil {
			sum += c.GetValue()
		}
	}
	return sum
}
