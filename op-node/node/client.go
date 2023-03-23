package node

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-node/client"
	"github.com/ethereum-optimism/optimism/op-node/sources"

	"github.com/ethereum/go-ethereum/log"
	gn "github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/rpc"
)

type L2EndpointSetup interface {
	// Setup a RPC client to a L2 execution engine to process rollup blocks with.
	Setup(ctx context.Context, log log.Logger) (cl client.RPC, err error)
	Check() error
}

type L2SyncEndpointSetup interface {
	// Setup a RPC client to another L2 node to sync L2 blocks from.
	// It may return a nil client with nil error if RPC based sync is not enabled.
	Setup(ctx context.Context, log log.Logger) (cl client.RPC, trust bool, err error)
	Check() error
}

type L1EndpointSetup interface {
	// Setup a RPC client to a L1 node to pull rollup input-data from.
	// The results of the RPC client may be trusted for faster processing, or strictly validated.
	// The kind of the RPC may be non-basic, to optimize RPC usage.
	Setup(ctx context.Context, log log.Logger) (cl client.RPC, trust bool, kind sources.RPCProviderKind, err error)
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

func (cfg *L2EndpointConfig) Setup(ctx context.Context, log log.Logger) (client.RPC, error) {
	if err := cfg.Check(); err != nil {
		return nil, err
	}
	auth := rpc.WithHTTPAuth(gn.NewJWTAuth(cfg.L2EngineJWTSecret))
	l2Node, err := client.NewRPC(ctx, log, cfg.L2EngineAddr, auth)
	if err != nil {
		return nil, err
	}

	return l2Node, nil
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

func (p *PreparedL2Endpoints) Setup(ctx context.Context, log log.Logger) (client.RPC, error) {
	return p.Client, nil
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
func (cfg *L2SyncEndpointConfig) Setup(ctx context.Context, log log.Logger) (cl client.RPC, trust bool, err error) {
	if cfg.L2NodeAddr == "" {
		return nil, false, nil
	}
	l2Node, err := client.NewRPC(ctx, log, cfg.L2NodeAddr)
	if err != nil {
		return nil, false, err
	}

	return l2Node, cfg.TrustRPC, nil
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

func (cfg *PreparedL2SyncEndpoint) Setup(ctx context.Context, log log.Logger) (cl client.RPC, trust bool, err error) {
	return cfg.Client, cfg.TrustRPC, nil
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
}

var _ L1EndpointSetup = (*L1EndpointConfig)(nil)

func (cfg *L1EndpointConfig) Setup(ctx context.Context, log log.Logger) (cl client.RPC, trust bool, kind sources.RPCProviderKind, err error) {
	l1Node, err := client.NewRPC(ctx, log, cfg.L1NodeAddr)
	if err != nil {
		return nil, false, sources.RPCKindBasic, fmt.Errorf("failed to dial L1 address (%s): %w", cfg.L1NodeAddr, err)
	}
	return l1Node, cfg.L1TrustRPC, cfg.L1RPCKind, nil
}

// PreparedL1Endpoint enables testing with an in-process pre-setup RPC connection to L1
type PreparedL1Endpoint struct {
	Client          client.RPC
	TrustRPC        bool
	RPCProviderKind sources.RPCProviderKind
}

var _ L1EndpointSetup = (*PreparedL1Endpoint)(nil)

func (p *PreparedL1Endpoint) Setup(ctx context.Context, log log.Logger) (cl client.RPC, trust bool, kind sources.RPCProviderKind, err error) {
	return p.Client, p.TrustRPC, p.RPCProviderKind, nil
}
