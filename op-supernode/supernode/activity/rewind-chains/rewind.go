package rewind

import (
	"context"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supernode/supernode/activity"
	cc "github.com/ethereum-optimism/optimism/op-supernode/supernode/chain_container"
	gethlog "github.com/ethereum/go-ethereum/log"
)

// compile time assertions
var (
	_ activity.RPCActivity = (*Rewind)(nil)
)

// Activity that allows any chain to be rewound to a specific timestamp via RPC.
type Rewind struct {
	log    gethlog.Logger
	chains map[eth.ChainID]cc.ChainContainer
	ctx    context.Context
	cancel context.CancelFunc
}

// New creates a new Heartbeat activity.
func New(log gethlog.Logger, chains map[eth.ChainID]cc.ChainContainer) *Rewind {
	return &Rewind{log: log, chains: chains}
}

// RPCNamespace returns the JSON-RPC namespace for this activity.
func (r *Rewind) RPCNamespace() string { return "rewind" }

// RPCService returns the service object whose exported methods are exposed in RPC.
func (r *Rewind) RPCService() interface{} { return (*api)(r) }

// api hosts JSON-RPC methods for the Heartbeat activity.
type api Rewind

// Check returns a random 4-byte for liveness.
func (a *api) ChainTo(ctx context.Context, chainID eth.ChainID, timestamp uint64) error {
	cc, ok := a.chains[chainID]
	if !ok {
		return fmt.Errorf("chain %d not found", chainID)
	}
	return cc.RewindEngine(ctx, timestamp)
}
