package metrics

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	gocl "github.com/prometheus/client_model/go"

	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-service/jsonrpc"
)

type describingRegisterer struct {
	descs []string
}

func (r *describingRegisterer) Register(c prometheus.Collector) error {
	ch := make(chan *prometheus.Desc, 1)
	go func() {
		c.Describe(ch)
		close(ch)
	}()
	for d := range ch {
		r.descs = append(r.descs, d.String())
	}
	return nil
}

func (r *describingRegisterer) MustRegister(cs ...prometheus.Collector) {
	for _, c := range cs {
		_ = r.Register(c)
	}
}

func (r *describingRegisterer) Unregister(prometheus.Collector) bool { return false }

// TestRPCMetricsDescriptors pins the RPC metric names and label sets, which dashboards and alerts depend on.
func TestRPCMetricsDescriptors(t *testing.T) {
	reg := new(describingRegisterer)
	MakeRPCMetrics("ns", With(reg))
	desc := func(name, help, labels string) string {
		return fmt.Sprintf("Desc{fqName: %q, help: %q, constLabels: {}, variableLabels: {%s}}", name, help, labels)
	}
	require.ElementsMatch(t, []string{
		desc("ns_rpc_client_requests_total", "Total RPC requests initiated", "rpc,method"),
		desc("ns_rpc_client_request_duration_seconds", "Histogram of RPC client request durations", "rpc,method"),
		desc("ns_rpc_client_responses_total", "Total RPC request responses received", "rpc,method,error"),
		desc("ns_rpc_server_requests_total", "Total requests to the RPC server", "rpc,method"),
		desc("ns_rpc_server_request_duration_seconds", "Histogram of RPC server request durations", "rpc,method"),
		desc("ns_rpc_server_responses_total", "Total RPC request responses served", "rpc,method,error"),
		desc("ns_rpc_client_notifications_received_total", "Total RPC notifications received", "rpc,method"),
		desc("ns_rpc_server_notifications_sent_total", "Total RPC notifications sent", "rpc,method"),
		desc("ns_rpc_client_params_size_total", "Total bytes of RPC params sent", "rpc,method"),
		desc("ns_rpc_client_results_size_total", "Total bytes of RPC results received", "rpc,method"),
		desc("ns_rpc_server_params_size_total", "Total bytes of RPC params received", "rpc,method"),
		desc("ns_rpc_server_results_size_total", "Total bytes of RPC results sent back", "rpc,method"),
	}, reg.descs)
}

func TestRPCMetrics(t *testing.T) {
	reg := NewRegistry()
	factory := With(reg)
	m := MakeRPCMetrics("testservice", factory)
	rec := m.NewRecorder("foobar")

	ctx := context.Background()

	// Incoming request / response
	reqIn := jsonrpc.Message{
		Method: "test_helloIn",
		Params: json.RawMessage(`[42, "hello", "world"]`),
	}
	onDone := rec.RecordIncoming(ctx, reqIn)
	onDone(ctx, jsonrpc.Response{Result: json.RawMessage(`"echo"`)})

	// Incoming request / response with error
	onDone = rec.RecordIncoming(ctx, reqIn)
	onDone(ctx, jsonrpc.Response{Error: &jsonrpc.Error{Code: -123}})

	// Outgoing request / response
	reqOut := jsonrpc.Message{
		Method: "test_helloOut",
		Params: json.RawMessage(`[42, "hello", "world"]`),
	}
	onDone = rec.RecordOutgoing(ctx, reqOut)
	onDone(ctx, jsonrpc.Response{Result: json.RawMessage(`"echo"`)})

	// Outgoing request / response with error
	onDone = rec.RecordOutgoing(ctx, reqOut)
	onDone(ctx, jsonrpc.Response{Error: &jsonrpc.Error{Code: -42}})

	// Incoming notification
	notificationIn := jsonrpc.Message{
		Method:       "test_notifyIn",
		Params:       json.RawMessage(`["hello"]`),
		Notification: true,
	}
	onDone = rec.RecordIncoming(ctx, notificationIn)
	require.Nil(t, onDone)

	// Outgoing notification
	notificationOut := jsonrpc.Message{
		Method:       "test_notifyOut",
		Params:       json.RawMessage(`["hello"]`),
		Notification: true,
	}
	onDone = rec.RecordOutgoing(ctx, notificationOut)
	require.Nil(t, onDone)

	data, err := reg.Gather()
	require.NoError(t, err)

	// To dump the data for debugging:
	//outStr, _ := json.MarshalIndent(data, "  ", "  ")
	//t.Log(string(outStr))

	for _, d := range data {
		name := *d.Name
		if !strings.HasPrefix(name, "testservice_rpc_") {
			continue
		}
		name = strings.TrimPrefix(name, "testservice_rpc_")

		// Find the non-error metric
		// And, if there was an RPC error, also the that metric
		var entry *gocl.Metric
		var entryErr *gocl.Metric
		for _, e := range d.Metric {
			for _, l := range e.Label {
				if l.GetName() == "error" && l.GetValue() == "<nil>" {
					entry = e
				} else {
					entryErr = e
				}
			}
		}
		if entry == nil {
			entry = d.Metric[0]
		}
		labels := make(map[string]string)
		if entry != nil {
			for _, label := range entry.Label {
				labels[label.GetName()] = label.GetValue()
			}
		}
		labelsErr := make(map[string]string)
		if entryErr != nil {
			for _, label := range entryErr.Label {
				labelsErr[label.GetName()] = label.GetValue()
			}
		}
		require.Equal(t, "foobar", labels["rpc"])

		switch name {
		case "server_params_size_total":
			require.NotZero(t, entry.Counter.GetValue())
			require.Equal(t, reqIn.Method, labels["method"])
		case "server_request_duration_seconds":
			require.NotZero(t, entry.Histogram.GetSampleSum())
			require.Equal(t, reqIn.Method, labels["method"])
		case "server_requests_total":
			require.EqualValues(t, 2, entry.Counter.GetValue())
			require.Equal(t, reqIn.Method, labels["method"])
		case "server_responses_total":
			require.EqualValues(t, 1, entry.Counter.GetValue())
			require.EqualValues(t, 1, entryErr.Counter.GetValue())
			require.Equal(t, reqIn.Method, labels["method"])
			require.Equal(t, "rpc_-123", labelsErr["error"])
		case "server_results_size_total":
			require.NotZero(t, entry.Counter.GetValue())
			require.Equal(t, reqIn.Method, labels["method"])
		case "server_notifications_sent_total":
			require.EqualValues(t, 1, entry.Counter.GetValue())
			require.Equal(t, notificationOut.Method, labels["method"])
		case "client_notifications_received_total":
			require.EqualValues(t, 1, entry.Counter.GetValue())
			require.Equal(t, notificationIn.Method, labels["method"])
		case "client_params_size_total":
			require.NotZero(t, entry.Counter.GetValue())
		case "client_request_duration_seconds":
			require.NotZero(t, entry.Histogram.GetSampleSum())
		case "client_requests_total":
			require.EqualValues(t, 2, entry.Counter.GetValue())
		case "client_responses_total":
			require.EqualValues(t, 1, entry.Counter.GetValue())
			require.EqualValues(t, 1, entryErr.Counter.GetValue())
			require.Equal(t, "rpc_-42", labelsErr["error"])
		case "client_results_size_total":
			require.NotZero(t, entry.Counter.GetValue())
		default:
			t.Error("unrecognized rpc metric", name)
		}
	}
}
