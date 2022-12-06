package client

import (
	"context"
	"fmt"
	"regexp"

	"github.com/ethereum-optimism/optimism/op-service/backoff"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/log"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/ethereum-optimism/optimism/op-node/metrics"
	"github.com/ethereum/go-ethereum/rpc"
)

var httpRegex = regexp.MustCompile("^http(s)?://")

type RPC interface {
	Close()
	CallContext(ctx context.Context, result any, method string, args ...any) error
	BatchCallContext(ctx context.Context, b []rpc.BatchElem) error
	EthSubscribe(ctx context.Context, channel any, args ...any) (ethereum.Subscription, error)
}

// NewRPC returns the correct client.RPC instance for a given RPC url.
func NewRPC(ctx context.Context, lgr log.Logger, addr string, opts ...rpc.ClientOption) (RPC, error) {
	underlying, err := DialRPCClientWithBackoff(ctx, lgr, addr, opts...)
	if err != nil {
		return nil, err
	}

	wrapped := &BaseRPCClient{
		c: underlying,
	}
	if httpRegex.MatchString(addr) {
		return NewPollingClient(ctx, lgr, wrapped), nil
	}
	return wrapped, nil
}

// Dials a JSON-RPC endpoint repeatedly, with a backoff, until a client connection is established. Auth is optional.
func DialRPCClientWithBackoff(ctx context.Context, log log.Logger, addr string, opts ...rpc.ClientOption) (*rpc.Client, error) {
	bOff := backoff.Exponential()
	var ret *rpc.Client
	err := backoff.DoCtx(ctx, 10, bOff, func() error {
		client, err := rpc.DialOptions(ctx, addr, opts...)
		if err != nil {
			if client == nil {
				return fmt.Errorf("failed to dial address (%s): %w", addr, err)
			}
			log.Warn("failed to dial address, but may connect later", "addr", addr, "err", err)
		}
		ret = client
		return nil
	})
	if err != nil {
		return nil, err
	}
	return ret, nil
}

// BaseRPCClient is a wrapper around a concrete *rpc.Client instance to make it compliant
// with the client.RPC interface.
type BaseRPCClient struct {
	c *rpc.Client
}

func NewBaseRPCClient(c *rpc.Client) *BaseRPCClient {
	return &BaseRPCClient{c: c}
}

func (b *BaseRPCClient) Close() {
	b.c.Close()
}

func (b *BaseRPCClient) CallContext(ctx context.Context, result any, method string, args ...any) error {
	return b.c.CallContext(ctx, result, method, args...)
}

func (b *BaseRPCClient) BatchCallContext(ctx context.Context, batch []rpc.BatchElem) error {
	return b.c.BatchCallContext(ctx, batch)
}

func (b *BaseRPCClient) EthSubscribe(ctx context.Context, channel any, args ...any) (ethereum.Subscription, error) {
	return b.c.EthSubscribe(ctx, channel, args...)
}

// InstrumentedRPCClient is an RPC client that tracks
// Prometheus metrics for each call.
type InstrumentedRPCClient struct {
	c RPC
	m *metrics.Metrics
}

// NewInstrumentedRPC creates a new instrumented RPC client.
func NewInstrumentedRPC(c RPC, m *metrics.Metrics) *InstrumentedRPCClient {
	return &InstrumentedRPCClient{
		c: c,
		m: m,
	}
}

func (ic *InstrumentedRPCClient) Close() {
	ic.c.Close()
}

func (ic *InstrumentedRPCClient) CallContext(ctx context.Context, result any, method string, args ...any) error {
	return instrument1(ic.m, method, func() error {
		return ic.c.CallContext(ctx, result, method, args...)
	})
}

func (ic *InstrumentedRPCClient) BatchCallContext(ctx context.Context, b []rpc.BatchElem) error {
	return instrumentBatch(ic.m, func() error {
		return ic.c.BatchCallContext(ctx, b)
	}, b)
}

func (ic *InstrumentedRPCClient) EthSubscribe(ctx context.Context, channel any, args ...any) (ethereum.Subscription, error) {
	return ic.c.EthSubscribe(ctx, channel, args...)
}

// instrumentBatch handles metrics for batch calls. Request metrics are
// increased for each batch element. Request durations are tracked for
// the batch as a whole using a special <batch> method. Errors are tracked
// for each individual batch response, unless the overall request fails in
// which case the <batch> method is used.
func instrumentBatch(m *metrics.Metrics, cb func() error, b []rpc.BatchElem) error {
	m.RPCClientRequestsTotal.WithLabelValues(metrics.BatchMethod).Inc()
	for _, elem := range b {
		m.RPCClientRequestsTotal.WithLabelValues(elem.Method).Inc()
	}
	timer := prometheus.NewTimer(m.RPCClientRequestDurationSeconds.WithLabelValues(metrics.BatchMethod))
	defer timer.ObserveDuration()

	// Track response times for batch requests separately.
	if err := cb(); err != nil {
		m.RecordRPCClientResponse(metrics.BatchMethod, err)
		return err
	}
	for _, elem := range b {
		m.RecordRPCClientResponse(elem.Method, elem.Error)
	}
	return nil
}
