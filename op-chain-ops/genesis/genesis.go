package genesis

import (
	"errors"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
)

// NewL2Genesis will create a new L2 genesis
func NewL2Genesis(config *DeployConfig, chain ethereum.ChainReader) (*core.Genesis, error) {
	if config.L2ChainID == nil {
		return nil, errors.New("must define L2 ChainID")
	}

	optimismChainConfig := params.ChainConfig{
		ChainID:                       config.L2ChainID,
		HomesteadBlock:                big.NewInt(0),
		DAOForkBlock:                  nil,
		DAOForkSupport:                false,
		EIP150Block:                   big.NewInt(0),
		EIP150Hash:                    common.Hash{},
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
		ShanghaiBlock:                 nil,
		CancunBlock:                   nil,
		TerminalTotalDifficulty:       big.NewInt(0),
		TerminalTotalDifficultyPassed: true,
		Optimism: &params.OptimismConfig{
			BaseFeeRecipient: config.OptimismBaseFeeRecipient,
			L1FeeRecipient:   config.OptimismL2FeeRecipient,
		},
	}

	extraData := config.L2GenesisBlockExtraData
	if len(extraData) == 0 {
		extraData = common.Hash{}.Bytes()
	}
	gasLimit := config.L2GenesisBlockGasLimit
	if gasLimit == 0 {
		gasLimit = uint64(15_000_000)
	}
	baseFee := config.L2GenesisBlockBaseFeePerGas
	if baseFee == nil {
		baseFee = big.NewInt(params.InitialBaseFee)
	}
	difficulty := config.L2GenesisBlockDifficulty
	if difficulty == nil {
		difficulty = big.NewInt(1)
	}

	block, err := getBlockFromTag(chain, config.L1StartingBlockTag)
	if err != nil {
		return nil, err
	}

	return &core.Genesis{
		Config:     &optimismChainConfig,
		Nonce:      config.L2GenesisBlockNonce,
		Timestamp:  block.Time(),
		ExtraData:  extraData,
		GasLimit:   gasLimit,
		Difficulty: difficulty,
		Mixhash:    config.L2GenesisBlockMixHash,
		Coinbase:   config.L2GenesisBlockCoinbase,
		Number:     config.L2GenesisBlockNumber,
		GasUsed:    config.L2GenesisBlockGasUsed,
		ParentHash: config.L2GenesisBlockParentHash,
		BaseFee:    baseFee,
		Alloc:      map[common.Address]core.GenesisAccount{},
	}, nil
}

// NewL1Genesis will create a new L1 genesis config
func NewL1Genesis(config *DeployConfig) (*core.Genesis, error) {
	if config.L1ChainID == nil {
		return nil, errors.New("must define L1 ChainID")
	}

	chainConfig := *params.AllCliqueProtocolChanges
	chainConfig.Clique = &params.CliqueConfig{
		Period: config.L1BlockTime,
		Epoch:  30000,
	}
	chainConfig.ChainID = config.L1ChainID

	gasLimit := config.L1GenesisBlockGasLimit
	if gasLimit == 0 {
		gasLimit = uint64(15_000_000)
	}
	baseFee := config.L1GenesisBlockBaseFeePerGas
	if baseFee == nil {
		baseFee = big.NewInt(params.InitialBaseFee)
	}
	difficulty := config.L1GenesisBlockDifficulty
	if difficulty == nil {
		difficulty = big.NewInt(1)
	}
	timestamp := config.L1GenesisBlockTimestamp
	if timestamp == 0 {
		timestamp = uint64(time.Now().Unix())
	}

	extraData := append(append(make([]byte, 32), config.CliqueSignerAddress[:]...), make([]byte, crypto.SignatureLength)...)

	return &core.Genesis{
		Config:     &chainConfig,
		Nonce:      config.L1GenesisBlockNonce,
		Timestamp:  timestamp,
		ExtraData:  extraData,
		GasLimit:   gasLimit,
		Difficulty: difficulty,
		Mixhash:    config.L1GenesisBlockMixHash,
		Coinbase:   config.L1GenesisBlockCoinbase,
		Number:     config.L1GenesisBlockNumber,
		GasUsed:    config.L1GenesisBlockGasUsed,
		ParentHash: config.L1GenesisBlockParentHash,
		BaseFee:    baseFee,
		Alloc:      map[common.Address]core.GenesisAccount{},
	}, nil
}
