package client

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/backoff"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/log"
	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/time/rate"

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

type rpcConfig struct {
	gethRPCOptions   []rpc.ClientOption
	httpPollInterval time.Duration
	backoffAttempts  int
	limit            float64
	burst            int
}

type RPCOption func(cfg *rpcConfig) error

// WithDialBackoff configures the number of attempts for the initial dial to the RPC,
// attempts are executed with an exponential backoff strategy.
func WithDialBackoff(attempts int) RPCOption {
	return func(cfg *rpcConfig) error {
		cfg.backoffAttempts = attempts
		return nil
	}
}

// WithHttpPollInterval configures the RPC to poll at the given rate, in case RPC subscriptions are not available.
func WithHttpPollInterval(duration time.Duration) RPCOption {
	return func(cfg *rpcConfig) error {
		cfg.httpPollInterval = duration
		return nil
	}
}

// WithGethRPCOptions passes the list of go-ethereum RPC options to the internal RPC instance.
func WithGethRPCOptions(gethRPCOptions ...rpc.ClientOption) RPCOption {
	return func(cfg *rpcConfig) error {
		cfg.gethRPCOptions = append(cfg.gethRPCOptions, gethRPCOptions...)
		return nil
	}
}

// WithRateLimit configures the RPC to target the given rate limit (in requests / second).
// See NewRateLimitingClient for more details.
func WithRateLimit(rateLimit float64, burst int) RPCOption {
	return func(cfg *rpcConfig) error {
		cfg.limit = rateLimit
		cfg.burst = burst
		return nil
	}
}

// NewRPC returns the correct client.RPC instance for a given RPC url.
func NewRPC(ctx context.Context, lgr log.Logger, addr string, opts ...RPCOption) (RPC, error) {
	var cfg rpcConfig
	for i, opt := range opts {
		if err := opt(&cfg); err != nil {
			return nil, fmt.Errorf("rpc option %d failed to apply to RPC config: %w", i, err)
		}
	}
	if cfg.backoffAttempts < 1 { // default to at least 1 attempt, or it always fails to dial.
		cfg.backoffAttempts = 1
	}
	underlying, err := dialRPCClientWithBackoff(ctx, lgr, addr, cfg.backoffAttempts, cfg.gethRPCOptions...)
	if err != nil {
		return nil, err
	}

	var wrapped RPC = &BaseRPCClient{c: underlying}

	if cfg.limit != 0 {
		wrapped = NewRateLimitingClient(wrapped, rate.Limit(cfg.limit), cfg.burst)
	}

	if httpRegex.MatchString(addr) {
		wrapped = NewPollingClient(ctx, lgr, wrapped, WithPollRate(cfg.httpPollInterval))
	}

	return wrapped, nil
}

// Dials a JSON-RPC endpoint repeatedly, with a backoff, until a client connection is established. Auth is optional.
func dialRPCClientWithBackoff(ctx context.Context, log log.Logger, addr string, attempts int, opts ...rpc.ClientOption) (*rpc.Client, error) {
	bOff := backoff.Exponential()
	var ret *rpc.Client
	err := backoff.DoCtx(ctx, attempts, bOff, func() error {
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
// It sets a timeout of 10s on CallContext & 20s on BatchCallContext made through it.
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
	cCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	return b.c.CallContext(cCtx, result, method, args...)
}

func (b *BaseRPCClient) BatchCallContext(ctx context.Context, batch []rpc.BatchElem) error {
	cCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	return b.c.BatchCallContext(cCtx, batch)
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
