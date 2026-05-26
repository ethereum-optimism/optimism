package superchain

import (
	"fmt"

	"github.com/ethereum/go-ethereum/params"
	gethsuperchain "github.com/ethereum/go-ethereum/superchain"
)

// LoadChainConfigFromChainID loads the OP-Stack params.ChainConfig for the
// given L2 chain ID from the superchain registry.
//
// TODO(20271): once the params.ChainConfig type and the
// params.LoadOPStackChainConfig function have been migrated into this
// repository, this wrapper should use the local ChainConfig loader instead of
// the op-geth one.
func LoadChainConfigFromChainID(chainID uint64) (*params.ChainConfig, error) {
	chain, err := gethsuperchain.GetChain(chainID)
	if err != nil {
		return nil, fmt.Errorf("unable to get chain %d from superchain registry: %w", chainID, err)
	}
	chainCfg, err := chain.Config()
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve chain %d config: %w", chainID, err)
	}
	return params.LoadOPStackChainConfig(chainCfg)
}
