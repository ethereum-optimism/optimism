package superchain

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common/math"

	"github.com/ethereum-optimism/optimism/op-core/interop/depset"
	registry "github.com/ethereum-optimism/optimism/op-core/superchain"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// LoadDependencySet loads the dependency set containing the requested chain ID
// from the superchain-registry.
// Returns an error matching registry.ErrUnknownChain if the chain is not
// available in the superchain-registry.
func LoadDependencySet(chainID eth.ChainID) (depset.DependencySet, error) {
	id, ok := chainID.Uint64()
	if !ok {
		return nil, fmt.Errorf("%w: %v", registry.ErrUnknownChain, chainID)
	}
	depSet, err := registry.GetDepset(id)
	if err != nil {
		return nil, err
	}
	chains := make(map[eth.ChainID]*depset.StaticConfigDependency)
	for idStr := range depSet {
		depChainID, ok := math.ParseUint64(idStr)
		if !ok {
			return nil, fmt.Errorf("invalid chain ID in dependency set: %s", idStr)
		}
		chains[eth.ChainIDFromUInt64(depChainID)] = &depset.StaticConfigDependency{}
	}
	return depset.NewStaticConfigDependencySet(chains)
}
