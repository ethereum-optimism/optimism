package sysgo

import (
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// StallableProxy is a reverse proxy for an HTTP RPC endpoint that can be
// switched into a stalled mode: while stalled, incoming requests are held
// open without a response until the client gives up (e.g. its call timeout)
// or the proxy is resumed. It lets tests simulate an unresponsive upstream
// without stopping the upstream itself.
type StallableProxy struct {
	name   string
	target *url.URL
	server *http.Server
	addr   string

	mu      sync.Mutex
	stallCh chan struct{} // non-nil while stalled; closed by Resume

	stalledRequests atomic.Int64
}

// StartStallableProxy starts a StallableProxy in front of targetURL. The proxy
// begins in passthrough mode and is shut down via test cleanup.
func StartStallableProxy(p devtest.T, name string, targetURL string) *StallableProxy {
	target, err := url.Parse(targetURL)
	p.Require().NoError(err, "invalid stallable proxy target URL %q", targetURL)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	p.Require().NoError(err, "failed to listen for stallable proxy %s", name)

	logger := p.Logger().New("component", "stallable-proxy-"+name)
	proxy := &StallableProxy{
		name:   name,
		target: target,
		addr:   "http://" + listener.Addr().String(),
	}
	rp := httputil.NewSingleHostReverseProxy(target)
	proxy.server = &http.Server{
		// Bound only the header-read phase. Read/write timeouts are deliberately
		// absent: a stalled request must be able to stay open longer than any
		// client-side call timeout, which is the point of this proxy.
		ReadHeaderTimeout: 5 * time.Second,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if ch := proxy.currentStall(); ch != nil {
				n := proxy.stalledRequests.Add(1)
				logger.Info("Stalling proxied request", "target", target, "stalled_requests", n)
				select {
				case <-r.Context().Done():
					// Client gave up (e.g. RPC call timeout) or the server is closing.
					return
				case <-ch:
					// Resumed; serve the request after all.
				}
			}
			rp.ServeHTTP(w, r)
		}),
	}
	go func() {
		_ = proxy.server.Serve(listener)
	}()
	p.Cleanup(func() {
		_ = proxy.server.Close()
	})
	logger.Info("Started stallable proxy", "addr", proxy.addr, "target", target)
	return proxy
}

// URL returns the address clients should connect to instead of the target.
func (p *StallableProxy) URL() string {
	return p.addr
}

// Stall holds all new requests open without a response until Resume is called.
// Idempotent.
func (p *StallableProxy) Stall() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.stallCh == nil {
		p.stallCh = make(chan struct{})
	}
}

// Resume releases all held requests and returns to passthrough mode. Idempotent.
func (p *StallableProxy) Resume() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.stallCh != nil {
		close(p.stallCh)
		p.stallCh = nil
	}
}

// StalledRequests returns the number of requests that have been held by the
// stall so far. Tests use it to assert traffic actually flowed through the
// stalled proxy, so a mis-wired proxy cannot pass vacuously.
func (p *StallableProxy) StalledRequests() int64 {
	return p.stalledRequests.Load()
}

func (p *StallableProxy) currentStall() chan struct{} {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.stallCh
}

// StallableFollowSourceProxies routes the follow-source endpoint of every L2 CL
// that has one through its own StallableProxy, so tests can stall follow-source
// RPC traffic per chain while the follow source itself keeps running.
type StallableFollowSourceProxies struct {
	mu      sync.Mutex
	proxies map[eth.ChainID]*StallableProxy
}

func NewStallableFollowSourceProxies() *StallableFollowSourceProxies {
	return &StallableFollowSourceProxies{
		proxies: make(map[eth.ChainID]*StallableProxy),
	}
}

// L2CLOption returns the option that interposes a proxy in front of
// cfg.FollowSource. CLs without a follow source are left untouched.
func (s *StallableFollowSourceProxies) L2CLOption() L2CLOption {
	return L2CLOptionFn(func(p devtest.T, target ComponentTarget, cfg *L2CLConfig) {
		if cfg.FollowSource == "" {
			return
		}
		s.mu.Lock()
		defer s.mu.Unlock()
		_, exists := s.proxies[target.ChainID]
		p.Require().False(exists, "follow-source proxy already registered for chain %s", target.ChainID)
		proxy := StartStallableProxy(p, "follow-source-"+target.String(), cfg.FollowSource)
		s.proxies[target.ChainID] = proxy
		cfg.FollowSource = proxy.URL()
	})
}

// ForChain returns the proxy interposed for the given chain's follow-source CL,
// failing the test if none was wired (e.g. the option was not applied).
func (s *StallableFollowSourceProxies) ForChain(p devtest.T, chainID eth.ChainID) *StallableProxy {
	s.mu.Lock()
	defer s.mu.Unlock()
	proxy, ok := s.proxies[chainID]
	p.Require().True(ok, "no follow-source proxy wired for chain %s", chainID)
	return proxy
}
