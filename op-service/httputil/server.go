package httputil

import (
	"context"
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

	srvCtx, srvCancel := context.WithCancel(context.Background())
	srv := &http.Server{
		Handler:           handler,
		ReadTimeout:       DefaultTimeouts.ReadTimeout,
		ReadHeaderTimeout: DefaultTimeouts.ReadHeaderTimeout,
		WriteTimeout:      DefaultTimeouts.WriteTimeout,
		IdleTimeout:       DefaultTimeouts.IdleTimeout,
		BaseContext: func(listener net.Listener) context.Context {
			return srvCtx
		},
	}
	out := &HTTPServer{listener: listener, srv: srv}
	for _, opt := range opts {
		if err := opt(out); err != nil {
			srvCancel()
			return nil, errors.Join(fmt.Errorf("failed to apply HTTP option: %w", err), listener.Close())
		}
	}
	go func() {
		err := out.srv.Serve(listener)
		srvCancel()
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

// Stop is a convenience method to gracefully shut down the server, but force-close if the ctx is cancelled.
// The ctx error is not returned when the force-close is successful.
func (s *HTTPServer) Stop(ctx context.Context) error {
	if err := s.Shutdown(ctx); err != nil {
		if errors.Is(err, ctx.Err()) { // force-close connections if we cancelled the stopping
			return s.Close()
		}
		return err
	}
	return nil
}

// Shutdown shuts down the HTTP server and its listener,
// but allows active connections to close gracefully.
// If the function exits due to a ctx cancellation the listener is closed but active connections may remain,
// a call to Close() can force-close any remaining active connections.
func (s *HTTPServer) Shutdown(ctx context.Context) error {
	// closes the underlying listener too.
	return s.srv.Shutdown(ctx)
}

// Close force-closes the HTTPServer, its listener, and all its active connections.
func (s *HTTPServer) Close() error {
	// closes the underlying listener too
	return s.srv.Close()
}

func (s *HTTPServer) Addr() net.Addr {
	return s.listener.Addr()
}

func WithMaxHeaderBytes(max int) HTTPOption {
	return func(srv *HTTPServer) error {
		srv.srv.MaxHeaderBytes = max
		return nil
	}
}
