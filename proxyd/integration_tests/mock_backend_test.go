package integration_tests

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"

	"github.com/ethereum-optimism/optimism/proxyd"
	"github.com/gorilla/websocket"
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

func BatchedResponseHandler(code int, responses ...string) http.HandlerFunc {
	// all proxyd upstream requests are batched
	return func(w http.ResponseWriter, r *http.Request) {
		var body string
		body += "["
		for i, response := range responses {
			body += response
			if i+1 < len(responses) {
				body += ","
			}
		}
		body += "]"
		SingleResponseHandler(code, body)(w, r)
	}
}

type responseMapping struct {
	result interface{}
	calls  int
}
type BatchRPCResponseRouter struct {
	m        map[string]map[string]*responseMapping
	fallback map[string]interface{}
	mtx      sync.Mutex
}

func NewBatchRPCResponseRouter() *BatchRPCResponseRouter {
	return &BatchRPCResponseRouter{
		m:        make(map[string]map[string]*responseMapping),
		fallback: make(map[string]interface{}),
	}
}

func (h *BatchRPCResponseRouter) SetRoute(method string, id string, result interface{}) {
	h.mtx.Lock()
	defer h.mtx.Unlock()

	switch result.(type) {
	case string:
	case nil:
		break
	default:
		panic("invalid result type")
	}

	m := h.m[method]
	if m == nil {
		m = make(map[string]*responseMapping)
	}
	m[id] = &responseMapping{result: result}
	h.m[method] = m
}

func (h *BatchRPCResponseRouter) SetFallbackRoute(method string, result interface{}) {
	h.mtx.Lock()
	defer h.mtx.Unlock()

	switch result.(type) {
	case string:
	case nil:
		break
	default:
		panic("invalid result type")
	}

	h.fallback[method] = result
}

func (h *BatchRPCResponseRouter) GetNumCalls(method string, id string) int {
	h.mtx.Lock()
	defer h.mtx.Unlock()

	if m := h.m[method]; m != nil {
		if rm := m[id]; rm != nil {
			return rm.calls
		}
	}
	return 0
}

func (h *BatchRPCResponseRouter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mtx.Lock()
	defer h.mtx.Unlock()

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		panic(err)
	}

	if proxyd.IsBatch(body) {
		batch, err := proxyd.ParseBatchRPCReq(body)
		if err != nil {
			panic(err)
		}
		out := make([]*proxyd.RPCRes, len(batch))
		for i := range batch {
			req, err := proxyd.ParseRPCReq(batch[i])
			if err != nil {
				panic(err)
			}

			var result interface{}
			var resultHasValue bool

			if mappings, exists := h.m[req.Method]; exists {
				if rm := mappings[string(req.ID)]; rm != nil {
					result = rm.result
					resultHasValue = true
					rm.calls++
				}
			}
			if !resultHasValue {
				result, resultHasValue = h.fallback[req.Method]
			}
			if !resultHasValue {
				w.WriteHeader(400)
				return
			}

			out[i] = &proxyd.RPCRes{
				JSONRPC: proxyd.JSONRPCVersion,
				Result:  result,
				ID:      req.ID,
			}
		}
		if err := json.NewEncoder(w).Encode(out); err != nil {
			panic(err)
		}
		return
	}

	req, err := proxyd.ParseRPCReq(body)
	if err != nil {
		panic(err)
	}

	var result interface{}
	var resultHasValue bool

	if mappings, exists := h.m[req.Method]; exists {
		if rm := mappings[string(req.ID)]; rm != nil {
			result = rm.result
			resultHasValue = true
			rm.calls++
		}
	}
	if !resultHasValue {
		result, resultHasValue = h.fallback[req.Method]
	}
	if !resultHasValue {
		w.WriteHeader(400)
		return
	}

	out := &proxyd.RPCRes{
		JSONRPC: proxyd.JSONRPCVersion,
		Result:  result,
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

type MockWSBackend struct {
	connCB   MockWSBackendOnConnect
	msgCB    MockWSBackendOnMessage
	closeCB  MockWSBackendOnClose
	server   *httptest.Server
	upgrader websocket.Upgrader
	conns    []*websocket.Conn
	connsMu  sync.Mutex
}

type MockWSBackendOnConnect func(conn *websocket.Conn)
type MockWSBackendOnMessage func(conn *websocket.Conn, msgType int, data []byte)
type MockWSBackendOnClose func(conn *websocket.Conn, err error)

func NewMockWSBackend(
	connCB MockWSBackendOnConnect,
	msgCB MockWSBackendOnMessage,
	closeCB MockWSBackendOnClose,
) *MockWSBackend {
	mb := &MockWSBackend{
		connCB:  connCB,
		msgCB:   msgCB,
		closeCB: closeCB,
	}
	mb.server = httptest.NewServer(mb)
	return mb
}

func (m *MockWSBackend) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	conn, err := m.upgrader.Upgrade(w, r, nil)
	if err != nil {
		panic(err)
	}
	if m.connCB != nil {
		m.connCB(conn)
	}
	go func() {
		for {
			mType, msg, err := conn.ReadMessage()
			if err != nil {
				if m.closeCB != nil {
					m.closeCB(conn, err)
				}
				return
			}
			if m.msgCB != nil {
				m.msgCB(conn, mType, msg)
			}
		}
	}()
	m.connsMu.Lock()
	m.conns = append(m.conns, conn)
	m.connsMu.Unlock()
}

func (m *MockWSBackend) URL() string {
	return strings.Replace(m.server.URL, "http://", "ws://", 1)
}

func (m *MockWSBackend) Close() {
	m.server.Close()

	m.connsMu.Lock()
	for _, conn := range m.conns {
		conn.Close()
	}
	m.connsMu.Unlock()
}
