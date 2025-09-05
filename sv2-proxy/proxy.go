package sv2proxy

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// RollupProxy is an HTTP reverse proxy that forwards rollup JSON-RPC requests
// to Supervisor v2's embedded op-node exposed at /opnode/{chainId}/.
type RollupProxy struct {
	srv *http.Server
	ln  net.Listener
}

// TCPProxy is a simple TCP forwarder that binds to a given address/port and forwards to an upstream host:port.
// It supports HTTP and WebSocket transparently since it operates at the TCP level.
type TCPProxy struct {
	ln           net.Listener
	upstreamAddr string
	conns        map[net.Conn]struct{}
	mu           sync.Mutex
	wg           sync.WaitGroup
	stopped      atomic.Bool
}

// ELToSV2Proxy groups the per-EL proxies used by SV2: one for user RPC and one for auth RPC.
// DownstreamUserURL/AuthURL are URLs that SV2 should connect to; upstreams are the original EL endpoints.
type ELToSV2Proxy struct {
	User *TCPProxy
	Auth *TCPProxy

	DownstreamUserURL string
	DownstreamAuthURL string
}

// StartRollupProxy starts a reverse proxy bound to 127.0.0.1:0 that forwards to
// supervisorBaseURL/opnode/{chainID}/. It returns the proxy instance and the
// reachable HTTP URL (e.g., http://127.0.0.1:PORT).
func StartRollupProxy(ctx context.Context, supervisorBaseURL string, chainID uint64) (*RollupProxy, string, error) {
	return StartRollupProxyOn(ctx, supervisorBaseURL, chainID, "127.0.0.1", 0)
}

// StartRollupProxyOn starts the rollup reverse proxy bound to listenAddr:listenPort.
// listenPort=0 chooses an ephemeral port.
func StartRollupProxyOn(_ context.Context, supervisorBaseURL string, chainID uint64, listenAddr string, listenPort int) (*RollupProxy, string, error) {
	upstreamURL, err := url.Parse(fmt.Sprintf("%s/opnode/%d/", supervisorBaseURL, chainID))
	if err != nil {
		return nil, "", err
	}
	return startHTTPProxy(listenAddr, listenPort, upstreamURL, true)
}

// Close gracefully shuts down the proxy.
func (p *RollupProxy) Close(ctx context.Context) error {
	if p == nil {
		return nil
	}
	if ctx == nil {
		c, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		ctx = c
	}
	_ = p.srv.Shutdown(ctx)
	if p.ln != nil {
		_ = p.ln.Close()
	}
	return nil
}

// Close gracefully shuts down the TCP proxy.
func (p *TCPProxy) Close(_ context.Context) error {
	if p == nil || p.ln == nil {
		return nil
	}
	p.stopped.Store(true)
	_ = p.ln.Close()
	p.mu.Lock()
	for c := range p.conns {
		_ = c.Close()
	}
	p.mu.Unlock()
	p.wg.Wait()
	return nil
}

// Close both proxies in the composite.
func (p *ELToSV2Proxy) Close(ctx context.Context) error {
	var err1, err2 error
	if p == nil {
		return nil
	}
	if p.User != nil {
		err1 = p.User.Close(ctx)
	}
	if p.Auth != nil {
		err2 = p.Auth.Close(ctx)
	}
	if err1 != nil {
		return err1
	}
	return err2
}

// StartELUserProxy starts a transparent TCP proxy for the EL user RPC at listenAddr:listenPort.
// If listenPort=0, an ephemeral port is chosen. upstream must be a URL like http://host:port or ws://host:port.
func StartELUserProxy(_ context.Context, listenAddr string, listenPort int, upstream string) (*TCPProxy, string, error) {
	return startTCPProxy(listenAddr, listenPort, upstream)
}

// StartELAuthProxy starts a transparent TCP proxy for the EL auth RPC at listenAddr:listenPort.
// If listenPort=0, an ephemeral port is chosen. upstream must be a URL like http://host:port or ws://host:port.
func StartELAuthProxy(_ context.Context, listenAddr string, listenPort int, upstream string) (*TCPProxy, string, error) {
	return startTCPProxy(listenAddr, listenPort, upstream)
}

// StartELToSV2Proxy creates a pair of TCP proxies for a single EL (user+auth) with ephemeral ports and
// returns a composite that can be used to configure SV2 connections to this EL.
func StartELToSV2Proxy(ctx context.Context, elUserUpstream string, elAuthUpstream string) (*ELToSV2Proxy, error) {
	return StartELToSV2ProxyOn(ctx, "127.0.0.1", 0, "127.0.0.1", 0, elUserUpstream, elAuthUpstream)
}

// StartELToSV2ProxyOn creates a pair of TCP proxies bound to specific addresses/ports.
func StartELToSV2ProxyOn(ctx context.Context, userListenAddr string, userListenPort int, authListenAddr string, authListenPort int, elUserUpstream string, elAuthUpstream string) (*ELToSV2Proxy, error) {
	user, userURL, err := StartELUserProxy(ctx, userListenAddr, userListenPort, elUserUpstream)
	if err != nil {
		return nil, err
	}
	auth, authURL, err := StartELAuthProxy(ctx, authListenAddr, authListenPort, elAuthUpstream)
	if err != nil {
		_ = user.Close(ctx)
		return nil, err
	}
	return &ELToSV2Proxy{User: user, Auth: auth, DownstreamUserURL: userURL, DownstreamAuthURL: authURL}, nil
}

// startHTTPProxy is a helper to spin up an HTTP reverse proxy. If stripToRoot is true,
// it rewrites the request path to "/" before forwarding (used for rollup base-path proxying).
func startHTTPProxy(listenAddr string, listenPort int, upstream *url.URL, stripToRoot bool) (*RollupProxy, string, error) {
	rp := httputil.NewSingleHostReverseProxy(upstream)
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if stripToRoot {
			r.URL.Path = "/"
		}
		rp.ServeHTTP(w, r)
	})
	srv := &http.Server{Handler: h}
	bind := fmt.Sprintf("%s:%d", listenAddr, listenPort)
	ln, err := net.Listen("tcp", bind)
	if err != nil {
		return nil, "", err
	}
	go func() { _ = srv.Serve(ln) }()
	addr := "http://" + ln.Addr().String()
	return &RollupProxy{srv: srv, ln: ln}, addr, nil
}

// startTCPProxy is a helper to spin up a raw TCP forwarder bound to listenAddr:listenPort, forwarding to upstream host:port.
func startTCPProxy(listenAddr string, listenPort int, upstream string) (*TCPProxy, string, error) {
	u, err := url.Parse(upstream)
	if err != nil {
		return nil, "", err
	}
	hostport := u.Host
	if hostport == "" {
		// allow raw host:port input
		hostport = strings.TrimPrefix(upstream, "ws://")
		hostport = strings.TrimPrefix(hostport, "http://")
		hostport = strings.TrimPrefix(hostport, "wss://")
		hostport = strings.TrimPrefix(hostport, "https://")
	}
	bind := fmt.Sprintf("%s:%d", listenAddr, listenPort)
	ln, err := net.Listen("tcp", bind)
	if err != nil {
		return nil, "", err
	}
	p := &TCPProxy{
		ln:           ln,
		upstreamAddr: hostport,
		conns:        make(map[net.Conn]struct{}),
	}
	// accept loop
	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		for {
			downConn, err := p.ln.Accept()
			if err != nil {
				if p.stopped.Load() {
					return
				}
				// continue accept loop on transient errors
				continue
			}
			p.wg.Add(1)
			go func() {
				defer p.wg.Done()
				p.handleConn(downConn)
			}()
		}
	}()
	addr := "http://" + ln.Addr().String()
	return p, addr, nil
}

func (p *TCPProxy) handleConn(downConn net.Conn) {
	defer downConn.Close()
	upConn, err := net.Dial("tcp", p.upstreamAddr)
	if err != nil {
		return
	}
	defer upConn.Close()
	p.mu.Lock()
	p.conns[downConn] = struct{}{}
	p.conns[upConn] = struct{}{}
	p.mu.Unlock()
	var wg sync.WaitGroup
	wg.Add(2)
	pump := func(dst io.Writer, src io.Reader) {
		defer wg.Done()
		_, _ = io.Copy(dst, src)
	}
	go pump(downConn, upConn)
	go pump(upConn, downConn)
	wg.Wait()
	p.mu.Lock()
	delete(p.conns, downConn)
	delete(p.conns, upConn)
	p.mu.Unlock()
}
