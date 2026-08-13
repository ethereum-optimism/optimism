package sysgo

import (
	"bytes"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum/go-ethereum/beacon/engine"
)

// EngineFaultProxy is an Engine API reverse proxy that can return INVALID for the next
// forkchoiceUpdated request carrying payload attributes. It is intended for recovery tests that
// need to exercise the CL's handling of an EL-rejected derived payload deterministically.
type EngineFaultProxy struct {
	addr string

	mu                         sync.Mutex
	invalidForkchoiceArmed     bool
	invalidForkchoiceCount     uint64
	lastInvalidForkchoiceAttrs []byte
}

// StartEngineFaultProxy starts a passthrough Engine API proxy in front of targetURL.
func StartEngineFaultProxy(p devtest.T, name, targetURL string) *EngineFaultProxy {
	targetURL = strings.Replace(targetURL, "ws://", "http://", 1)
	target, err := url.Parse(targetURL)
	p.Require().NoError(err, "invalid engine fault proxy target URL %q", targetURL)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	p.Require().NoError(err, "failed to listen for engine fault proxy %s", name)

	proxy := &EngineFaultProxy{addr: "http://" + listener.Addr().String()}
	rp := httputil.NewSingleHostReverseProxy(target)
	server := &http.Server{
		ReadHeaderTimeout: 5 * time.Second,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			r.Body = io.NopCloser(bytes.NewReader(body))

			if proxy.shouldInvalidateForkchoice(body) {
				var req struct {
					JSONRPC string          `json:"jsonrpc"`
					ID      json.RawMessage `json:"id"`
				}
				if err := json.Unmarshal(body, &req); err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}
				validationError := "injected invalid derived payload"
				response := struct {
					JSONRPC string                    `json:"jsonrpc"`
					ID      json.RawMessage           `json:"id"`
					Result  engine.ForkChoiceResponse `json:"result"`
				}{
					JSONRPC: req.JSONRPC,
					ID:      req.ID,
					Result: engine.ForkChoiceResponse{PayloadStatus: engine.PayloadStatusV1{
						Status:          engine.INVALID,
						ValidationError: &validationError,
					}},
				}
				w.Header().Set("Content-Type", "application/json")
				p.Require().NoError(json.NewEncoder(w).Encode(response))
				return
			}

			rp.ServeHTTP(w, r)
		}),
	}
	go func() {
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			p.Logger().Error("Engine fault proxy stopped", "name", name, "err", err)
		}
	}()
	p.Cleanup(func() {
		_ = server.Close()
	})
	return proxy
}

// URL returns the HTTP endpoint that an L2 CL should use as its Engine API endpoint.
func (p *EngineFaultProxy) URL() string {
	return p.addr
}

// InvalidateNextForkchoiceWithAttributes arms one injected INVALID response.
func (p *EngineFaultProxy) InvalidateNextForkchoiceWithAttributes() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.invalidForkchoiceArmed = true
}

// ReinjectLastInvalidForkchoiceWithAttributes arms one INVALID response matching the payload
// attributes of the previously injected request.
func (p *EngineFaultProxy) ReinjectLastInvalidForkchoiceWithAttributes() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.invalidForkchoiceArmed = true
}

// InvalidForkchoiceCount returns the number of injected INVALID responses.
func (p *EngineFaultProxy) InvalidForkchoiceCount() uint64 {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.invalidForkchoiceCount
}

func (p *EngineFaultProxy) shouldInvalidateForkchoice(body []byte) bool {
	var req struct {
		Method string            `json:"method"`
		Params []json.RawMessage `json:"params"`
	}
	if err := json.Unmarshal(body, &req); err != nil ||
		!strings.HasPrefix(req.Method, "engine_forkchoiceUpdatedV") ||
		len(req.Params) < 2 ||
		bytes.Equal(bytes.TrimSpace(req.Params[1]), []byte("null")) {
		return false
	}

	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.invalidForkchoiceArmed {
		return false
	}
	attrs := bytes.TrimSpace(req.Params[1])
	if p.lastInvalidForkchoiceAttrs != nil && !bytes.Equal(p.lastInvalidForkchoiceAttrs, attrs) {
		return false
	}
	p.invalidForkchoiceArmed = false
	p.invalidForkchoiceCount++
	p.lastInvalidForkchoiceAttrs = append(p.lastInvalidForkchoiceAttrs[:0], attrs...)
	return true
}
