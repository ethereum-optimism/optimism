package resources

import (
	"net/http"
	"net/http/httptest"
	"testing"

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
