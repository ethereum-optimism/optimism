package zkproposer

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	devtestmetrics "github.com/ethereum-optimism/optimism/op-devstack/devtest/metrics"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/stretchr/testify/require"
)

type stubHTTP func(context.Context, string, url.Values, http.Header) (*http.Response, error)

type stubRuntime struct {
	metrics client.HTTP
}

func (r stubRuntime) MetricsClient() client.HTTP {
	return r.metrics
}

func (s stubHTTP) Get(ctx context.Context, path string, query url.Values, headers http.Header) (*http.Response, error) {
	return s(ctx, path, query, headers)
}

func metricsResponse(payload string) *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Status:     "200 OK",
		Body:       io.NopCloser(strings.NewReader(payload)),
	}
}

func metricsPayload(spawned, failures int) string {
	return fmt.Sprintf(
		"# TYPE %s gauge\n%s %d\n# TYPE %s gauge\n%s %d\n# TYPE %s gauge\n%s %d\n",
		defenseTasksSpawnedMetric,
		defenseTasksSpawnedMetric,
		spawned,
		peakConcurrentDefenseTasksMetric,
		peakConcurrentDefenseTasksMetric,
		spawned,
		gameProvingFailuresMetric,
		gameProvingFailuresMetric,
		failures,
	)
}

func newTestProposer(t *testing.T, stub stubHTTP, options ...devtestmetrics.Option) (*ZKProposer, *testlog.CapturingHandler) {
	logger, logs := testlog.CaptureLogger(t, slog.LevelInfo)
	return &ZKProposer{
		log:     logger,
		metrics: devtestmetrics.NewMetricsClient(stub, options...),
	}, logs
}

func TestVerifyStateMatchesExactValuesFromOneSnapshot(t *testing.T) {
	fetches := 0
	proposer, _ := newTestProposer(t, func(context.Context, string, url.Values, http.Header) (*http.Response, error) {
		fetches++
		return metricsResponse(metricsPayload(2, 0)), nil
	})

	err := proposer.verifyState(
		context.Background(),
		DefenseTasksSpawned(2),
		PeakConcurrentDefenseTasks(2),
		ProvingFailures(0),
	)

	require.NoError(t, err)
	require.Equal(t, 1, fetches)
}

func TestNewUsesRuntimeMetricsTransport(t *testing.T) {
	fetches := 0
	transport := stubHTTP(func(context.Context, string, url.Values, http.Header) (*http.Response, error) {
		fetches++
		return metricsResponse(metricsPayload(2, 0)), nil
	})
	proposer := New(devtest.SerialT(t), stubRuntime{metrics: transport})

	err := proposer.verifyState(context.Background(), DefenseTasksSpawned(2), ProvingFailures(0))

	require.NoError(t, err)
	require.Equal(t, 1, fetches)
}

func TestNewPreservesTenMinuteWaitBudget(t *testing.T) {
	proposer := New(devtest.SerialT(t), stubRuntime{metrics: stubHTTP(
		func(ctx context.Context, _ string, _ url.Values, _ http.Header) (*http.Response, error) {
			<-ctx.Done()
			return nil, ctx.Err()
		},
	)})
	synctest.Test(t, func(t *testing.T) {
		started := time.Now()

		err := proposer.verifyState(context.Background(), DefenseTasksSpawned(1))

		require.ErrorIs(t, err, context.DeadlineExceeded)
		require.Equal(t, 10*time.Minute, time.Since(started))
	})
}

func TestVerifyStateDoesNotComposeDifferentSnapshots(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		fetches := 0
		proposer, _ := newTestProposer(t, func(context.Context, string, url.Values, http.Header) (*http.Response, error) {
			fetches++
			if fetches%2 == 1 {
				return metricsResponse(metricsPayload(2, 1)), nil
			}
			return metricsResponse(metricsPayload(3, 0)), nil
		}, devtestmetrics.WithWaitTimeout(250*time.Millisecond))

		err := proposer.verifyState(context.Background(), DefenseTasksSpawned(2), ProvingFailures(0))

		require.ErrorIs(t, err, context.DeadlineExceeded)
		require.GreaterOrEqual(t, fetches, 2)
	})
}

func TestVerifyStateRetriesTransientFetchFailure(t *testing.T) {
	fetches := 0
	transientErr := errors.New("temporarily unavailable")
	proposer, _ := newTestProposer(t, func(context.Context, string, url.Values, http.Header) (*http.Response, error) {
		fetches++
		if fetches == 1 {
			return nil, transientErr
		}
		return metricsResponse(metricsPayload(2, 0)), nil
	})

	err := proposer.verifyState(context.Background(), DefenseTasksSpawned(2), ProvingFailures(0))

	require.NoError(t, err)
	require.Equal(t, 2, fetches)
}

func TestVerifyStateValidatesExpectationsBeforeFetching(t *testing.T) {
	fetches := 0
	proposer, _ := newTestProposer(t, func(context.Context, string, url.Values, http.Header) (*http.Response, error) {
		fetches++
		return metricsResponse(metricsPayload(2, 0)), nil
	})

	err := proposer.verifyState(context.Background())
	require.ErrorContains(t, err, "at least one")
	err = proposer.verifyState(context.Background(), nil)
	require.ErrorContains(t, err, "expectation 0 is empty")
	err = proposer.verifyState(context.Background(), &StateExpectation{})
	require.ErrorContains(t, err, "expectation 0 is empty")
	require.Zero(t, fetches)
}

func TestVerifyStateRejectsDisabledMetricsBeforeFetching(t *testing.T) {
	proposer := New(devtest.SerialT(t), stubRuntime{})

	err := proposer.verifyState(context.Background(), DefenseTasksSpawned(2))

	require.EqualError(t, err,
		"ZK proposer metrics are disabled; pass presets.WithZKProposerOption(sysgo.WithZKMetrics()) when creating the preset")
}

func TestVerifyStateLogsExpectationsAndObservations(t *testing.T) {
	proposer, logs := newTestProposer(t, func(context.Context, string, url.Values, http.Header) (*http.Response, error) {
		return metricsResponse(metricsPayload(2, 0)), nil
	})

	err := proposer.verifyState(context.Background(), DefenseTasksSpawned(2))

	require.NoError(t, err)
	require.NotNil(t, logs.FindLog(
		testlog.NewMessageFilter("Observed ZK proposer state"),
		testlog.NewAttributesFilter("expectation", "defense tasks spawned"),
		testlog.NewAttributesFilter("expected", "2"),
		testlog.NewAttributesFilter("observed", "2"),
	))
}

func TestVerifyStateFailureIncludesLastMetricsPayload(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		payload := metricsPayload(1, 0)
		proposer, _ := newTestProposer(t, func(context.Context, string, url.Values, http.Header) (*http.Response, error) {
			return metricsResponse(payload), nil
		}, devtestmetrics.WithWaitTimeout(time.Millisecond))

		err := proposer.verifyState(context.Background(), DefenseTasksSpawned(2))

		require.ErrorContains(t, err, "defense tasks spawned expected 2 but observed 1")
		require.ErrorContains(t, err, "metrics payload:")
		require.ErrorContains(t, err, payload)
	})
}
