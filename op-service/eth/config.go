package eth

import (
	"github.com/ethereum/go-ethereum/params"
)

// L1ChainConfigByChainID returns the chain config for the given chain ID,
// if it is in the set of known chain IDs (Mainnet, Sepolia, Holesky, Hoodi).
// If the chain ID is not known, it returns nil.
func L1ChainConfigByChainID(chainID ChainID) *params.ChainConfig {
	switch chainID {
	case ChainIDFromBig(params.MainnetChainConfig.ChainID):
		return params.MainnetChainConfig
	case ChainIDFromBig(params.SepoliaChainConfig.ChainID):
		return params.SepoliaChainConfig
	case ChainIDFromBig(params.HoleskyChainConfig.ChainID):
		return params.HoleskyChainConfig
	case ChainIDFromBig(params.HoodiChainConfig.ChainID):
		return params.HoodiChainConfig
	default:
		return nil
	}
}
