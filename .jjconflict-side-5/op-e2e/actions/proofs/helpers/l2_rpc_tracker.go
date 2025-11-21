package helpers

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rpc"
)

// L2RPCTracker observes JSON-RPC calls made by the fault proof program host.
// It supports both:
// - in-process tracking (via WrapRPC)
// - more realistic tracking using an HTTP proxy (via StartProxy)
type L2RPCTracker struct {
	mu sync.Mutex

	totalByMethod map[string]int

	uniqueBlockByHash map[string]struct{}
	uniqueBlockByNum  map[string]struct{}

	forceProxy bool
	httpClient *http.Client
}

func NewL2RPCTracker() *L2RPCTracker {
	return &L2RPCTracker{
		totalByMethod:     make(map[string]int),
		uniqueBlockByHash: make(map[string]struct{}),
		uniqueBlockByNum:  make(map[string]struct{}),
		httpClient:        &http.Client{Timeout: 30 * time.Second},
	}
}

// ForceProxyMode makes the tracker collect data via the HTTP proxy.
func (t *L2RPCTracker) ForceProxyMode() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.forceProxy = true
}

func (t *L2RPCTracker) shouldUseProxy() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.forceProxy
}

// WrapRPC wraps an op-service client.RPC and increments counters for each call.
func (t *L2RPCTracker) WrapRPC(rpcClient client.RPC) client.RPC {
	return &trackingRPC{inner: rpcClient, tracker: t}
}

// StartProxy starts an HTTP JSON-RPC proxy in front of the upstream endpoint.
// The returned URL should be used in place of the upstream URL.
func (t *L2RPCTracker) StartProxy(upstreamURL string) (proxyURL string, closeFn func()) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		_ = r.Body.Close()

		t.observeJSONRPCBody(body)

		req, err := http.NewRequestWithContext(r.Context(), http.MethodPost, upstreamURL, bytes.NewReader(body))
		if err != nil {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		req.Header = r.Header.Clone()

		httpClient := t.httpClient
		if httpClient == nil {
			httpClient = http.DefaultClient
		}
		resp, err := httpClient.Do(req)
		if err != nil {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()

		for k, vals := range resp.Header {
			for _, v := range vals {
				w.Header().Add(k, v)
			}
		}
		w.WriteHeader(resp.StatusCode)
		_, _ = io.Copy(w, resp.Body)
	})

	// Avoid hanging test runs if the upstream is unresponsive.
	server := httptest.NewUnstartedServer(h)
	server.Config.ReadHeaderTimeout = 5 * time.Second
	server.Start()
	return server.URL, server.Close
}

type L2RPCSnapshot struct {
	TotalByMethod        map[string]int
	UniqueGetBlockByHash int
	UniqueGetBlockByNum  int
}

// Snapshot returns a snapshot of the current RPC call counts.
func (t *L2RPCTracker) Snapshot() L2RPCSnapshot {
	t.mu.Lock()
	defer t.mu.Unlock()

	out := L2RPCSnapshot{
		TotalByMethod:        make(map[string]int, len(t.totalByMethod)),
		UniqueGetBlockByHash: len(t.uniqueBlockByHash),
		UniqueGetBlockByNum:  len(t.uniqueBlockByNum),
	}
	for k, v := range t.totalByMethod {
		out.TotalByMethod[k] = v
	}
	return out
}

// UniqueBlockFetches returns the total number of unique block fetches by hash or number.
func (t *L2RPCTracker) UniqueBlockFetches() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return len(t.uniqueBlockByHash) + len(t.uniqueBlockByNum)
}

type trackingRPC struct {
	inner   client.RPC
	tracker *L2RPCTracker
}

// Close implements client.RPC
func (t *trackingRPC) Close() { t.inner.Close() }

// CallContext implements client.RPC
func (t *trackingRPC) CallContext(ctx context.Context, result any, method string, args ...any) error {
	t.tracker.observeCall(method, args)
	return t.inner.CallContext(ctx, result, method, args...)
}

// BatchCallContext implements client.RPC
func (t *trackingRPC) BatchCallContext(ctx context.Context, b []rpc.BatchElem) error {
	for i := range b {
		t.tracker.observeCall(b[i].Method, b[i].Args)
	}
	return t.inner.BatchCallContext(ctx, b)
}

// Subscribe implements client.RPC
func (t *trackingRPC) Subscribe(ctx context.Context, namespace string, channel any, args ...any) (ethereum.Subscription, error) {
	return t.inner.Subscribe(ctx, namespace, channel, args...)
}

func (t *L2RPCTracker) observeCall(method string, args []any) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.totalByMethod[method]++
	switch method {
	case "eth_getBlockByHash":
		if len(args) == 0 {
			return
		}
		if hash, ok := normalizeHashArg(args[0]); ok {
			t.uniqueBlockByHash[hash] = struct{}{}
		}
	case "eth_getBlockByNumber":
		if len(args) == 0 {
			return
		}
		if num, ok := normalizeNumberArg(args[0]); ok {
			t.uniqueBlockByNum[num] = struct{}{}
		}
	}
}

func (t *L2RPCTracker) observeJSONRPCBody(body []byte) {
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) == 0 {
		return
	}
	if trimmed[0] == '[' {
		var reqs []jsonrpcReq
		if err := json.Unmarshal(trimmed, &reqs); err != nil {
			return
		}
		for _, req := range reqs {
			t.observeCall(req.Method, req.Params)
		}
		return
	}
	var req jsonrpcReq
	if err := json.Unmarshal(trimmed, &req); err != nil {
		return
	}
	t.observeCall(req.Method, req.Params)
}

type jsonrpcReq struct {
	Method string `json:"method"`
	Params []any  `json:"params"`
}

func normalizeHashArg(arg any) (string, bool) {
	switch v := arg.(type) {
	case common.Hash:
		if v == (common.Hash{}) {
			return "", false
		}
		return strings.ToLower(v.Hex()), true
	case *common.Hash:
		if v == nil || *v == (common.Hash{}) {
			return "", false
		}
		return strings.ToLower(v.Hex()), true
	case string:
		if !strings.HasPrefix(v, "0x") {
			return "", false
		}
		return strings.ToLower(v), true
	case map[string]any:
		// in case the caller uses block-number-or-hash objects.
		if bh, ok := v["blockHash"].(string); ok && strings.HasPrefix(bh, "0x") {
			return strings.ToLower(bh), true
		}
	}
	// fallback to types that implement Hex() string.
	type hexer interface{ Hex() string }
	if h, ok := arg.(hexer); ok {
		hex := h.Hex()
		if strings.HasPrefix(hex, "0x") {
			return strings.ToLower(hex), true
		}
	}
	return "", false
}

func normalizeNumberArg(arg any) (string, bool) {
	switch v := arg.(type) {
	case string:
		// number is typically a hex quantity (e.g. "0x10"), but might be labels like "latest".
		if !strings.HasPrefix(v, "0x") {
			return "", false
		}
		return strings.ToLower(v), true
	}
	return "", false
}
