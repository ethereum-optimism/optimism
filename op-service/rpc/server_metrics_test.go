package rpc

import (
	"context"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum-optimism/optimism/op-service/jsonrpc"
	"github.com/ethereum-optimism/optimism/op-service/log"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

type metricsTestAPI struct{}

func (a *metricsTestAPI) Echo(n int) int { return n }

func (a *metricsTestAPI) Big(n int) string { return strings.Repeat("x", n) }

func (a *metricsTestAPI) Fail() error {
	return &jsonrpc.Error{Code: -39001, Message: "boom", Data: "details"}
}

func (a *metricsTestAPI) Count(ctx context.Context, n int) (*rpc.Subscription, error) {
	notifier, ok := rpc.NotifierFromContext(ctx)
	if !ok {
		return nil, rpc.ErrNotificationsUnsupported
	}
	sub := notifier.CreateSubscription()
	go func() {
		for i := range n {
			if err := notifier.Notify(sub.ID, i); err != nil {
				return
			}
		}
	}()
	return sub, nil
}

// gatherValues returns every sample of the registry's RPC metrics, keyed by metric name and
// labels in name order; histograms by their sample count.
func gatherValues(t *testing.T, reg *prometheus.Registry) map[string]float64 {
	families, err := reg.Gather()
	require.NoError(t, err)
	values := make(map[string]float64)
	for _, fam := range families {
		if !strings.HasPrefix(fam.GetName(), "ns_rpc_") {
			continue
		}
		for _, m := range fam.GetMetric() {
			key := strings.TrimPrefix(fam.GetName(), "ns_rpc_")
			for _, l := range m.GetLabel() {
				key += " " + l.GetName() + "=" + l.GetValue()
			}
			switch {
			case m.Counter != nil:
				values[key] = m.GetCounter().GetValue()
			case m.Histogram != nil:
				values[key] = float64(m.GetHistogram().GetSampleCount())
			}
		}
	}
	return values
}

// TestServerRPCMetrics checks the server-side RPC metrics that the handler records for the
// JSON-RPC traffic it serves over HTTP.
//
// It also pins a go-ethereum implementation detail that recording relies on: the rpc.Server
// writes each HTTP response, single or batch, in one Write call, which the recording middleware
// parses before passing it on. If a go-ethereum update breaks this, the response counts here drop
// to zero and the middleware needs to buffer the whole response instead.
func TestServerRPCMetrics(t *testing.T) {
	reg := opmetrics.NewRegistry()
	m := opmetrics.MakeRPCMetrics("ns", opmetrics.With(reg))
	server := ServerFromConfig(&ServerConfig{
		RpcOptions: []Option{
			WithLogger(testlog.Logger(t, log.LevelInfo)),
			WithWebsocketEnabled(),
			WithRPCRecorder(m.NewRecorder("main")),
		},
		Host:       "127.0.0.1",
		Port:       0,
		AppVersion: "test",
	})
	server.AddAPI(rpc.API{Namespace: "test", Service: new(metricsTestAPI)})
	require.NoError(t, server.Start())
	t.Cleanup(func() { require.NoError(t, server.Stop()) })

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	httpCl, err := rpc.DialContext(ctx, "http://"+server.Endpoint())
	require.NoError(t, err)
	defer httpCl.Close()
	var n int
	require.NoError(t, httpCl.CallContext(ctx, &n, "test_echo", 3))
	require.Error(t, httpCl.CallContext(ctx, nil, "test_fail"))
	require.Error(t, httpCl.CallContext(ctx, nil, "test_unknown"))
	batch := []rpc.BatchElem{
		{Method: "test_echo", Args: []any{3}, Result: new(int)},
		{Method: "test_fail", Result: new(any)},
	}
	require.NoError(t, httpCl.BatchCallContext(ctx, batch))
	// A large response is written in one call too.
	var big string
	require.NoError(t, httpCl.CallContext(ctx, &big, "test_big", 6<<20))
	require.Len(t, big, 6<<20)

	post := func(body io.Reader) (int, string) {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://"+server.Endpoint(), body)
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		respBody, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		return resp.StatusCode, string(respBody)
	}
	// A notification, and a batch with duplicate IDs that geth re-encodes in its response.
	post(strings.NewReader(`{"jsonrpc":"2.0","method":"test_echo","params":[5]}`))
	status, body := post(strings.NewReader(`[{"jsonrpc":"2.0","id":"<a>","method":"test_echo","params":[3]},` +
		`{"jsonrpc":"2.0","id":"<a>","method":"test_echo","params":[3]}]`))
	require.Equal(t, http.StatusOK, status)
	require.Equal(t, 2, strings.Count(body, `"result":3`), body)
	// A batch mixing a call and a notification.
	status, body = post(strings.NewReader(`[{"jsonrpc":"2.0","id":9,"method":"test_fail"},` +
		`{"jsonrpc":"2.0","method":"test_fail"}]`))
	require.Equal(t, http.StatusOK, status)
	require.Contains(t, body, `"code":-39001`)
	// A request the server rejects for its content type counts, but gets no response.
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://"+server.Endpoint(),
		strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"test_unknown"}`))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "text/plain")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	require.Equal(t, http.StatusUnsupportedMediaType, resp.StatusCode)
	status, body = post(strings.NewReader(`{"jsonrpc":`))
	require.Equal(t, http.StatusOK, status)
	require.Contains(t, body, `"code":-32700`)
	// An oversized body of unknown length, which the server truncates.
	status, body = post(io.MultiReader(strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"test_echo","params":["`),
		strings.NewReader(strings.Repeat("x", 6<<20)), strings.NewReader(`"]}`)))
	require.Equal(t, http.StatusOK, status)
	require.Contains(t, body, `"code":-32700`)

	// Websocket traffic is not recorded.
	wsCl, err := rpc.DialContext(ctx, "ws://"+server.Endpoint())
	require.NoError(t, err)
	defer wsCl.Close()
	ch := make(chan int)
	sub, err := wsCl.Subscribe(ctx, "test", ch, "count", 2)
	require.NoError(t, err)
	defer sub.Unsubscribe()
	for range 2 {
		select {
		case <-ch:
		case err := <-sub.Err():
			t.Fatalf("subscription failed: %v", err)
		case <-ctx.Done():
			t.Fatal("timed out waiting for notification")
		}
	}
	require.NoError(t, wsCl.CallContext(ctx, &n, "test_echo", 4))

	values := gatherValues(t, reg)
	var requests float64
	for key, v := range values {
		if strings.HasPrefix(key, "server_requests_total ") {
			requests += v
		}
	}
	require.Equal(t, 10.0, requests, "malformed, oversized, notification and websocket requests are not counted")
	for key, want := range map[string]float64{
		"server_requests_total method=test_echo rpc=main":                      4,
		"server_requests_total method=test_fail rpc=main":                      3,
		"server_requests_total method=test_unknown rpc=main":                   2,
		"server_request_duration_seconds method=test_echo rpc=main":            4,
		"server_responses_total error=<nil> method=test_echo rpc=main":         4,
		"server_responses_total error=rpc_-39001 method=test_fail rpc=main":    3,
		"server_responses_total error=rpc_-32601 method=test_unknown rpc=main": 1,
		"server_params_size_total method=test_echo rpc=main":                   12, // [3]
		"server_results_size_total method=test_echo rpc=main":                  4,  // 3
		"server_requests_total method=test_big rpc=main":                       1,
		"server_responses_total error=<nil> method=test_big rpc=main":          1,
		"client_notifications_received_total method=test_fail rpc=main":        1,
		"client_notifications_received_total method=test_echo rpc=main":        1,
	} {
		require.Equal(t, want, values[key], key)
	}
	for key := range values {
		require.NotContains(t, key, "notifications_sent", "websocket notifications are not recorded")
	}
}

func TestWebsocketRecordingWarning(t *testing.T) {
	m := opmetrics.MakeRPCMetrics("ns", opmetrics.With(opmetrics.NewRegistry()))
	rec := m.NewRecorder("main")
	warning := testlog.NewMessageContainsFilter("websocket traffic is not recorded")

	lgr, logs := testlog.CaptureLogger(t, log.LevelWarn)
	NewHandler("test", WithLogger(lgr), WithWebsocketEnabled(), WithRPCRecorder(rec))
	require.NotNil(t, logs.FindLog(warning))

	lgr, logs = testlog.CaptureLogger(t, log.LevelWarn)
	NewHandler("test", WithLogger(lgr), WithRPCRecorder(rec))
	require.Nil(t, logs.FindLog(warning))
}

type notificationRecorder struct {
	dones atomic.Int32
}

func (r *notificationRecorder) RecordIncoming(context.Context, jsonrpc.Message) jsonrpc.RecordDone {
	return func(context.Context, jsonrpc.Response) { r.dones.Add(1) }
}

func (r *notificationRecorder) RecordOutgoing(context.Context, jsonrpc.Message) jsonrpc.RecordDone {
	return nil
}

// TestServerRecordDoneNotCalledForNotifications checks that a RecordDone returned for a
// notification is never called, as notifications get no response.
func TestServerRecordDoneNotCalledForNotifications(t *testing.T) {
	rec := new(notificationRecorder)
	server := ServerFromConfig(&ServerConfig{
		RpcOptions: []Option{
			WithLogger(testlog.Logger(t, log.LevelInfo)),
			WithRPCRecorder(rec),
		},
		Host:       "127.0.0.1",
		Port:       0,
		AppVersion: "test",
	})
	server.AddAPI(rpc.API{Namespace: "test", Service: new(metricsTestAPI)})
	require.NoError(t, server.Start())
	t.Cleanup(func() { require.NoError(t, server.Stop()) })

	for _, body := range []string{
		`{"jsonrpc":"2.0","method":"test_echo","params":[1]}`,
		`{"jsonrpc":"2.0","id":1,"method":"test_echo","params":[1]}`,
	} {
		resp, err := http.Post("http://"+server.Endpoint(), "application/json", strings.NewReader(body))
		require.NoError(t, err)
		require.NoError(t, resp.Body.Close())
	}
	require.EqualValues(t, 1, rec.dones.Load(), "only the call's response is recorded")
}

// TestServerRPCMetricsUnauthenticated checks that requests the JWT check rejects are not
// recorded, so unauthenticated callers cannot add metric series.
func TestServerRPCMetricsUnauthenticated(t *testing.T) {
	reg := opmetrics.NewRegistry()
	m := opmetrics.MakeRPCMetrics("ns", opmetrics.With(reg))
	server := ServerFromConfig(&ServerConfig{
		RpcOptions: []Option{
			WithLogger(testlog.Logger(t, log.LevelInfo)),
			WithJWTSecret(make([]byte, 32)),
			WithRPCRecorder(m.NewRecorder("main")),
		},
		Host:       "127.0.0.1",
		Port:       0,
		AppVersion: "test",
	})
	require.NoError(t, server.Start())
	t.Cleanup(func() { require.NoError(t, server.Stop()) })

	resp, err := http.Post("http://"+server.Endpoint(), "application/json",
		strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"evil_method"}`))
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	require.Empty(t, gatherValues(t, reg))
}
