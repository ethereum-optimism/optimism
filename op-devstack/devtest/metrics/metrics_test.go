package metrics

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type stubHTTP func(context.Context, string, url.Values, http.Header) (*http.Response, error)

func (s stubHTTP) Get(ctx context.Context, path string, query url.Values, headers http.Header) (*http.Response, error) {
	return s(ctx, path, query, headers)
}

func metricsResponse(status int, payload string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Status:     fmt.Sprintf("%d %s", status, http.StatusText(status)),
		Body:       io.NopCloser(strings.NewReader(payload)),
	}
}

type errorReader struct {
	err error
}

func (r errorReader) Read([]byte) (int, error) {
	return 0, r.err
}

func TestMetricsClientFetchParsesPrometheusResponse(t *testing.T) {
	payload := "# TYPE first gauge\nfirst 1\n# TYPE second counter\nsecond 2\n"
	stub := stubHTTP(func(ctx context.Context, path string, query url.Values, headers http.Header) (*http.Response, error) {
		require.Equal(t, "/metrics", path)
		require.Nil(t, query)
		require.Nil(t, headers)
		return metricsResponse(http.StatusOK, payload), nil
	})

	snapshot, err := NewMetricsClient(stub, WithFetchTimeout(time.Second)).Fetch(context.Background())
	require.NoError(t, err)
	require.Equal(t, payload, snapshot.Payload())
	require.Len(t, snapshot.families, 2)
}

func TestMetricsClientFetchRejectsNilHTTPClient(t *testing.T) {
	_, err := NewMetricsClient(nil).Fetch(context.Background())

	require.ErrorContains(t, err, "HTTP client")
}

func TestMetricsClientFetchRejectsNonPositiveTimeout(t *testing.T) {
	for _, timeout := range []time.Duration{0, -time.Second} {
		t.Run(timeout.String(), func(t *testing.T) {
			fetches := 0
			stub := stubHTTP(func(context.Context, string, url.Values, http.Header) (*http.Response, error) {
				fetches++
				return metricsResponse(http.StatusOK, ""), nil
			})

			_, err := NewMetricsClient(stub, WithFetchTimeout(timeout)).Fetch(context.Background())

			require.ErrorContains(t, err, "fetch timeout")
			require.ErrorContains(t, err, "must be positive")
			require.Zero(t, fetches)
		})
	}
}

func TestMetricsClientWaitForGaugeRejectsNonPositivePollInterval(t *testing.T) {
	for _, pollInterval := range []time.Duration{0, -time.Second} {
		t.Run(pollInterval.String(), func(t *testing.T) {
			fetches := 0
			stub := stubHTTP(func(context.Context, string, url.Values, http.Header) (*http.Response, error) {
				fetches++
				return metricsResponse(http.StatusOK, ""), nil
			})

			err := NewMetricsClient(stub).WaitForGauge(
				context.Background(),
				GaugeDefinition{Name: "target_metric", Expected: 1},
				pollInterval,
			)

			require.ErrorContains(t, err, "poll interval")
			require.ErrorContains(t, err, "must be positive")
			require.Zero(t, fetches)
		})
	}
}

func TestMetricsClientFetchErrors(t *testing.T) {
	fetchErr := errors.New("boom")
	readErr := errors.New("read failed")
	tests := []struct {
		name     string
		stub     stubHTTP
		contains []string
		cause    error
	}{
		{
			name: "request failure",
			stub: func(context.Context, string, url.Values, http.Header) (*http.Response, error) {
				return nil, fetchErr
			},
			contains: []string{"fetch metrics", "boom"},
			cause:    fetchErr,
		},
		{
			name: "unexpected HTTP status",
			stub: func(context.Context, string, url.Values, http.Header) (*http.Response, error) {
				return metricsResponse(http.StatusServiceUnavailable, "unavailable"), nil
			},
			contains: []string{"HTTP 503", "unavailable"},
		},
		{
			name: "malformed response",
			stub: func(context.Context, string, url.Values, http.Header) (*http.Response, error) {
				return metricsResponse(http.StatusOK, "not prometheus"), nil
			},
			contains: []string{"parse metrics response"},
		},
		{
			name: "response read failure",
			stub: func(context.Context, string, url.Values, http.Header) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Status:     "200 OK",
					Body:       io.NopCloser(errorReader{err: readErr}),
				}, nil
			},
			contains: []string{"read metrics response", "read failed"},
			cause:    readErr,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := NewMetricsClient(test.stub).Fetch(context.Background())
			require.Error(t, err)
			for _, expected := range test.contains {
				require.ErrorContains(t, err, expected)
			}
			if test.cause != nil {
				require.ErrorIs(t, err, test.cause)
			}
		})
	}
}

func TestSnapshotGaugeSelectsLabels(t *testing.T) {
	snapshot, err := parseSnapshot(
		"# TYPE op_dispute_mon_games gauge\n" +
			"op_dispute_mon_games{game_type=\"super-cannon-kona\",chain=\"a\"} 2\n" +
			"op_dispute_mon_games{game_type=\"super-permissioned\",chain=\"a\"} 1\n",
	)
	require.NoError(t, err)

	value, err := snapshot.Gauge(
		"op_dispute_mon_games",
		map[string]string{"game_type": "super-permissioned"},
	)
	require.NoError(t, err)
	require.Equal(t, float64(1), value)
}

func TestSnapshotGaugeErrors(t *testing.T) {
	tests := []struct {
		name          string
		payload       string
		metric        string
		labels        map[string]string
		errorContains string
	}{
		{
			name:          "missing family",
			payload:       "# TYPE another_metric gauge\nanother_metric 1\n",
			metric:        "missing_metric",
			errorContains: "metric family missing_metric not found",
		},
		{
			name:          "missing requested labels",
			payload:       "# TYPE op_dispute_mon_games gauge\nop_dispute_mon_games{chain=\"a\"} 1\n",
			metric:        "op_dispute_mon_games",
			labels:        map[string]string{"chain": "b"},
			errorContains: "metric op_dispute_mon_games with labels map[chain:b] not found",
		},
		{
			name: "duplicate subset matches",
			payload: "# TYPE op_dispute_mon_games gauge\n" +
				"op_dispute_mon_games{game_type=\"permissioned\",chain=\"a\"} 1\n" +
				"op_dispute_mon_games{game_type=\"permissioned\",chain=\"b\"} 2\n",
			metric:        "op_dispute_mon_games",
			labels:        map[string]string{"game_type": "permissioned"},
			errorContains: "metric op_dispute_mon_games with labels map[game_type:permissioned] matched multiple series",
		},
		{
			name:          "counter requested as gauge",
			payload:       "# TYPE request_count counter\nrequest_count 1\n",
			metric:        "request_count",
			errorContains: "metric request_count is not a gauge",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			snapshot, err := parseSnapshot(test.payload)
			require.NoError(t, err)

			_, err = snapshot.Gauge(test.metric, test.labels)
			require.ErrorContains(t, err, test.errorContains)
		})
	}
}

func TestMetricsClientWaitForGauge(t *testing.T) {
	values := []int{0, 0, 1}
	fetches := 0
	stub := stubHTTP(func(context.Context, string, url.Values, http.Header) (*http.Response, error) {
		value := values[fetches]
		fetches++
		payload := fmt.Sprintf(
			"# TYPE target_metric gauge\ntarget_metric{state=\"ready\"} %d\n",
			value,
		)
		return metricsResponse(http.StatusOK, payload), nil
	})
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	err := NewMetricsClient(stub).WaitForGauge(ctx, GaugeDefinition{
		Name:     "target_metric",
		Labels:   map[string]string{"state": "ready"},
		Expected: 1,
	}, time.Millisecond)

	require.NoError(t, err)
	require.Equal(t, 3, fetches)
}

func TestMetricsClientWaitForGaugeCancellationIncludesLastObservation(t *testing.T) {
	parentCtx, stop := context.WithTimeout(context.Background(), time.Second)
	defer stop()
	ctx, cancel := context.WithCancel(parentCtx)
	fetches := 0
	stub := stubHTTP(func(fetchCtx context.Context, _ string, _ url.Values, _ http.Header) (*http.Response, error) {
		fetches++
		if fetches == 2 {
			cancel()
			return nil, fetchCtx.Err()
		}
		payload := fmt.Sprintf(
			"# attempt %d\n# TYPE target_metric gauge\ntarget_metric 0\n",
			fetches,
		)
		return metricsResponse(http.StatusOK, payload), nil
	})

	err := NewMetricsClient(stub).WaitForGauge(ctx, GaugeDefinition{
		Name:     "target_metric",
		Expected: 1,
	}, time.Millisecond)

	require.ErrorIs(t, err, context.Canceled)
	require.ErrorContains(t, err, "target_metric")
	require.ErrorContains(t, err, "expected 1")
	require.ErrorContains(t, err, "observed 0")
	require.ErrorContains(t, err, "# attempt 1\n# TYPE target_metric gauge\ntarget_metric 0\n")
	require.Equal(t, 2, fetches)
}
