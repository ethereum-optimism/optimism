package resources

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
)

func TestMetricsFanIn(t *testing.T) {
	namedCounterWithLabels := func(name string, labels map[string]string) prometheus.Counter {
		return prometheus.NewCounter(prometheus.CounterOpts{Name: name, ConstLabels: labels})
	}

	registryA := prometheus.NewRegistry()
	registryA.MustRegister(namedCounterWithLabels("apples", map[string]string{"color": "red"}))

	registryB := prometheus.NewRegistry()
	registryB.MustRegister(namedCounterWithLabels("apples", map[string]string{"color": "green"}))

	fanIn := NewMetricsFanIn(2)

	fanIn.SetMetricsRegistry("a", registryA)
	fanIn.SetMetricsRegistry("b", registryB)

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	fanIn.ServeHTTP(rec, req)
	res := rec.Result()
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", res.StatusCode)
	}
	body, _ := io.ReadAll(res.Body)
	want := strings.Join([]string{`# HELP apples `, `# TYPE apples counter`, `apples{color="green"} 0`, `apples{color="red"} 0`}, "\n") + "\n"
	require.Equal(t, want, string(body))
}
