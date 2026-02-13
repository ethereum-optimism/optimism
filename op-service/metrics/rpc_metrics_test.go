package metrics

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	gocl "github.com/prometheus/client_model/go"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/rpc"
)

type testMessage struct {
	id     json.RawMessage
	method string
	params json.RawMessage
	err    *rpc.JsonError
	result json.RawMessage
}

func (m *testMessage) MsgIsNotification() bool {
	return len(m.id) == 0
}

func (m *testMessage) MsgIsResponse() bool {
	return len(m.params) == 0
}

func (m *testMessage) MsgID() json.RawMessage {
	return m.id
}

func (m *testMessage) MsgMethod() string {
	return m.method
}

func (m *testMessage) MsgParams() json.RawMessage {
	return m.params
}

func (m *testMessage) MsgError() *rpc.JsonError {
	return m.err
}

func (m *testMessage) MsgResult() json.RawMessage {
	return m.result
}

var _ rpc.RecordedMsg = (*testMessage)(nil)

func TestRPCMetrics(t *testing.T) {
	reg := NewRegistry()
	factory := With(reg)
	m := MakeRPCMetrics("testservice", factory)
	rec := m.NewRecorder("foobar")

	ctx := context.Background()

	// Incoming request / response
	reqIn := &testMessage{
		method: "test_helloIn",
		id:     json.RawMessage(`123`),
		params: json.RawMessage(`[42, "hello", "world"]`),
	}
	onDone := rec.RecordIncoming(ctx, reqIn)
	respIn := &testMessage{
		id:     reqIn.id,
		result: json.RawMessage(`"echo"`),
	}
	onDone(ctx, reqIn, respIn)

	// Incoming request / response with error
	onDone = rec.RecordIncoming(ctx, reqIn)
	respInErr := &testMessage{
		id:     reqIn.id,
		result: nil,
		err:    &rpc.JsonError{Code: -123},
	}
	onDone(ctx, reqIn, respInErr)

	// Outgoing request / response
	reqOut := &testMessage{
		method: "test_helloOut",
		id:     json.RawMessage(`42`),
		params: json.RawMessage(`[42, "hello", "world"]`),
	}
	respOut := &testMessage{
		id:     reqOut.id,
		result: json.RawMessage(`"echo"`),
	}
	onDone = rec.RecordOutgoing(ctx, reqIn)
	onDone(ctx, reqOut, respOut)

	// Outgoing request / response with error
	onDone = rec.RecordOutgoing(ctx, reqIn)
	respOutErr := &testMessage{
		id:     reqOut.id,
		result: nil,
		err:    &rpc.JsonError{Code: -42},
	}
	onDone(ctx, reqOut, respOutErr)

	// Incoming notification
	notificationIn := &testMessage{
		method: "test_notifyIn",
		params: json.RawMessage(`["hello"]`),
	}
	onDone = rec.RecordIncoming(ctx, notificationIn)
	require.Nil(t, onDone)

	// Outgoing notification
	notificationOut := &testMessage{
		method: "test_notifyOut",
		params: json.RawMessage(`["hello"]`),
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
			require.Equal(t, reqIn.method, labels["method"])
		case "server_request_duration_seconds":
			require.NotZero(t, entry.Histogram.GetSampleSum())
			require.Equal(t, reqIn.method, labels["method"])
		case "server_requests_total":
			require.EqualValues(t, 2, entry.Counter.GetValue())
			require.Equal(t, reqIn.method, labels["method"])
		case "server_responses_total":
			require.EqualValues(t, 1, entry.Counter.GetValue())
			require.EqualValues(t, 1, entryErr.Counter.GetValue())
			require.Equal(t, reqIn.method, labels["method"])
			require.Equal(t, "rpc_-123", labelsErr["error"])
		case "server_results_size_total":
			require.NotZero(t, entry.Counter.GetValue())
			require.Equal(t, reqIn.method, labels["method"])
		case "server_notifications_sent_total":
			require.EqualValues(t, 1, entry.Counter.GetValue())
			require.Equal(t, notificationOut.method, labels["method"])
		case "client_notifications_received_total":
			require.EqualValues(t, 1, entry.Counter.GetValue())
			require.Equal(t, notificationIn.method, labels["method"])
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
