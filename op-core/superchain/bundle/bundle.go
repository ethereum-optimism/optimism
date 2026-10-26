// Package bundle is the superchain-registry built into the monorepo: it embeds
// superchain-configs.zip, which sync-superchain.sh builds from the superchain-registry submodule.
//
// The zip is gitignored, so it is absent from the published module and this package does not
// compile outside the monorepo. Code that only needs the registry's types or a loader for its
// own bundle imports [superchain] instead.
package bundle

import (
	_ "embed"

	"github.com/ethereum-optimism/optimism/op-core/superchain"
)

//go:embed superchain-configs.zip
var configData []byte

var builtIn *superchain.ChainConfigLoader

// Chains holds the built-in registry's chains, keyed by chain ID.
var Chains map[uint64]*superchain.Chain

func init() {
	if err := verifyEmbeddedBundle(); err != nil {
		panic("op-core/superchain/bundle: " + err.Error())
	}
	loader, err := superchain.NewChainConfigLoader(configData)
	if err != nil {
		panic("op-core/superchain/bundle: " + err.Error())
	}
	builtIn, Chains = loader, loader.Chains
}

// GetChain returns the built-in registry's chain with the given chain ID.
func GetChain(chainID uint64) (*superchain.Chain, error) {
	return builtIn.GetChain(chainID)
}

// ChainIDByName returns the chain ID of the built-in registry's chain named "<name>-<network>".
func ChainIDByName(name string) (uint64, error) {
	return builtIn.ChainIDByName(name)
}

// ChainNames returns the sorted "<name>-<network>" names of the built-in registry's chains.
func ChainNames() []string {
	return builtIn.ChainNames()
}

// GetSuperchain returns the built-in registry's superchain config of the given network.
func GetSuperchain(network string) (superchain.Superchain, error) {
	return builtIn.GetSuperchain(network)
}
