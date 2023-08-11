package chainconfig

import (
	"fmt"
	"math/big"
	"strconv"

	"github.com/ethereum-optimism/optimism/op-node/chaincfg"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum/go-ethereum/params"
)

var enabledFromBedrockBlock = uint64(0)

var OPGoerliChainConfig = &params.ChainConfig{
	ChainID:                       big.NewInt(420),
	HomesteadBlock:                big.NewInt(0),
	DAOForkBlock:                  nil,
	DAOForkSupport:                false,
	EIP150Block:                   big.NewInt(0),
	EIP155Block:                   big.NewInt(0),
	EIP158Block:                   big.NewInt(0),
	ByzantiumBlock:                big.NewInt(0),
	ConstantinopleBlock:           big.NewInt(0),
	PetersburgBlock:               big.NewInt(0),
	IstanbulBlock:                 big.NewInt(0),
	MuirGlacierBlock:              big.NewInt(0),
	BerlinBlock:                   big.NewInt(0),
	LondonBlock:                   big.NewInt(4061224),
	ArrowGlacierBlock:             big.NewInt(4061224),
	GrayGlacierBlock:              big.NewInt(4061224),
	MergeNetsplitBlock:            big.NewInt(4061224),
	BedrockBlock:                  big.NewInt(4061224),
	RegolithTime:                  &params.OptimismGoerliRegolithTime,
	TerminalTotalDifficulty:       big.NewInt(0),
	TerminalTotalDifficultyPassed: true,
	Optimism: &params.OptimismConfig{
		EIP1559Elasticity:  10,
		EIP1559Denominator: 50,
	},
}

var OPSepoliaChainConfig = &params.ChainConfig{
	ChainID:                       big.NewInt(11155420),
	HomesteadBlock:                big.NewInt(0),
	DAOForkBlock:                  nil,
	DAOForkSupport:                false,
	EIP150Block:                   big.NewInt(0),
	EIP155Block:                   big.NewInt(0),
	EIP158Block:                   big.NewInt(0),
	ByzantiumBlock:                big.NewInt(0),
	ConstantinopleBlock:           big.NewInt(0),
	PetersburgBlock:               big.NewInt(0),
	IstanbulBlock:                 big.NewInt(0),
	MuirGlacierBlock:              big.NewInt(0),
	BerlinBlock:                   big.NewInt(0),
	LondonBlock:                   big.NewInt(0),
	ArrowGlacierBlock:             big.NewInt(0),
	GrayGlacierBlock:              big.NewInt(0),
	MergeNetsplitBlock:            big.NewInt(0),
	BedrockBlock:                  big.NewInt(0),
	RegolithTime:                  &enabledFromBedrockBlock,
	TerminalTotalDifficulty:       big.NewInt(0),
	TerminalTotalDifficultyPassed: true,
	Optimism: &params.OptimismConfig{
		EIP1559Elasticity:  6,
		EIP1559Denominator: 50,
	},
}

var OPMainnetChainConfig = &params.ChainConfig{
	ChainID:                       big.NewInt(10),
	HomesteadBlock:                big.NewInt(0),
	DAOForkBlock:                  nil,
	DAOForkSupport:                false,
	EIP150Block:                   big.NewInt(0),
	EIP155Block:                   big.NewInt(0),
	EIP158Block:                   big.NewInt(0),
	ByzantiumBlock:                big.NewInt(0),
	ConstantinopleBlock:           big.NewInt(0),
	PetersburgBlock:               big.NewInt(0),
	IstanbulBlock:                 big.NewInt(0),
	MuirGlacierBlock:              big.NewInt(0),
	BerlinBlock:                   big.NewInt(3950000),
	LondonBlock:                   big.NewInt(105235063),
	ArrowGlacierBlock:             big.NewInt(105235063),
	GrayGlacierBlock:              big.NewInt(105235063),
	MergeNetsplitBlock:            big.NewInt(105235063),
	BedrockBlock:                  big.NewInt(105235063),
	RegolithTime:                  &enabledFromBedrockBlock,
	TerminalTotalDifficulty:       big.NewInt(0),
	TerminalTotalDifficultyPassed: true,
	Optimism: &params.OptimismConfig{
		EIP1559Elasticity:  6,
		EIP1559Denominator: 50,
	},
}

var L2ChainConfigsByName = map[string]*params.ChainConfig{
	"goerli":  OPGoerliChainConfig,
	"sepolia": OPSepoliaChainConfig,
	"mainnet": OPMainnetChainConfig,
}

func RollupConfigByChainID(chainID uint64) (*rollup.Config, error) {
	network := chaincfg.L2ChainIDToNetworkName[strconv.FormatUint(chainID, 10)]
	if network == "" {
		return nil, fmt.Errorf("unknown chain ID: %d", chainID)
	}
	config, ok := chaincfg.NetworksByName[network]
	if !ok {
		return nil, fmt.Errorf("unknown network %s for chain ID %d", network, chainID)
	}
	return &config, nil
}

func ChainConfigByChainID(chainID uint64) (*params.ChainConfig, error) {
	network := chaincfg.L2ChainIDToNetworkName[strconv.FormatUint(chainID, 10)]
	chainConfig, ok := L2ChainConfigsByName[network]
	if !ok {
		return nil, fmt.Errorf("unknown network %s for chain ID %d", network, chainID)
	}
	return chainConfig, nil
}
