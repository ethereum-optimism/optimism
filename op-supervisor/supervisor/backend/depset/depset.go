package depset

import (
	"context"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/superchain"
)

type DependencySetSource interface {
	LoadDependencySet(ctx context.Context) (DependencySet, error)
}

// DependencySet is an initialized dependency set, ready to answer queries
// of what is and what is not part of the dependency set.
type DependencySet interface {
	// Chains returns the list of chains that are part of the dependency set.
	Chains() []eth.ChainID

	// HasChain determines if a chain is being tracked for interop purposes.
	// See CanExecuteAt and CanInitiateAt to check if a chain may message at a given time.
	HasChain(chainID eth.ChainID) bool

	// MessageExpiryWindow returns the message expiry window to use for this dependency set.
	MessageExpiryWindow() uint64
}

// FromRegistry loads a dependency set from the superchain-registry.
// Returns error of type superchain.ErrUnknownChain if the chain is not available in the superchain registry.
func FromRegistry(chainID eth.ChainID) (DependencySet, error) {
	id, ok := chainID.Uint64()
	if !ok {
		return nil, fmt.Errorf("%w: %v", superchain.ErrUnknownChain, chainID)
	}
	depSet, err := superchain.GetDepset(id)
	if err != nil {
		return nil, err
	}
	chains := make(map[eth.ChainID]*StaticConfigDependency)
	for idStr := range depSet {
		id, ok := math.ParseUint64(idStr)
		if !ok {
			return nil, fmt.Errorf("invalid chain ID in dependency set: %s", idStr)
		}
		chains[eth.ChainIDFromUInt64(id)] = &StaticConfigDependency{}
	}
	return NewStaticConfigDependencySet(chains)
}
