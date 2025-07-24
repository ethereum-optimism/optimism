package config

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-service/client"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum-optimism/optimism/op-service/sources"
)

type L1EndpointSetup interface {
	// Setup a RPC client to a L1 node to pull rollup input-data from.
	// The results of the RPC client may be trusted for faster processing, or strictly validated.
	// The kind of the RPC may be non-basic, to optimize RPC usage.
	Setup(ctx context.Context, log log.Logger, defaultCacheSize int, metrics opmetrics.RPCMetricer) (cl client.RPC, rpcCfg *sources.L1ClientConfig, err error)
	Check() error
}

type L1EndpointConfig struct {
	L1NodeAddr string // Address of L1 User JSON-RPC endpoint to use (eth namespace required)

	// L1TrustRPC: if we trust the L1 RPC we do not have to validate L1 response contents like headers
	// against block hashes, or cached transaction sender addresses.
	// Thus we can sync faster at the risk of the source RPC being wrong.
	L1TrustRPC bool

	// L1RPCKind identifies the RPC provider kind that serves the RPC,
	// to inform the optimal usage of the RPC for transaction receipts fetching.
	L1RPCKind sources.RPCProviderKind

	// RateLimit specifies a self-imposed rate-limit on L1 requests. 0 is no rate-limit.
	RateLimit float64

	// BatchSize specifies the maximum batch-size, which also applies as L1 rate-limit burst amount (if set).
	BatchSize int

	// MaxConcurrency specifies the maximum number of concurrent requests to the L1 RPC.
	MaxConcurrency int

	// HttpPollInterval specifies the interval between polling for the latest L1 block,
	// when the RPC is detected to be an HTTP type.
	// It is recommended to use websockets or IPC for efficient following of the changing block.
	// Setting this to 0 disables polling.
	HttpPollInterval time.Duration

	// CacheSize specifies the cache size for blocks, receipts and transactions. It's optional and a
	// sane default of 3/2 the sequencing window size is used during Setup if this field is set to 0.
	// Note that receipts and transactions are cached per block, which is why there's only one cache
	// size to configure.
	CacheSize uint
}

var _ L1EndpointSetup = (*L1EndpointConfig)(nil)

func (cfg *L1EndpointConfig) Check() error {
	if cfg.BatchSize < 1 || cfg.BatchSize > 500 {
		return fmt.Errorf("batch size is invalid or unreasonable: %d", cfg.BatchSize)
	}
	if cfg.RateLimit < 0 {
		return fmt.Errorf("rate limit cannot be negative: %f", cfg.RateLimit)
	}
	if cfg.MaxConcurrency < 1 {
		return fmt.Errorf("max concurrent requests cannot be less than 1, was %d", cfg.MaxConcurrency)
	}
	if cfg.CacheSize > 1_000_000 {
		return fmt.Errorf("cache size is dangerously large: %d", cfg.CacheSize)
	}
	return nil
}

func (cfg *L1EndpointConfig) Setup(ctx context.Context, log log.Logger, defaultCacheSize int, metrics opmetrics.RPCMetricer) (client.RPC, *sources.L1ClientConfig, error) {
	opts := []client.RPCOption{
		client.WithHttpPollInterval(cfg.HttpPollInterval),
		client.WithDialAttempts(10),
		client.WithRPCRecorder(metrics.NewRecorder("l1")),
	}
	if cfg.RateLimit != 0 {
		opts = append(opts, client.WithRateLimit(cfg.RateLimit, cfg.BatchSize))
	}

	l1RPC, err := client.NewRPC(ctx, log, cfg.L1NodeAddr, opts...)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to dial L1 address (%s): %w", cfg.L1NodeAddr, err)
	}

	cacheSize := defaultCacheSize
	if cfg.CacheSize > 0 {
		cacheSize = int(cfg.CacheSize)
	}
	l1Cfg := sources.L1ClientSimpleConfig(cfg.L1TrustRPC, cfg.L1RPCKind, cacheSize)
	l1Cfg.MaxRequestsPerBatch = cfg.BatchSize
	l1Cfg.MaxConcurrentRequests = cfg.MaxConcurrency
	return l1RPC, l1Cfg, nil
}

// PreparedL1Endpoint enables testing with an in-process pre-setup RPC connection to L1
type PreparedL1Endpoint struct {
	Client          client.RPC
	TrustRPC        bool
	RPCProviderKind sources.RPCProviderKind
}

var _ L1EndpointSetup = (*PreparedL1Endpoint)(nil)

func (p *PreparedL1Endpoint) Setup(ctx context.Context, log log.Logger, defaultCacheSize int, metrics opmetrics.RPCMetricer) (client.RPC, *sources.L1ClientConfig, error) {
	return p.Client, sources.L1ClientSimpleConfig(p.TrustRPC, p.RPCProviderKind, defaultCacheSize), nil
}

func (cfg *PreparedL1Endpoint) Check() error {
	if cfg.Client == nil {
		return errors.New("rpc client cannot be nil")
	}

	return nil
}
