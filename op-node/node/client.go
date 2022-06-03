package node

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-node/backoff"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
)

type L2EndpointsSetup interface {
	// Setup a RPC client to a L2 execution engine to process rollup blocks with.
	Setup(ctx context.Context, log log.Logger) (cl []*rpc.Client, err error)
	Check() error
}

type L1EndpointSetup interface {
	// Setup a RPC client to a L1 node to pull rollup input-data from.
	Setup(ctx context.Context, log log.Logger) (cl *rpc.Client, trust bool, err error)
}

type L2EndpointsConfig struct {
	L2EngineAddrs []string // Addresses of L2 Engine JSON-RPC endpoints to use (engine and eth namespace required)

	// JWT secrets for L2 Engine API authentication during HTTP or initial Websocket communication, one per L2 engine.
	// Any value for an IPC connection.
	L2EngineJWTSecrets [][32]byte
}

var _ L2EndpointsSetup = (*L2EndpointsConfig)(nil)

func (cfg *L2EndpointsConfig) Check() error {
	if len(cfg.L2EngineAddrs) == 0 {
		return errors.New("need at least one L2 engine to connect to")
	}
	if len(cfg.L2EngineAddrs) != len(cfg.L2EngineJWTSecrets) {
		return fmt.Errorf("have %d L2 engines, but %d authentication secrets", len(cfg.L2EngineAddrs), len(cfg.L2EngineJWTSecrets))
	}
	return nil
}

func (cfg *L2EndpointsConfig) Setup(ctx context.Context, log log.Logger) ([]*rpc.Client, error) {
	if err := cfg.Check(); err != nil {
		return nil, err
	}
	var out []*rpc.Client
	for i, addr := range cfg.L2EngineAddrs {
		auth := rpc.NewJWTAuthProvider(cfg.L2EngineJWTSecrets[i])
		l2Node, err := dialRPCClientWithBackoff(ctx, log, addr, auth)
		if err != nil {
			// close clients again if we cannot complete the full setup
			for _, cl := range out {
				cl.Close()
			}
			return out, err
		}
		out = append(out, l2Node)
	}
	return out, nil
}

// PreparedL2Endpoints enables testing with in-process pre-setup RPC connections to L2 engines
type PreparedL2Endpoints struct {
	Clients []*rpc.Client
}

func (p *PreparedL2Endpoints) Check() error {
	if len(p.Clients) == 0 {
		return errors.New("need at least one L2 engine to connect to")
	}
	return nil
}

var _ L2EndpointsSetup = (*PreparedL2Endpoints)(nil)

func (p *PreparedL2Endpoints) Setup(ctx context.Context, log log.Logger) ([]*rpc.Client, error) {
	return p.Clients, nil
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
	l1Node, err := dialRPCClientWithBackoff(ctx, log, cfg.L1NodeAddr, nil)
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
func dialRPCClientWithBackoff(ctx context.Context, log log.Logger, addr string, auth rpc.HeaderAuthProvider) (*rpc.Client, error) {
	bOff := backoff.Exponential()
	var ret *rpc.Client
	err := backoff.Do(10, bOff, func() error {
		var client *rpc.Client
		var err error
		if auth == nil {
			client, err = rpc.DialContext(ctx, addr)
		} else {
			client, err = rpc.DialWithAuth(ctx, addr, auth)
		}
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
