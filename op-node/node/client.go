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

type L1EndpointSetup interface {
	// Setup a RPC client to a L1 node to pull rollup input-data from.
	// The results of the RPC client may be trusted for faster processing, or strictly validated.
	// The kind of the RPC may be non-basic, to optimize RPC usage.
	Setup(ctx context.Context, log log.Logger) (cl client.RPC, trust bool, kind sources.RPCProviderKind, err error)
}

type L1BeaconEndpointSetup interface {
	Setup(ctx context.Context, log log.Logger) (cl client.HTTP, err error)
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

type L1BeaconEndpointConfig struct {
	BeaconAddr string // Address of L1 User Beacon-API endpoint to use (beacon namespace required)
}

var _ L1BeaconEndpointSetup = (*L1BeaconEndpointConfig)(nil)

func (cfg *L1BeaconEndpointConfig) Setup(ctx context.Context, log log.Logger) (cl client.HTTP, err error) {
	return client.NewBasicHTTPClient(cfg.BeaconAddr, log), nil
}

func (cfg *L1BeaconEndpointConfig) Check() error {
	if cfg.BeaconAddr != "" {
		return fmt.Errorf("expected beacon address, but got none")
	}
	return nil
}

// PreparedL1BeaconEndpoint enables testing with an in-process pre-setup connection to a beacon node mock
type PreparedL1BeaconEndpoint struct {
	Client client.HTTP
}

var _ L1BeaconEndpointSetup = (*PreparedL1BeaconEndpoint)(nil)

func (p *PreparedL1BeaconEndpoint) Setup(ctx context.Context, log log.Logger) (cl client.HTTP, err error) {
	return p.Client, nil
}

func (p *PreparedL1BeaconEndpoint) Check() error {
	if p.Client == nil {
		return fmt.Errorf("expected beacon client, but got none")
	}
	return nil
}
