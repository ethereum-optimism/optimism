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

func TestFirstWriteRecorderPassThrough(t *testing.T) {
	var recorded []string
	w := httptest.NewRecorder()
	fw := &firstWriteRecorder{ResponseWriter: w, record: func(body []byte) { recorded = append(recorded, string(body)) }}
	fw.WriteHeader(http.StatusInternalServerError)
	_, err := fw.Write([]byte("first"))
	require.NoError(t, err)
	_, err = fw.Write([]byte("second"))
	require.NoError(t, err)
	fw.Flush()

	require.Equal(t, []string{"first"}, recorded)
	require.Equal(t, http.StatusInternalServerError, w.Code)
	require.Equal(t, "firstsecond", w.Body.String())
	require.True(t, w.Flushed, "flushes must reach the underlying writer")
}

func TestIDKey(t *testing.T) {
	for _, tc := range []struct {
		request, response string
	}{
		{`1`, `1`},
		{`"a"`, `"a"`},
		{`"<a>"`, `"\u003ca\u003e"`},
		{`null`, `null`},
		{` 7 `, `7`},
	} {
		require.Equal(t, idKey([]byte(tc.response)), idKey([]byte(tc.request)), tc.request)
	}
	require.NotEqual(t, idKey([]byte(`1`)), idKey([]byte(`"1"`)))
}

func TestParseEnvelopes(t *testing.T) {
	batch := parseEnvelopes([]byte(" \n[{\"id\":1,\"method\":\"a\"},{\"method\":\"b\"}]"))
	require.Len(t, batch, 2)
	require.Equal(t, "a", batch[0].Method)
	require.Empty(t, batch[1].ID)
	require.Len(t, parseEnvelopes([]byte(`{"id":1,"result":2}`)), 1)
	require.Nil(t, parseEnvelopes([]byte(`not json`)))
}
