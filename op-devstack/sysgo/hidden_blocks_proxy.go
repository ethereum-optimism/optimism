package sysgo

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/coder/websocket"
	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-service/log"
)

// HiddenBlocks routes the ZK proposer's L2 EL endpoints through WebSocket
// JSON-RPC proxies that refuse every request naming a hidden block hash, as an
// EL that pruned the block, or never had it, would. Other messages pass through
// unchanged, so the proposer reads everything else from the real ELs.
type HiddenBlocks struct {
	mu     sync.Mutex
	hashes map[common.Hash]struct{}

	forwarded atomic.Int64
	refused   atomic.Int64
}

func NewHiddenBlocks() *HiddenBlocks {
	return &HiddenBlocks{hashes: make(map[common.Hash]struct{})}
}

// Hide makes the proxies refuse every request with hash among its params.
func (h *HiddenBlocks) Hide(hash common.Hash) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.hashes[hash] = struct{}{}
}

// ForwardedRequests returns the number of requests the proxies passed to an
// EL. Tests use it to assert the proposer's traffic flowed through the
// proxies, so a mis-wired option cannot pass vacuously.
func (h *HiddenBlocks) ForwardedRequests() int64 {
	return h.forwarded.Load()
}

// RefusedRequests returns the number of requests the proxies refused because
// they named a hidden block.
func (h *HiddenBlocks) RefusedRequests() int64 {
	return h.refused.Load()
}

// ZKProposerOption interposes a proxy in front of every L2 EL endpoint given to
// the ZK proposer, which its in-process kona-host shares.
func (h *HiddenBlocks) ZKProposerOption() ZKProposerOption {
	return func(cfg *zkProposerConfig) {
		cfg.L2RPCProxy = h.startProxy
	}
}

// startProxy serves a WebSocket endpoint in front of the WebSocket EL endpoint
// targetURL and returns its URL.
func (h *HiddenBlocks) startProxy(p devtest.T, name string, targetURL string) string {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	p.Require().NoError(err, "failed to listen for hidden-blocks proxy %s", name)

	logger := p.Logger().New("component", "hidden-blocks-proxy-"+name)
	ctx, cancel := context.WithCancel(p.Ctx())
	server := &http.Server{
		ReadHeaderTimeout: 5 * time.Second,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h.relay(ctx, logger, w, r, targetURL)
		}),
	}
	go func() {
		_ = server.Serve(listener)
	}()
	p.Cleanup(func() {
		// Hijacked WebSocket connections outlive Close; the cancel ends their relays.
		cancel()
		_ = server.Close()
	})
	addr := "ws://" + listener.Addr().String()
	logger.Info("Started hidden-blocks proxy", "addr", addr, "target", targetURL)
	return addr
}

// relay pairs one client connection with its own connection to the EL and
// answers requests for hidden blocks itself.
func (h *HiddenBlocks) relay(ctx context.Context, logger log.Logger, w http.ResponseWriter, r *http.Request, targetURL string) {
	client, err := websocket.Accept(w, r, nil)
	if err != nil {
		logger.Warn("Failed to accept a proxied connection", "err", err)
		return
	}
	defer func() { _ = client.CloseNow() }()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	// Not client.DialWS: its reads time out after 30s idle, and the proposer idles between games.
	upstream, resp, err := websocket.Dial(ctx, targetURL, nil)
	if resp != nil && resp.Body != nil {
		_ = resp.Body.Close()
	}
	if err != nil {
		logger.Warn("Failed to dial the EL", "target", targetURL, "err", err)
		return
	}
	defer func() { _ = upstream.CloseNow() }()
	// Raw blocks and receipts exceed the default 32 KiB message limit.
	client.SetReadLimit(-1)
	upstream.SetReadLimit(-1)

	go func() {
		defer cancel()
		for {
			typ, msg, err := upstream.Read(ctx)
			if err != nil || client.Write(ctx, typ, msg) != nil {
				return
			}
		}
	}()
	for {
		typ, msg, err := client.Read(ctx)
		if err != nil {
			return
		}
		requests, batch := parseHiddenBlockRPCBody(msg)
		if hash, hidden := h.hiddenHash(requests); hidden {
			h.refused.Add(1)
			logger.Info("Refusing a request for a hidden block", "hash", hash)
			if client.Write(ctx, websocket.MessageText, hiddenBlockErrors(requests, batch, hash)) != nil {
				return
			}
			continue
		}
		h.forwarded.Add(1)
		if upstream.Write(ctx, typ, msg) != nil {
			return
		}
	}
}

type hiddenBlockRPCRequest struct {
	ID     json.RawMessage   `json:"id"`
	Params []json.RawMessage `json:"params"`
}

// parseHiddenBlockRPCBody decodes a single or batch JSON-RPC message. A message
// that does not decode yields no requests, so it is forwarded untouched.
func parseHiddenBlockRPCBody(body []byte) (requests []hiddenBlockRPCRequest, batch bool) {
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) > 0 && trimmed[0] == '[' {
		if json.Unmarshal(trimmed, &requests) != nil {
			return nil, true
		}
		return requests, true
	}
	var req hiddenBlockRPCRequest
	if json.Unmarshal(trimmed, &req) != nil {
		return nil, false
	}
	return []hiddenBlockRPCRequest{req}, false
}

// hiddenHash returns the first hidden hash among the requests' params. A batch
// naming one is refused as a whole.
func (h *HiddenBlocks) hiddenHash(requests []hiddenBlockRPCRequest) (common.Hash, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, req := range requests {
		for _, param := range req.Params {
			var s string
			if json.Unmarshal(param, &s) != nil || len(s) != 2+2*common.HashLength {
				continue
			}
			hash := common.HexToHash(s)
			if _, ok := h.hashes[hash]; ok {
				return hash, true
			}
		}
	}
	return common.Hash{}, false
}

func hiddenBlockErrors(requests []hiddenBlockRPCRequest, batch bool, hash common.Hash) []byte {
	type rpcError struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	type rpcResponse struct {
		JSONRPC string          `json:"jsonrpc"`
		ID      json.RawMessage `json:"id"`
		Error   rpcError        `json:"error"`
	}
	responses := make([]rpcResponse, len(requests))
	for i, req := range requests {
		responses[i] = rpcResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   rpcError{Code: -32000, Message: fmt.Sprintf("block %s is hidden by the test proxy", hash)},
		}
	}
	var out any = responses[0]
	if batch {
		out = responses
	}
	msg, _ := json.Marshal(out) // plain structs of strings and raw JSON always marshal
	return msg
}
