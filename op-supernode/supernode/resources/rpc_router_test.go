package resources

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	gethlog "github.com/ethereum/go-ethereum/log"
)

func rpcEchoHandler(t *testing.T, name string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Chain", name)
		_, _ = w.Write([]byte(r.URL.Path))
	})
}

func TestDispatchToCorrectChain(t *testing.T) {
	l := gethlog.Root()
	router := NewRouter(l, RouterConfig{})
	router.SetHandler("10", rpcEchoHandler(t, "10"))
	router.SetHandler("20", rpcEchoHandler(t, "20"))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/10", nil)
	router.ServeHTTP(rec, req)
	if got := rec.Header().Get("X-Chain"); got != "10" {
		t.Fatalf("expected chain 10, got %q", got)
	}
	if body := rec.Body.String(); body != "/" {
		t.Fatalf("expected path /, got %q", body)
	}
}

func TestPathRewriting(t *testing.T) {
	l := gethlog.Root()
	router := NewRouter(l, RouterConfig{})
	router.SetHandler("10", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(r.URL.Path))
	}))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/10/eth_blockNumber", nil)
	router.ServeHTTP(rec, req)
	if body := rec.Body.String(); body != "/eth_blockNumber" {
		t.Fatalf("expected path /eth_blockNumber, got %q", body)
	}
}

func TestUnknownChain(t *testing.T) {
	l := gethlog.Root()
	router := NewRouter(l, RouterConfig{})
	router.SetHandler("10", http.NotFoundHandler())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/999", nil)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestRouterHoldsUntilReady(t *testing.T) {
	l := gethlog.Root()
	router := NewRouter(l, RouterConfig{GateTimeout: time.Second})

	var ready atomic.Bool
	handlerCalled := make(chan struct{})
	router.SetHandler("10", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(handlerCalled)
		w.WriteHeader(http.StatusNoContent)
	}))
	router.SetReadinessCheck("10", ready.Load)

	done := make(chan int, 1)
	go func() {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/10", nil)
		router.ServeHTTP(rec, req)
		done <- rec.Code
	}()

	select {
	case code := <-done:
		t.Fatalf("expected request to wait for readiness, completed with %d", code)
	case <-time.After(2 * defaultGatePoll):
	}

	select {
	case <-handlerCalled:
		t.Fatalf("handler was called before route became ready")
	default:
	}

	ready.Store(true)

	select {
	case code := <-done:
		if code != http.StatusNoContent {
			t.Fatalf("expected 204 after route became ready, got %d", code)
		}
	case <-time.After(time.Second):
		t.Fatalf("request did not complete after route became ready")
	}
}

func TestRouterGateRespectsContextDeadline(t *testing.T) {
	l := gethlog.Root()
	router := NewRouter(l, RouterConfig{GateTimeout: 5 * time.Second})

	var handlerCalled atomic.Bool
	router.SetHandler("10", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled.Store(true)
		w.WriteHeader(http.StatusNoContent)
	}))
	router.SetReadinessCheck("10", func() bool { return false })

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/10", nil)
	ctx, cancel := context.WithTimeout(req.Context(), 100*time.Millisecond)
	defer cancel()
	req = req.WithContext(ctx)

	done := make(chan int, 1)
	go func() {
		router.ServeHTTP(rec, req)
		done <- rec.Code
	}()

	select {
	case code := <-done:
		if code != http.StatusServiceUnavailable {
			t.Fatalf("expected 503, got %d", code)
		}
	case <-time.After(3 * time.Second):
		t.Fatalf("request did not complete after context deadline")
	}
	if handlerCalled.Load() {
		t.Fatalf("handler should not be called while route is not ready")
	}
}

func TestRouterHandlerSwapDuringHold(t *testing.T) {
	l := gethlog.Root()
	router := NewRouter(l, RouterConfig{GateTimeout: time.Second})

	var ready atomic.Bool
	router.SetHandler("10", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Handler", "old")
	}))
	router.SetReadinessCheck("10", ready.Load)

	done := make(chan *httptest.ResponseRecorder, 1)
	go func() {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/10", nil)
		router.ServeHTTP(rec, req)
		done <- rec
	}()

	select {
	case rec := <-done:
		t.Fatalf("expected request to wait for readiness, completed with %d", rec.Code)
	case <-time.After(2 * defaultGatePoll):
	}

	router.SetHandler("10", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Handler", "new")
		w.WriteHeader(http.StatusNoContent)
	}))
	ready.Store(true)

	select {
	case rec := <-done:
		if rec.Code != http.StatusNoContent {
			t.Fatalf("expected 204 after route became ready, got %d", rec.Code)
		}
		if got := rec.Header().Get("X-Handler"); got != "new" {
			t.Fatalf("expected swapped handler to serve request, got %q", got)
		}
	case <-time.After(time.Second):
		t.Fatalf("request did not complete after route became ready")
	}
}

func TestRouterNoReadinessCheckDispatches(t *testing.T) {
	l := gethlog.Root()
	router := NewRouter(l, RouterConfig{})

	router.SetHandler("10", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/10", nil)
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected ungated route to dispatch immediately, got %d", rec.Code)
	}
}

func TestRouterRemoveHandlerDrainsWaiter(t *testing.T) {
	l := gethlog.Root()
	router := NewRouter(l, RouterConfig{GateTimeout: time.Second})

	router.SetHandler("10", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("handler should not be called after route is removed")
	}))
	router.SetReadinessCheck("10", func() bool { return false })

	done := make(chan int, 1)
	go func() {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/10", nil)
		router.ServeHTTP(rec, req)
		done <- rec.Code
	}()

	select {
	case code := <-done:
		t.Fatalf("expected request to wait before removal, completed with %d", code)
	case <-time.After(2 * defaultGatePoll):
	}

	router.RemoveHandler("10")

	select {
	case code := <-done:
		if code != http.StatusNotFound {
			t.Fatalf("expected 404 after route removal, got %d", code)
		}
	case <-time.After(time.Second):
		t.Fatalf("request did not complete after route removal")
	}
}

func TestRouterGateTimeoutBackstop(t *testing.T) {
	l := gethlog.Root()
	router := NewRouter(l, RouterConfig{GateTimeout: 100 * time.Millisecond})

	var handlerCalled atomic.Bool
	router.SetHandler("10", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled.Store(true)
		w.WriteHeader(http.StatusNoContent)
	}))
	router.SetReadinessCheck("10", func() bool { return false })

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/10", nil)
	ctx, cancel := context.WithCancel(req.Context())
	defer cancel()
	req = req.WithContext(ctx)

	done := make(chan int, 1)
	go func() {
		router.ServeHTTP(rec, req)
		done <- rec.Code
	}()

	select {
	case code := <-done:
		if code != http.StatusServiceUnavailable {
			t.Fatalf("expected 503, got %d", code)
		}
	case <-time.After(3 * time.Second):
		t.Fatalf("request did not complete after gate timeout")
	}
	if handlerCalled.Load() {
		t.Fatalf("handler should not be called while route is not ready")
	}
}
