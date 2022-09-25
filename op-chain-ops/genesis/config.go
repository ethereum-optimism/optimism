package genesis

import (
	"encoding/json"
	"math/big"
	"os"
	"path/filepath"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum-optimism/optimism/op-bindings/predeploys"
	"github.com/ethereum-optimism/optimism/op-chain-ops/immutables"
	"github.com/ethereum-optimism/optimism/op-chain-ops/state"
	"github.com/ethereum/go-ethereum/common"
)

// DeployConfig represents the deployment configuration for Optimism
type DeployConfig struct {
	L1StartingBlockTag *rpc.BlockNumberOrHash `json:"l1StartingBlockTag"`
	L1ChainID          uint64                 `json:"l1ChainID"`
	L2ChainID          uint64                 `json:"l2ChainID"`
	L2BlockTime        uint64                 `json:"l2BlockTime"`

	FinalizationPeriodSeconds uint64         `json:"finalizationPeriodSeconds"`
	MaxSequencerDrift         uint64         `json:"maxSequencerDrift"`
	SequencerWindowSize       uint64         `json:"sequencerWindowSize"`
	ChannelTimeout            uint64         `json:"channelTimeout"`
	P2PSequencerAddress       common.Address `json:"p2pSequencerAddress"`
	OptimismL2FeeRecipient    common.Address `json:"optimismL2FeeRecipient"`
	BatchInboxAddress         common.Address `json:"batchInboxAddress"`
	BatchSenderAddress        common.Address `json:"batchSenderAddress"`

	L2OutputOracleSubmissionInterval uint64         `json:"l2OutputOracleSubmissionInterval"`
	L2OutputOracleStartingTimestamp  int            `json:"l2OutputOracleStartingTimestamp"`
	L2OutputOracleProposer           common.Address `json:"l2OutputOracleProposer"`
	L2OutputOracleOwner              common.Address `json:"l2OutputOracleOwner"`
	L2OutputOracleGenesisL2Output    common.Hash    `json:"l2OutputOracleGenesisL2Output"`

	L1BlockTime                 uint64         `json:"l1BlockTime"`
	L1GenesisBlockTimestamp     hexutil.Uint64 `json:"l1GenesisBlockTimestamp"`
	L1GenesisBlockNonce         hexutil.Uint64 `json:"l1GenesisBlockNonce"`
	CliqueSignerAddress         common.Address `json:"cliqueSignerAddress"`
	L1GenesisBlockGasLimit      hexutil.Uint64 `json:"l1GenesisBlockGasLimit"`
	L1GenesisBlockDifficulty    *hexutil.Big   `json:"l1GenesisBlockDifficulty"`
	L1GenesisBlockMixHash       common.Hash    `json:"l1GenesisBlockMixHash"`
	L1GenesisBlockCoinbase      common.Address `json:"l1GenesisBlockCoinbase"`
	L1GenesisBlockNumber        hexutil.Uint64 `json:"l1GenesisBlockNumber"`
	L1GenesisBlockGasUsed       hexutil.Uint64 `json:"l1GenesisBlockGasUsed"`
	L1GenesisBlockParentHash    common.Hash    `json:"l1GenesisBlockParentHash"`
	L1GenesisBlockBaseFeePerGas *hexutil.Big   `json:"l1GenesisBlockBaseFeePerGas"`

	L2GenesisBlockNonce         hexutil.Uint64 `json:"l2GenesisBlockNonce"`
	L2GenesisBlockExtraData     hexutil.Bytes  `json:"l2GenesisBlockExtraData"`
	L2GenesisBlockGasLimit      hexutil.Uint64 `json:"l2GenesisBlockGasLimit"`
	L2GenesisBlockDifficulty    *hexutil.Big   `json:"l2GenesisBlockDifficulty"`
	L2GenesisBlockMixHash       common.Hash    `json:"l2GenesisBlockMixHash"`
	L2GenesisBlockCoinbase      common.Address `json:"l2GenesisBlockCoinbase"`
	L2GenesisBlockNumber        hexutil.Uint64 `json:"l2GenesisBlockNumber"`
	L2GenesisBlockGasUsed       hexutil.Uint64 `json:"l2GenesisBlockGasUsed"`
	L2GenesisBlockParentHash    common.Hash    `json:"l2GenesisBlockParentHash"`
	L2GenesisBlockBaseFeePerGas *hexutil.Big   `json:"l2GenesisBlockBaseFeePerGas"`

	L2CrossDomainMessengerOwner common.Address `json:"l2CrossDomainMessengerOwner"`
	OptimismBaseFeeRecipient    common.Address `json:"optimismBaseFeeRecipient"`
	OptimismL1FeeRecipient      common.Address `json:"optimismL1FeeRecipient"`
	GasPriceOracleOwner         common.Address `json:"gasPriceOracleOwner"`
	GasPriceOracleOverhead      uint           `json:"gasPriceOracleOverhead"`
	GasPriceOracleScalar        uint           `json:"gasPriceOracleScalar"`
	GasPriceOracleDecimals      uint           `json:"gasPriceOracleDecimals"`

	DeploymentWaitConfirmations int `json:"deploymentWaitConfirmations"`

	EIP1559Elasticity  uint64 `json:"eip1559Elasticity"`
	EIP1559Denominator uint64 `json:"eip1559Denominator"`

	FundDevAccounts bool `json:"fundDevAccounts"`
}

// NewDeployConfig reads a config file given a path on the filesystem.
func NewDeployConfig(path string) (*DeployConfig, error) {
	file, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config DeployConfig
	if err := json.Unmarshal(file, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

// NewDeployConfigWithNetwork takes a path to a deploy config directory
// and the network name. The config file in the deploy config directory
// must match the network name and be a JSON file.
func NewDeployConfigWithNetwork(network, path string) (*DeployConfig, error) {
	deployConfig := filepath.Join(path, network+".json")
	return NewDeployConfig(deployConfig)
}

// NewL2ImmutableConfig will create an ImmutableConfig given an instance of a
// Hardhat and a DeployConfig.
func NewL2ImmutableConfig(config *DeployConfig, block *types.Block, proxyL1StandardBridge, proxyL1CrossDomainMessenger, proxyL1ERC721Bridge common.Address) (immutables.ImmutableConfig, error) {
	immutable := make(immutables.ImmutableConfig)

	immutable["L2StandardBridge"] = immutables.ImmutableValues{
		"otherBridge": proxyL1StandardBridge,
	}
	immutable["L2CrossDomainMessenger"] = immutables.ImmutableValues{
		"otherMessenger": proxyL1CrossDomainMessenger,
	}
	immutable["L2ERC721Bridge"] = immutables.ImmutableValues{
		"messenger":   predeploys.L2CrossDomainMessengerAddr,
		"otherBridge": proxyL1ERC721Bridge,
	}
	immutable["OptimismMintableERC721Factory"] = immutables.ImmutableValues{
		"bridge":        predeploys.L2ERC721BridgeAddr,
		"remoteChainId": new(big.Int).SetUint64(config.L1ChainID),
	}

	return immutable, nil
}

// NewL2StorageConfig will create a StorageConfig given an instance of a
// Hardhat and a DeployConfig.
func NewL2StorageConfig(config *DeployConfig, block *types.Block, proxyL1StandardBridge common.Address, proxyL1CrossDomainMessenger common.Address) (state.StorageConfig, error) {
	storage := make(state.StorageConfig)

	storage["L2ToL1MessagePasser"] = state.StorageValues{
		"nonce": 0,
	}
	storage["L2CrossDomainMessenger"] = state.StorageValues{
		"_initialized": 1,
		"_owner":       config.L2CrossDomainMessengerOwner,
		// re-entrency lock
		"_status":          1,
		"_initializing":    false,
		"_paused":          false,
		"xDomainMsgSender": "0x000000000000000000000000000000000000dEaD",
		"msgNonce":         0,
	}
	storage["GasPriceOracle"] = state.StorageValues{
		"_owner":   config.GasPriceOracleOwner,
		"overhead": config.GasPriceOracleOverhead,
		"scalar":   config.GasPriceOracleScalar,
		"decimals": config.GasPriceOracleDecimals,
	}
	storage["SequencerFeeVault"] = state.StorageValues{
		"l1FeeWallet": config.OptimismL1FeeRecipient,
	}
	storage["L1Block"] = state.StorageValues{
		"number":         block.Number(),
		"timestamp":      block.Time(),
		"basefee":        block.BaseFee(),
		"hash":           block.Hash(),
		"sequenceNumber": 0,
	}
	storage["LegacyERC20ETH"] = state.StorageValues{
		"bridge":      predeploys.L2StandardBridge,
		"remoteToken": common.Address{},
		"_name":       "Ether",
		"_symbol":     "ETH",
	}
	storage["WETH9"] = state.StorageValues{
		"name":     "Wrapped Ether",
		"symbol":   "WETH",
		"decimals": 18,
	}
	storage["GovernanceToken"] = state.StorageValues{
		"_name":   "Optimism",
		"_symbol": "OP",
		// TODO: this should be set to the MintManager
		"_owner": common.Address{},
	}
	return storage, nil
}
