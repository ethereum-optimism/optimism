package eth

import (
	"fmt"

	"github.com/ethereum/go-ethereum/params"
)

func L1ChainConfigByChainID(chainID ChainID) (*params.ChainConfig, error) {
	switch chainID {
	case ChainIDFromBig(params.MainnetChainConfig.ChainID):
		return params.MainnetChainConfig, nil
	case ChainIDFromBig(params.SepoliaChainConfig.ChainID):
		return params.SepoliaChainConfig, nil
	case ChainIDFromBig(params.HoleskyChainConfig.ChainID):
		return params.HoleskyChainConfig, nil
	case ChainIDFromBig(params.HoodiChainConfig.ChainID):
		return params.HoodiChainConfig, nil
	default:
		return nil, fmt.Errorf("unknown chain ID: %v", chainID)
	}
}
