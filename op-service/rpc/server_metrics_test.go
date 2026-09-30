package rpc

import (
	"context"
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
// traffic it serves.
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

	values := gatherValues(t, reg)
	for key, want := range map[string]float64{
		"server_requests_total method=test_echo rpc=main":                      2,
		"server_requests_total method=test_fail rpc=main":                      2,
		"server_requests_total method=test_unknown rpc=main":                   1,
		"server_requests_total method=test_subscribe rpc=main":                 1,
		"server_request_duration_seconds method=test_echo rpc=main":            2,
		"server_responses_total error=<nil> method=test_echo rpc=main":         2,
		"server_responses_total error=rpc_-39001 method=test_fail rpc=main":    2,
		"server_responses_total error=rpc_-32601 method=test_unknown rpc=main": 1,
		"server_responses_total error=<nil> method=test_subscribe rpc=main":    1,
		"server_params_size_total method=test_echo rpc=main":                   6, // [3]
		"server_results_size_total method=test_echo rpc=main":                  2, // 3
		"server_notifications_sent_total method=test_subscription rpc=main":    2,
	} {
		require.Equal(t, want, values[key], key)
	}
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
