package integration_tests

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"sync"

	"github.com/ethereum-optimism/optimism/go/proxyd"
)

type RecordedRequest struct {
	Method  string
	Headers http.Header
	Body    []byte
}

type MockBackend struct {
	handler  http.Handler
	server   *httptest.Server
	mtx      sync.RWMutex
	requests []*RecordedRequest
}

func SingleResponseHandler(code int, response string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(code)
		_, _ = w.Write([]byte(response))
	}
}

type RPCResponseHandler struct {
	mtx          sync.RWMutex
	rpcResponses map[string]interface{}
}

func NewRPCResponseHandler(rpcResponses map[string]interface{}) *RPCResponseHandler {
	return &RPCResponseHandler{
		rpcResponses: rpcResponses,
	}
}

func (h *RPCResponseHandler) SetResponse(method string, response interface{}) {
	h.mtx.Lock()
	defer h.mtx.Unlock()

	switch response.(type) {
	case string:
	case nil:
		break
	default:
		panic("invalid response type")
	}

	h.rpcResponses[method] = response
}

func (h *RPCResponseHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		panic(err)
	}
	req, err := proxyd.ParseRPCReq(body)
	if err != nil {
		panic(err)
	}
	h.mtx.RLock()
	res := h.rpcResponses[req.Method]
	h.mtx.RUnlock()
	if res == "" {
		w.WriteHeader(400)
		return
	}

	out := &proxyd.RPCRes{
		JSONRPC: proxyd.JSONRPCVersion,
		Result:  res,
		ID:      req.ID,
	}
	enc := json.NewEncoder(w)
	if err := enc.Encode(out); err != nil {
		panic(err)
	}
}

func NewMockBackend(handler http.Handler) *MockBackend {
	mb := &MockBackend{
		handler: handler,
	}
	mb.server = httptest.NewServer(http.HandlerFunc(mb.wrappedHandler))
	return mb
}

func (m *MockBackend) URL() string {
	return m.server.URL
}

func (m *MockBackend) Close() {
	m.server.Close()
}

func (m *MockBackend) SetHandler(handler http.Handler) {
	m.mtx.Lock()
	m.handler = handler
	m.mtx.Unlock()
}

func (m *MockBackend) Reset() {
	m.mtx.Lock()
	m.requests = nil
	m.mtx.Unlock()
}

func (m *MockBackend) Requests() []*RecordedRequest {
	m.mtx.RLock()
	defer m.mtx.RUnlock()
	out := make([]*RecordedRequest, len(m.requests))
	for i := 0; i < len(m.requests); i++ {
		out[i] = m.requests[i]
	}
	return out
}

func (m *MockBackend) wrappedHandler(w http.ResponseWriter, r *http.Request) {
	m.mtx.Lock()
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		panic(err)
	}
	clone := r.Clone(context.Background())
	clone.Body = ioutil.NopCloser(bytes.NewReader(body))
	m.requests = append(m.requests, &RecordedRequest{
		Method:  r.Method,
		Headers: r.Header.Clone(),
		Body:    body,
	})
	m.handler.ServeHTTP(w, clone)
	m.mtx.Unlock()
}
