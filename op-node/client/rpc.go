package client

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-node/metrics"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/prometheus/client_golang/prometheus"
)

type RPC interface {
	Close()
	CallContext(ctx context.Context, result interface{}, method string, args ...interface{}) error
	BatchCallContext(ctx context.Context, b []rpc.BatchElem) error
	EthSubscribe(ctx context.Context, channel interface{}, args ...interface{}) (*rpc.ClientSubscription, error)
}

// InstrumentedRPCClient is an RPC client that tracks
// Prometheus metrics for each call.
type InstrumentedRPCClient struct {
	c *rpc.Client
	m *metrics.Metrics
}

// NewInstrumentedRPC creates a new instrumented RPC client. It takes
// a concrete *rpc.Client to prevent people from passing in an already
// instrumented client.
func NewInstrumentedRPC(c *rpc.Client, m *metrics.Metrics) *InstrumentedRPCClient {
	return &InstrumentedRPCClient{
		c: c,
		m: m,
	}
}

func (ic *InstrumentedRPCClient) Close() {
	ic.c.Close()
}

func (ic *InstrumentedRPCClient) CallContext(ctx context.Context, result interface{}, method string, args ...interface{}) error {
	return instrument1(ic.m, method, func() error {
		return ic.c.CallContext(ctx, result, method, args...)
	})
}

func (ic *InstrumentedRPCClient) BatchCallContext(ctx context.Context, b []rpc.BatchElem) error {
	return instrumentBatch(ic.m, func() error {
		return ic.c.BatchCallContext(ctx, b)
	}, b)
}

func (ic *InstrumentedRPCClient) EthSubscribe(ctx context.Context, channel interface{}, args ...interface{}) (*rpc.ClientSubscription, error) {
	return ic.c.EthSubscribe(ctx, channel, args...)
}

func (ic *InstrumentedRPCClient) Client() Client {
	return NewInstrumentedClient(ic.c, ic.m)
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
