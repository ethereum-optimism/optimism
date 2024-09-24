package client

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/rpc"
)

// LazyRPC defers connection attempts to the usage of the RPC.
// This allows a websocket connection to be established lazily.
// The underlying RPC should handle reconnects.
type LazyRPC struct {
	// mutex to prevent more than one active dial attempt at a time.
	mu sync.Mutex
	// inner is the actual RPC client.
	// It is initialized once. The underlying RPC handles reconnections.
	inner RPC
	// options to initialize `inner` with.
	opts     []rpc.ClientOption
	endpoint string
	// If we have not initialized `inner` yet,
	// do not try to do so after closing the client.
	closed bool
}

var _ RPC = (*LazyRPC)(nil)

func NewLazyRPC(endpoint string, opts ...rpc.ClientOption) *LazyRPC {
	return &LazyRPC{
		opts:     opts,
		endpoint: endpoint,
	}
}

func (l *LazyRPC) dial(ctx context.Context) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.inner != nil {
		return nil
	}
	if l.closed {
		return errors.New("cannot dial RPC, client was already closed")
	}
	underlying, err := rpc.DialOptions(ctx, l.endpoint, l.opts...)
	if err != nil {
		return fmt.Errorf("failed to dial: %w", err)
	}
	l.inner = NewBaseRPCClient(underlying)
	return nil
}

func (l *LazyRPC) Close() {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.inner != nil {
		l.inner.Close()
	}
	l.closed = true
}

func (l *LazyRPC) CallContext(ctx context.Context, result any, method string, args ...any) error {
	if err := l.dial(ctx); err != nil {
		return err
	}
	return l.inner.CallContext(ctx, result, method, args...)
}

func (l *LazyRPC) BatchCallContext(ctx context.Context, b []rpc.BatchElem) error {
	if err := l.dial(ctx); err != nil {
		return err
	}
	return l.inner.BatchCallContext(ctx, b)
}

func (l *LazyRPC) EthSubscribe(ctx context.Context, channel any, args ...any) (ethereum.Subscription, error) {
	if err := l.dial(ctx); err != nil {
		return nil, err
	}
	return l.inner.EthSubscribe(ctx, channel, args...)
}
