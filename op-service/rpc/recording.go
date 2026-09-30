package rpc

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	"github.com/ethereum-optimism/optimism/op-service/jsonrpc"
)

// maxRequestBodySize is the go-ethereum server's default request body limit. The server rejects
// or truncates larger requests, so they pass through unrecorded.
const maxRequestBodySize = 5 * 1024 * 1024

// envelope holds the JSON-RPC message fields that recording reads, of a request or a response.
type envelope struct {
	ID     json.RawMessage `json:"id"`
	Method string          `json:"method"`
	Params json.RawMessage `json:"params"`
	Result json.RawMessage `json:"result"`
	Error  *jsonrpc.Error  `json:"error"`
}

// newRecordingHandler reports the JSON-RPC requests that next serves over HTTP, and their
// responses, to rec. The response is recorded as the server writes it, before it reaches the
// client, so recorded durations end when the response is built. Each element of a batch is timed
// as the whole HTTP request. A request gets no recorded response if the server rejects the HTTP
// request (e.g. for its content type).
func newRecordingHandler(rec jsonrpc.Recorder, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests := readRequests(r)
		ctx := r.Context()
		dones := make(map[string][]jsonrpc.RecordDone, len(requests))
		for _, req := range requests {
			if req.Method == "" {
				continue
			}
			msg := jsonrpc.Message{Method: req.Method, Params: req.Params, Notification: len(req.ID) == 0}
			if done := rec.RecordIncoming(ctx, msg); done != nil && !msg.Notification {
				key := idKey(req.ID)
				dones[key] = append(dones[key], done)
			}
		}
		if len(dones) == 0 {
			next.ServeHTTP(w, r)
			return
		}
		next.ServeHTTP(&firstWriteRecorder{ResponseWriter: w, record: func(body []byte) {
			for _, resp := range parseEnvelopes(body) {
				key := idKey(resp.ID)
				if pending := dones[key]; len(pending) > 0 {
					dones[key] = pending[1:]
					pending[0](ctx, jsonrpc.Response{Result: resp.Result, Error: resp.Error})
				}
			}
		}}, r)
	})
}

// idKey returns the ID as the server encodes it in the response, so that a request and its
// response map to the same key.
func idKey(id json.RawMessage) string {
	enc, err := json.Marshal(id)
	if err != nil {
		return string(id)
	}
	return string(enc)
}

// readRequests parses the JSON-RPC requests in the body of r, and leaves the body for the
// next reader. It returns nil for a body it does not record.
func readRequests(r *http.Request) []envelope {
	if r.Method != http.MethodPost || r.ContentLength > maxRequestBodySize {
		return nil
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, maxRequestBodySize+1))
	if err != nil || len(body) > maxRequestBodySize {
		r.Body = readCloser{io.MultiReader(bytes.NewReader(body), r.Body), r.Body}
		return nil
	}
	r.Body = readCloser{bytes.NewReader(body), r.Body}
	return parseEnvelopes(body)
}

// parseEnvelopes parses a single JSON-RPC message or a batch. It returns nil if data is neither.
func parseEnvelopes(data []byte) []envelope {
	data = bytes.TrimLeft(data, " \t\r\n")
	if len(data) > 0 && data[0] == '[' {
		var batch []envelope
		if err := json.Unmarshal(data, &batch); err != nil {
			return nil
		}
		return batch
	}
	var single envelope
	if err := json.Unmarshal(data, &single); err != nil {
		return nil
	}
	return []envelope{single}
}

type readCloser struct {
	io.Reader
	io.Closer
}

// firstWriteRecorder passes the response through, calling record with its first body write.
// The go-ethereum server writes each HTTP response in a single call, so that is the whole
// response; TestServerRPCMetrics pins this.
type firstWriteRecorder struct {
	http.ResponseWriter
	record   func(body []byte)
	recorded bool
}

func (w *firstWriteRecorder) Write(b []byte) (int, error) {
	if !w.recorded {
		w.recorded = true
		w.record(b)
	}
	return w.ResponseWriter.Write(b)
}

// Flush passes flushes on, as the go-ethereum server flushes its timeout error response.
func (w *firstWriteRecorder) Flush() {
	_ = http.NewResponseController(w.ResponseWriter).Flush()
}

func (w *firstWriteRecorder) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}
