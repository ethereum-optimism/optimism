package rpc

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-service/jsonrpc"
)

type eventLog []string

type orderRecorder struct{ events *eventLog }

func (r orderRecorder) RecordIncoming(context.Context, jsonrpc.Message) jsonrpc.RecordDone {
	return func(context.Context, jsonrpc.Response) { *r.events = append(*r.events, "recorded") }
}

func (r orderRecorder) RecordOutgoing(context.Context, jsonrpc.Message) jsonrpc.RecordDone {
	return nil
}

type orderWriter struct {
	*httptest.ResponseRecorder
	events *eventLog
}

func (w orderWriter) Write(b []byte) (int, error) {
	*w.events = append(*w.events, "written")
	return w.ResponseRecorder.Write(b)
}

// TestRecordingBeforeResponseWrite checks that a response is recorded before the client gets
// it, so that recorded durations exclude writing the response.
func TestRecordingBeforeResponseWrite(t *testing.T) {
	const response = `{"jsonrpc":"2.0","id":1,"result":3}`
	var events eventLog
	handler := newRecordingHandler(orderRecorder{&events}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.ReadAll(r.Body)
		_, _ = w.Write([]byte(response))
	}))
	w := orderWriter{httptest.NewRecorder(), &events}
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"test_echo","params":[3]}`))
	handler.ServeHTTP(w, req)

	require.Equal(t, eventLog{"recorded", "written"}, events)
	require.Equal(t, response, w.Body.String())
}
