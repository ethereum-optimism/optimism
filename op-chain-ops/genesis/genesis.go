package genesis

import (
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/predeploys"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
)

// defaultGasLimit represents the default gas limit for a genesis block.
const defaultGasLimit = 30_000_000

// BedrockTransitionBlockExtraData represents the default extra data for the bedrock transition block.
var BedrockTransitionBlockExtraData = []byte("BEDROCK")

// NewL2Genesis will create a new L2 genesis
func NewL2Genesis(config *DeployConfig, block *types.Block) (*core.Genesis, error) {
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

	optimismChainConfig := params.ChainConfig{
		ChainID:                       new(big.Int).SetUint64(config.L2ChainID),
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
		TerminalTotalDifficulty:       big.NewInt(0),
		TerminalTotalDifficultyPassed: true,
		BedrockBlock:                  new(big.Int).SetUint64(uint64(config.L2GenesisBlockNumber)),
		RegolithTime:                  config.RegolithTime(block.Time()),
		CanyonTime:                    config.CanyonTime(block.Time()),
		ShanghaiTime:                  config.CanyonTime(block.Time()),
		CancunTime:                    config.EcotoneTime(block.Time()),
		EcotoneTime:                   config.EcotoneTime(block.Time()),
		FjordTime:                     config.FjordTime(block.Time()),
		InteropTime:                   config.InteropTime(block.Time()),
		Optimism: &params.OptimismConfig{
			EIP1559Denominator:       eip1559Denom,
			EIP1559Elasticity:        eip1559Elasticity,
			EIP1559DenominatorCanyon: eip1559DenomCanyon,
		},
	}

	gasLimit := config.L2GenesisBlockGasLimit
	if gasLimit == 0 {
		gasLimit = defaultGasLimit
	}
	baseFee := config.L2GenesisBlockBaseFeePerGas
	if baseFee == nil {
		baseFee = newHexBig(params.InitialBaseFee)
	}
	difficulty := config.L2GenesisBlockDifficulty
	if difficulty == nil {
		difficulty = newHexBig(0)
	}

	extraData := config.L2GenesisBlockExtraData
	if extraData == nil {
		// L2GenesisBlockExtraData is optional, so use a default value when nil
		extraData = BedrockTransitionBlockExtraData
	}
	// Ensure that the extradata is valid
	if size := len(extraData); size > 32 {
		return nil, fmt.Errorf("transition block extradata too long: %d", size)
	}

	genesis := &core.Genesis{
		Config:     &optimismChainConfig,
		Nonce:      uint64(config.L2GenesisBlockNonce),
		Timestamp:  block.Time(),
		ExtraData:  extraData,
		GasLimit:   uint64(gasLimit),
		Difficulty: difficulty.ToInt(),
		Mixhash:    config.L2GenesisBlockMixHash,
		Coinbase:   predeploys.SequencerFeeVaultAddr,
		Number:     uint64(config.L2GenesisBlockNumber),
		GasUsed:    uint64(config.L2GenesisBlockGasUsed),
		ParentHash: config.L2GenesisBlockParentHash,
		BaseFee:    baseFee.ToInt(),
		Alloc:      map[common.Address]types.Account{},
	}

	if optimismChainConfig.IsEcotone(genesis.Timestamp) {
		genesis.BlobGasUsed = u64ptr(0)
		genesis.ExcessBlobGas = u64ptr(0)
	}

	return genesis, nil
}

// NewL1Genesis will create a new L1 genesis config
func NewL1Genesis(config *DeployConfig) (*core.Genesis, error) {
	if config.L1ChainID == 0 {
		return nil, errors.New("must define L1 ChainID")
	}

	chainConfig := params.ChainConfig{
		ChainID:             uint642Big(config.L1ChainID),
		HomesteadBlock:      big.NewInt(0),
		DAOForkBlock:        nil,
		DAOForkSupport:      false,
		EIP150Block:         big.NewInt(0),
		EIP155Block:         big.NewInt(0),
		EIP158Block:         big.NewInt(0),
		ByzantiumBlock:      big.NewInt(0),
		ConstantinopleBlock: big.NewInt(0),
		PetersburgBlock:     big.NewInt(0),
		IstanbulBlock:       big.NewInt(0),
		MuirGlacierBlock:    big.NewInt(0),
		BerlinBlock:         big.NewInt(0),
		LondonBlock:         big.NewInt(0),
		ArrowGlacierBlock:   big.NewInt(0),
		GrayGlacierBlock:    big.NewInt(0),
		ShanghaiTime:        nil,
		CancunTime:          nil,
	}

	extraData := make([]byte, 0)
	if config.L1UseClique {
		// warning: clique has an overly strict block header timestamp check against the system wallclock,
		// causing blocks to get scheduled as "future block" and not get mined instantly when produced.
		chainConfig.Clique = &params.CliqueConfig{
			Period: config.L1BlockTime,
			Epoch:  30000,
		}
		extraData = append(append(make([]byte, 32), config.CliqueSignerAddress[:]...), make([]byte, crypto.SignatureLength)...)
	} else {
		chainConfig.MergeNetsplitBlock = big.NewInt(0)
		chainConfig.TerminalTotalDifficulty = big.NewInt(0)
		chainConfig.TerminalTotalDifficultyPassed = true
		chainConfig.ShanghaiTime = u64ptr(0)
		chainConfig.CancunTime = u64ptr(0)
	}

	gasLimit := config.L1GenesisBlockGasLimit
	if gasLimit == 0 {
		gasLimit = defaultGasLimit
	}
	baseFee := config.L1GenesisBlockBaseFeePerGas
	if baseFee == nil {
		baseFee = newHexBig(params.InitialBaseFee)
	}
	difficulty := config.L1GenesisBlockDifficulty
	if difficulty == nil {
		difficulty = newHexBig(1)
	}
	timestamp := config.L1GenesisBlockTimestamp
	if timestamp == 0 {
		timestamp = hexutil.Uint64(time.Now().Unix())
	}
	if !config.L1UseClique && config.L1CancunTimeOffset != nil {
		cancunTime := uint64(timestamp) + uint64(*config.L1CancunTimeOffset)
		chainConfig.CancunTime = &cancunTime
	}

	return &core.Genesis{
		Config:     &chainConfig,
		Nonce:      uint64(config.L1GenesisBlockNonce),
		Timestamp:  uint64(timestamp),
		ExtraData:  extraData,
		GasLimit:   uint64(gasLimit),
		Difficulty: difficulty.ToInt(),
		Mixhash:    config.L1GenesisBlockMixHash,
		Coinbase:   config.L1GenesisBlockCoinbase,
		Number:     uint64(config.L1GenesisBlockNumber),
		GasUsed:    uint64(config.L1GenesisBlockGasUsed),
		ParentHash: config.L1GenesisBlockParentHash,
		BaseFee:    baseFee.ToInt(),
		Alloc:      map[common.Address]types.Account{},
	}, nil
}

func u64ptr(n uint64) *uint64 {
	return &n
}
