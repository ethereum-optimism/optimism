package node

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ethereum-optimism/optimism/op-node/client"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/sources"

	"github.com/ethereum/go-ethereum/log"
	gn "github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/rpc"
)

type L2EndpointSetup interface {
	// Setup a RPC client to a L2 execution engine to process rollup blocks with.
	Setup(ctx context.Context, log log.Logger, rollupCfg *rollup.Config) (cl client.RPC, rpcCfg *sources.EngineClientConfig, err error)
	Check() error
}

type L2SyncEndpointSetup interface {
	// Setup a RPC client to another L2 node to sync L2 blocks from.
	// It may return a nil client with nil error if RPC based sync is not enabled.
	Setup(ctx context.Context, log log.Logger, rollupCfg *rollup.Config) (cl client.RPC, rpcCfg *sources.SyncClientConfig, err error)
	Check() error
}

type L1EndpointSetup interface {
	// Setup a RPC client to a L1 node to pull rollup input-data from.
	// The results of the RPC client may be trusted for faster processing, or strictly validated.
	// The kind of the RPC may be non-basic, to optimize RPC usage.
	Setup(ctx context.Context, log log.Logger, rollupCfg *rollup.Config) (cl client.RPC, rpcCfg *sources.L1ClientConfig, err error)
	Check() error
}

type L2EndpointConfig struct {
	L2EngineAddr string // Address of L2 Engine JSON-RPC endpoint to use (engine and eth namespace required)

	// JWT secrets for L2 Engine API authentication during HTTP or initial Websocket communication.
	// Any value for an IPC connection.
	L2EngineJWTSecret [32]byte
}

var _ L2EndpointSetup = (*L2EndpointConfig)(nil)

func (cfg *L2EndpointConfig) Check() error {
	if cfg.L2EngineAddr == "" {
		return errors.New("empty L2 Engine Address")
	}

	return nil
}

func (cfg *L2EndpointConfig) Setup(ctx context.Context, log log.Logger, rollupCfg *rollup.Config) (client.RPC, *sources.EngineClientConfig, error) {
	if err := cfg.Check(); err != nil {
		return nil, nil, err
	}
	auth := rpc.WithHTTPAuth(gn.NewJWTAuth(cfg.L2EngineJWTSecret))
	l2Node, err := client.NewRPC(ctx, log, cfg.L2EngineAddr, client.WithGethRPCOptions(auth))
	if err != nil {
		return nil, nil, err
	}

	return l2Node, sources.EngineClientDefaultConfig(rollupCfg), nil
}

// PreparedL2Endpoints enables testing with in-process pre-setup RPC connections to L2 engines
type PreparedL2Endpoints struct {
	Client client.RPC
}

func (p *PreparedL2Endpoints) Check() error {
	if p.Client == nil {
		return errors.New("client cannot be nil")
	}
	return nil
}

var _ L2EndpointSetup = (*PreparedL2Endpoints)(nil)

func (p *PreparedL2Endpoints) Setup(ctx context.Context, log log.Logger, rollupCfg *rollup.Config) (client.RPC, *sources.EngineClientConfig, error) {
	return p.Client, sources.EngineClientDefaultConfig(rollupCfg), nil
}

// L2SyncEndpointConfig contains configuration for the fallback sync endpoint
type L2SyncEndpointConfig struct {
	// Address of the L2 RPC to use for backup sync, may be empty if RPC alt-sync is disabled.
	L2NodeAddr string
	TrustRPC   bool
}

var _ L2SyncEndpointSetup = (*L2SyncEndpointConfig)(nil)

// Setup creates an RPC client to sync from.
// It will return nil without error if no sync method is configured.
func (cfg *L2SyncEndpointConfig) Setup(ctx context.Context, log log.Logger, rollupCfg *rollup.Config) (client.RPC, *sources.SyncClientConfig, error) {
	if cfg.L2NodeAddr == "" {
		return nil, nil, nil
	}
	l2Node, err := client.NewRPC(ctx, log, cfg.L2NodeAddr)
	if err != nil {
		return nil, nil, err
	}

	return l2Node, sources.SyncClientDefaultConfig(rollupCfg, cfg.TrustRPC), nil
}

func (cfg *L2SyncEndpointConfig) Check() error {
	// empty addr is valid, as it is optional.
	return nil
}

type PreparedL2SyncEndpoint struct {
	// RPC endpoint to use for syncing, may be nil if RPC alt-sync is disabled.
	Client   client.RPC
	TrustRPC bool
}

var _ L2SyncEndpointSetup = (*PreparedL2SyncEndpoint)(nil)

func (cfg *PreparedL2SyncEndpoint) Setup(ctx context.Context, log log.Logger, rollupCfg *rollup.Config) (client.RPC, *sources.SyncClientConfig, error) {
	return cfg.Client, sources.SyncClientDefaultConfig(rollupCfg, cfg.TrustRPC), nil
}

func (cfg *PreparedL2SyncEndpoint) Check() error {
	return nil
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

	// HttpPollInterval specifies the interval between polling for the latest L1 block,
	// when the RPC is detected to be an HTTP type.
	// It is recommended to use websockets or IPC for efficient following of the changing block.
	// Setting this to 0 disables polling.
	HttpPollInterval time.Duration
}

var _ L1EndpointSetup = (*L1EndpointConfig)(nil)

func (cfg *L1EndpointConfig) Check() error {
	if cfg.BatchSize < 1 || cfg.BatchSize > 500 {
		return fmt.Errorf("batch size is invalid or unreasonable: %d", cfg.BatchSize)
	}
	if cfg.RateLimit < 0 {
		return fmt.Errorf("rate limit cannot be negative")
	}
	return nil
}

func (cfg *L1EndpointConfig) Setup(ctx context.Context, log log.Logger, rollupCfg *rollup.Config) (client.RPC, *sources.L1ClientConfig, error) {
	opts := []client.RPCOption{
		client.WithHttpPollInterval(cfg.HttpPollInterval),
		client.WithDialBackoff(10),
	}
	if cfg.RateLimit != 0 {
		opts = append(opts, client.WithRateLimit(cfg.RateLimit, cfg.BatchSize))
	}

	l1Node, err := client.NewRPC(ctx, log, cfg.L1NodeAddr, opts...)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to dial L1 address (%s): %w", cfg.L1NodeAddr, err)
	}
	rpcCfg := sources.L1ClientDefaultConfig(rollupCfg, cfg.L1TrustRPC, cfg.L1RPCKind)
	rpcCfg.MaxRequestsPerBatch = cfg.BatchSize
	return l1Node, rpcCfg, nil
}

// PreparedL1Endpoint enables testing with an in-process pre-setup RPC connection to L1
type PreparedL1Endpoint struct {
	Client          client.RPC
	TrustRPC        bool
	RPCProviderKind sources.RPCProviderKind
}

var _ L1EndpointSetup = (*PreparedL1Endpoint)(nil)

func (p *PreparedL1Endpoint) Setup(ctx context.Context, log log.Logger, rollupCfg *rollup.Config) (client.RPC, *sources.L1ClientConfig, error) {
	return p.Client, sources.L1ClientDefaultConfig(rollupCfg, p.TrustRPC, p.RPCProviderKind), nil
}

func (cfg *PreparedL1Endpoint) Check() error {
	if cfg.Client == nil {
		return errors.New("rpc client cannot be nil")
	}

	return nil
}
