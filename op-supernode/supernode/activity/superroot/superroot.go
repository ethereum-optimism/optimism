package superroot

import (
	"context"
	"fmt"
	"strings"

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

// AtTimestamp returns a map[chainID]bool indicating if the given timestamp is safe per chain.
func (api *superrootAPI) AtTimestamp(ctx context.Context, ts hexutil.Uint64) (string, error) {
	var b strings.Builder
	b.WriteString("I dont know how to build a superroot yet but here's the data I know I need: \n")

	for chainID, chain := range api.s.chains {
		ok, err := chain.FullyValidAt(ctx, uint64(ts))
		if err != nil {
			return "", err
		}
		if !ok {
			return "", fmt.Errorf("chain %s not fully valid at timestamp %d", chainID.String(), uint64(ts))
		}
		fmt.Fprintf(&b, "Chain %s: Fully valid at timestamp %d\n", chainID.String(), uint64(ts))
	}

	for chainID, chain := range api.s.chains {
		ref, err := chain.BlockAtTimestamp(ctx, uint64(ts))
		if err != nil {
			return "", err
		}
		fmt.Fprintf(&b, "Chain %s: BlockAtTimestamp => number=%d hash=%s time=%d\n", chainID.String(), ref.Number, ref.Hash, ref.Time)
	}

	return b.String(), nil
}
