package httputil

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync/atomic"
)

// HTTPServer wraps a http.Server, while providing conveniences
// like exposing the running state and address.
type HTTPServer struct {
	listener net.Listener
	srv      *http.Server
	closed   atomic.Bool
}

// HTTPOption applies a change to an HTTP server
type HTTPOption func(srv *HTTPServer) error

func StartHTTPServer(addr string, handler http.Handler, opts ...HTTPOption) (*HTTPServer, error) {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("failed to bind to address %q: %w", addr, err)
	}

	srv := &http.Server{
		Handler:           handler,
		ReadTimeout:       DefaultTimeouts.ReadTimeout,
		ReadHeaderTimeout: DefaultTimeouts.ReadHeaderTimeout,
		WriteTimeout:      DefaultTimeouts.WriteTimeout,
		IdleTimeout:       DefaultTimeouts.IdleTimeout,
	}
	out := &HTTPServer{listener: listener, srv: srv}
	for _, opt := range opts {
		if err := opt(out); err != nil {
			return nil, errors.Join(fmt.Errorf("failed to apply HTTP option: %w", err), listener.Close())
		}
	}
	go func() {
		err := out.srv.Serve(listener)
		// no error, unless ErrServerClosed (or unused base context closes, or unused http2 config error)
		if errors.Is(err, http.ErrServerClosed) {
			out.closed.Store(true)
		} else {
			panic(fmt.Errorf("unexpected serve error: %w", err))
		}
	}()
	return out, nil
}

func (s *HTTPServer) Closed() bool {
	return s.closed.Load()
}

func (s *HTTPServer) Close() error {
	// closes the underlying listener too
	err := s.srv.Close()
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

func (s *HTTPServer) Addr() string {
	return s.listener.Addr().String()
}

func WithMaxHeaderBytes(max int) HTTPOption {
	return func(srv *HTTPServer) error {
		srv.srv.MaxHeaderBytes = max
		return nil
	}
}
