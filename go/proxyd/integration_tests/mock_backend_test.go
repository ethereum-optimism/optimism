package integration_tests

import (
	"bytes"
	"context"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"sync"
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
	Requests []*RecordedRequest
}

func CannedResponseHandler(code int, response string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(code)
		w.Write([]byte(response))
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
	m.Requests = nil
	m.mtx.Unlock()
}

func (m *MockBackend) wrappedHandler(w http.ResponseWriter, r *http.Request) {
	m.mtx.RLock()
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		panic(err)
	}
	clone := r.Clone(context.Background())
	clone.Body = ioutil.NopCloser(bytes.NewReader(body))
	m.Requests = append(m.Requests, &RecordedRequest{
		Method:  r.Method,
		Headers: r.Header.Clone(),
		Body:    body,
	})
	m.handler.ServeHTTP(w, clone)
	m.mtx.RUnlock()
}
