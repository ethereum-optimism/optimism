package superroot

import (
	"context"
	"fmt"
	"slices"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	cc "github.com/ethereum-optimism/optimism/op-supernode/supernode/chain_container"
	"github.com/ethereum/go-ethereum/common/hexutil"
	gethlog "github.com/ethereum/go-ethereum/log"
)

// Superroot is an Activity that operates across all chain containers, constructing the superroot at a given timestamp.
type Superroot struct {
	log    gethlog.Logger
	chains map[eth.ChainID]cc.ChainContainer
}

// New creates a new Superroot rpc activity
func New(log gethlog.Logger, chains map[eth.ChainID]cc.ChainContainer) *Superroot {
	return &Superroot{
		log:    log,
		chains: chains,
	}
}

// ActivityName returns the routing name for this activity.
func (s *Superroot) ActivityName() string { return "superroot" }

// RPCAPIs implements RPCActivity by returning a JSON-RPC API namespace for superroot.
func (s *Superroot) RPCNamespace() string    { return "superroot" }
func (s *Superroot) RPCService() interface{} { return &superrootAPI{s: s} }

// superrootAPI hosts JSON-RPC methods for the Superroot activity.
type superrootAPI struct{ s *Superroot }

// AtTimestamp computes the super-root at the given timestamp.
func (api *superrootAPI) AtTimestamp(ctx context.Context, ts hexutil.Uint64) (eth.SuperRootResponse, error) {
	// Ensure all safe data is available for the given timestamp
	// Not the same as confirming the data is verified by all Verification Activities
	for chainID, chain := range api.s.chains {
		ok, err := chain.SafeAtTimestamp(ctx, uint64(ts))
		if err != nil {
			return eth.SuperRootResponse{}, err
		}
		if !ok {
			return eth.SuperRootResponse{}, fmt.Errorf("chain %s not safe at timestamp %d", chainID.String(), uint64(ts))
		}
	}

	// Gather per-chain output roots
	chains := make([]eth.ChainRootInfo, 0, len(api.s.chains))
	outs := make([]eth.ChainIDAndOutput, 0, len(api.s.chains))
	for chainID, chain := range api.s.chains {
		out, err := chain.OutputRootAtTimestamp(ctx, uint64(ts))
		if err != nil {
			return eth.SuperRootResponse{}, err
		}
		chains = append(chains, eth.ChainRootInfo{
			ChainID:   chainID,
			Canonical: out,
			Pending:   nil,
		})
		outs = append(outs, eth.ChainIDAndOutput{ChainID: chainID, Output: out})
	}

	// Sort chains for deterministic RPC response ordering
	slices.SortFunc(chains, func(a, b eth.ChainRootInfo) int { return a.ChainID.Cmp(b.ChainID) })

	// Build SuperV1 and compute root (constructor sorts outs internally)
	super := eth.NewSuperV1(uint64(ts), outs...)
	superRoot := eth.SuperRoot(super)

	// Assemble response
	resp := eth.SuperRootResponse{
		Timestamp: uint64(ts),
		SuperRoot: superRoot,
		Version:   eth.SuperRootVersionV1,
		Chains:    chains,
	}
	return resp, nil
}
