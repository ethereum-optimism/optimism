package node

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-node/backoff"
	"github.com/ethereum/go-ethereum/log"
	gn "github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/rpc"
)

type L2EndpointSetup interface {
	// Setup a RPC client to a L2 execution engine to process rollup blocks with.
	Setup(ctx context.Context, log log.Logger) (cl *rpc.Client, err error)
	Check() error
}

type L1EndpointSetup interface {
	// Setup a RPC client to a L1 node to pull rollup input-data from.
	Setup(ctx context.Context, log log.Logger) (cl *rpc.Client, trust bool, err error)
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

func (cfg *L2EndpointConfig) Setup(ctx context.Context, log log.Logger) (*rpc.Client, error) {
	if err := cfg.Check(); err != nil {
		return nil, err
	}
	auth := rpc.WithHTTPAuth(gn.NewJWTAuth(cfg.L2EngineJWTSecret))
	l2Node, err := dialRPCClientWithBackoff(ctx, log, cfg.L2EngineAddr, auth)
	if err != nil {
		return nil, err
	}

	return l2Node, nil
}

// PreparedL2Endpoints enables testing with in-process pre-setup RPC connections to L2 engines
type PreparedL2Endpoints struct {
	Client *rpc.Client
}

func (p *PreparedL2Endpoints) Check() error {
	if p.Client == nil {
		return errors.New("client cannot be nil")
	}
	return nil
}

var _ L2EndpointSetup = (*PreparedL2Endpoints)(nil)

func (p *PreparedL2Endpoints) Setup(ctx context.Context, log log.Logger) (*rpc.Client, error) {
	return p.Client, nil
}

type L1EndpointConfig struct {
	L1NodeAddr string // Address of L1 User JSON-RPC endpoint to use (eth namespace required)

	// L1TrustRPC: if we trust the L1 RPC we do not have to validate L1 response contents like headers
	// against block hashes, or cached transaction sender addresses.
	// Thus we can sync faster at the risk of the source RPC being wrong.
	L1TrustRPC bool
}

var _ L1EndpointSetup = (*L1EndpointConfig)(nil)

func (cfg *L1EndpointConfig) Setup(ctx context.Context, log log.Logger) (cl *rpc.Client, trust bool, err error) {
	l1Node, err := dialRPCClientWithBackoff(ctx, log, cfg.L1NodeAddr)
	if err != nil {
		return nil, false, fmt.Errorf("failed to dial L1 address (%s): %w", cfg.L1NodeAddr, err)
	}
	return l1Node, cfg.L1TrustRPC, nil
}

// PreparedL1Endpoint enables testing with an in-process pre-setup RPC connection to L1
type PreparedL1Endpoint struct {
	Client   *rpc.Client
	TrustRPC bool
}

var _ L1EndpointSetup = (*PreparedL1Endpoint)(nil)

func (p *PreparedL1Endpoint) Setup(ctx context.Context, log log.Logger) (cl *rpc.Client, trust bool, err error) {
	return p.Client, p.TrustRPC, nil
}

// Dials a JSON-RPC endpoint repeatedly, with a backoff, until a client connection is established. Auth is optional.
func dialRPCClientWithBackoff(ctx context.Context, log log.Logger, addr string, opts ...rpc.ClientOption) (*rpc.Client, error) {
	bOff := backoff.Exponential()
	var ret *rpc.Client
	err := backoff.Do(10, bOff, func() error {
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
