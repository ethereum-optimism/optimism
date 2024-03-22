package genesis

import (
	"errors"
	"fmt"
	"math/big"

	"github.com/bobanetwork/boba/boba-bindings/predeploys"
	"github.com/ledgerwatch/erigon-lib/chain"
	"github.com/ledgerwatch/erigon-lib/common"
	"github.com/ledgerwatch/erigon/core/types"
	"github.com/ledgerwatch/erigon/params"
)

var (
	BedrockTransitionBlockExtraData = []byte("BEDROCK")
)

// defaultL2GasLimit represents the default gas limit for an L2 block.
const defaultL2GasLimit = 30_000_000

// NewL2Genesis will create a new L2 genesis
func NewL2Genesis(config *DeployConfig, header *types.Header) (*types.Genesis, error) {
	if config.L2ChainID == 0 {
		return nil, errors.New("must define L2 ChainID")
	}

	eip1559Denom := config.EIP1559Denominator
	if eip1559Denom == 0 {
		eip1559Denom = 50
	}
	eip1559DenomCanyon := config.EIP1559DenominatorCanyon
	if eip1559DenomCanyon == 0 {
		eip1559DenomCanyon = 250
	}
	eip1559Elasticity := config.EIP1559Elasticity
	if eip1559Elasticity == 0 {
		eip1559Elasticity = 10
	}

	var regolithTime *big.Int
	regolithTimeUnit64 := config.RegolithTime(header.Time)
	if regolithTimeUnit64 != nil {
		regolithTime = new(big.Int).SetUint64(*regolithTimeUnit64)
	}
	var canyonTime *big.Int
	canyonTimeUint64 := config.CanyonTime(header.Time)
	if canyonTimeUint64 != nil {
		canyonTime = new(big.Int).SetUint64(*canyonTimeUint64)
	}
	var ecotoneTime *big.Int
	ecotoneTimeUint64 := config.EcotoneTime(header.Time)
	if ecotoneTimeUint64 != nil {
		ecotoneTime = new(big.Int).SetUint64(*ecotoneTimeUint64)
	}

	optimismChainConfig := chain.Config{
		ChainID:                       new(big.Int).SetUint64(config.L2ChainID),
		HomesteadBlock:                big.NewInt(0),
		DAOForkBlock:                  nil,
		TangerineWhistleBlock:         big.NewInt(0),
		SpuriousDragonBlock:           big.NewInt(0),
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
		TerminalTotalDifficulty:       big.NewInt(0),
		TerminalTotalDifficultyPassed: true,
		BedrockBlock:                  new(big.Int).SetUint64(uint64(config.L2GenesisBlockNumber)),
		RegolithTime:                  regolithTime,
		CanyonTime:                    canyonTime,
		ShanghaiTime:                  canyonTime,
		CancunTime:                    ecotoneTime,
		EcotoneTime:                   ecotoneTime,
		Optimism: &chain.OptimismConfig{
			EIP1559Denominator:       eip1559Denom,
			EIP1559Elasticity:        eip1559Elasticity,
			EIP1559DenominatorCanyon: eip1559DenomCanyon,
		},
	}

	gasLimit := config.L2GenesisBlockGasLimit
	if gasLimit == 0 {
		gasLimit = defaultL2GasLimit
	}
	baseFee := config.L2GenesisBlockBaseFeePerGas
	if baseFee == nil {
		baseFee = newHexBig(params.InitialBaseFee)
	}
	difficulty := config.L2GenesisBlockDifficulty
	if difficulty == nil {
		difficulty = newHexBig(0)
	}

	// Ensure that the extradata is valid
	if size := len(BedrockTransitionBlockExtraData); size > 32 {
		return nil, fmt.Errorf("transition block extradata too long: %d", size)
	}

	return &types.Genesis{
		Config:     &optimismChainConfig,
		Nonce:      uint64(config.L2GenesisBlockNonce),
		Timestamp:  header.Time,
		ExtraData:  BedrockTransitionBlockExtraData,
		GasLimit:   uint64(gasLimit),
		Difficulty: difficulty.ToInt(),
		Mixhash:    config.L2GenesisBlockMixHash,
		Coinbase:   predeploys.SequencerFeeVaultAddr,
		Number:     uint64(config.L2GenesisBlockNumber),
		GasUsed:    uint64(config.L2GenesisBlockGasUsed),
		ParentHash: config.L2GenesisBlockParentHash,
		BaseFee:    baseFee.ToInt(),
		Alloc:      map[common.Address]types.GenesisAccount{},
	}, nil
}
