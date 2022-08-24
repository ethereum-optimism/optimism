package genesis

import (
	"encoding/json"
	"math/big"
	"os"
	"path/filepath"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum-optimism/optimism/op-bindings/predeploys"
	"github.com/ethereum-optimism/optimism/op-chain-ops/hardhat"
	"github.com/ethereum-optimism/optimism/op-chain-ops/state"
)

// DeployConfig represents the deployment configuration for Optimism
type DeployConfig struct {
	L1StartingBlockTag               rpc.BlockNumberOrHash `json:"l1StartingBlockTag"`
	L1ChainID                        *big.Int              `json:"l1ChainID"`
	L2ChainID                        *big.Int              `json:"l2ChainID"`
	L2BlockTime                      uint                  `json:"l2BlockTime"`
	MaxSequencerDrift                uint                  `json:"maxSequencerDrift"`
	SequencerWindowSize              uint                  `json:"sequencerWindowSize"`
	ChannelTimeout                   uint                  `json:"channelTimeout"`
	P2PSequencerAddress              common.Address        `json:"p2pSequencerAddress"`
	OptimismL2FeeRecipient           common.Address        `json:"optimismL2FeeRecipient"`
	BatchInboxAddress                common.Address        `json:"batchInboxAddress"`
	BatchSenderAddress               common.Address        `json:"batchSenderAddress"`
	L2OutputOracleSubmissionInterval uint                  `json:"l2OutputOracleSubmissionInterval"`
	L2OutputOracleStartingTimestamp  int                   `json:"l2OutputOracleStartingTimestamp"`
	L2OutputOracleProposer           common.Address        `json:"l2OutputOracleProposer"`
	L2OutputOracleOwner              common.Address        `json:"l2OutputOracleOwner"`
	L1BlockTime                      uint64                `json:"l1BlockTime"`
	CliqueSignerAddress              common.Address        `json:"cliqueSignerAddress"`
	OptimismBaseFeeRecipient         common.Address        `json:"optimismBaseFeeRecipient"`
	OptimismL1FeeRecipient           common.Address        `json:"optimismL1FeeRecipient"`
	GasPriceOracleOwner              common.Address        `json:"gasPriceOracleOwner"`
	GasPriceOracleOverhead           uint                  `json:"gasPriceOracleOverhead"`
	GasPriceOracleScalar             uint                  `json:"gasPriceOracleScalar"`
	GasPriceOracleDecimals           uint                  `json:"gasPriceOracleDecimals"`
	L2CrossDomainMessengerOwner      common.Address        `json:"l2CrossDomainMessengerOwner"`
	L2GenesisBlockNonce              uint64                `json:"l2GenesisBlockNonce"`
	L2GenesisBlockExtraData          hexutil.Bytes         `json:"l2GenesisBlockExtraData"`
	L2GenesisBlockGasLimit           uint64                `json:"l2GenesisBlockGasLimit"`
	L2GenesisBlockDifficulty         *big.Int              `json:"l2GenesisBlockDifficulty"`
	L2GenesisBlockMixHash            common.Hash           `json:"l2GenesisBlockMixHash"`
	L2GenesisBlockCoinbase           common.Address        `json:"l2GenesisBlockCoinbase"`
	L2GenesisBlockNumber             uint64                `json:"l2GenesisBlockNumber"`
	L2GenesisBlockGasUsed            uint64                `json:"l2GenesisBlockGasUsed"`
	L2GenesisBlockParentHash         common.Hash           `json:"l2GenesisBlockParentHash"`
	L2GenesisBlockBaseFeePerGas      *big.Int              `json:"l2GenesisBlockBaseFeePerGas"`
	L1GenesisBlockTimestamp          uint64                `json:"l1GenesisBlockTimestamp"`
	L1GenesisBlockNonce              uint64                `json:"l1GenesisBlockNonce"`
	L1GenesisBlockGasLimit           uint64                `json:"l1GenesisBlockGasLimit"`
	L1GenesisBlockDifficulty         *big.Int              `json:"l1GenesisBlockDifficulty"`
	L1GenesisBlockMixHash            common.Hash           `json:"l1GenesisBlockMixHash"`
	L1GenesisBlockCoinbase           common.Address        `json:"l1GenesisBlockCoinbase"`
	L1GenesisBlockNumber             uint64                `json:"l1GenesisBlockNumber"`
	L1GenesisBlockGasUsed            uint64                `json:"l1GenesisBlockGasUsed"`
	L1GenesisBlockParentHash         common.Hash           `json:"l1GenesisBlockParentHash"`
	L1GenesisBlockBaseFeePerGas      *big.Int              `json:"l1GenesisBlockBaseFeePerGas"`
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

// StorageConfig represents the storage configuration for the L2 predeploy
// contracts.
type StorageConfig map[string]state.StorageValues

// NewStorageConfig will create a StorageConfig given an instance of a
// Hardhat and a DeployConfig.
func NewStorageConfig(hh *hardhat.Hardhat, config *DeployConfig, chain ethereum.ChainReader) (StorageConfig, error) {
	storage := make(StorageConfig)

	proxyL1StandardBridge, err := hh.GetDeployment("L1StandardBridgeProxy")
	if err != nil {
		return storage, err
	}
	proxyL1CrossDomainMessenger, err := hh.GetDeployment("L1CrossDomainMessengerProxy")
	if err != nil {
		return storage, err
	}

	block, err := getBlockFromTag(chain, config.L1StartingBlockTag)
	if err != nil {
		return storage, err
	}

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
		"otherMessenger":   proxyL1CrossDomainMessenger.Address,
		"blockedSystemAddresses": map[any]any{
			predeploys.L2CrossDomainMessenger: true,
			predeploys.L2ToL1MessagePasser:    true,
		},
	}
	storage["GasPriceOracle"] = state.StorageValues{
		"_owner":   config.GasPriceOracleOwner,
		"overhead": config.GasPriceOracleOverhead,
		"scalar":   config.GasPriceOracleScalar,
		"decimals": config.GasPriceOracleDecimals,
	}
	storage["L2StandardBridge"] = state.StorageValues{
		"_initialized":  true,
		"_initializing": false,
		"messenger":     predeploys.L2CrossDomainMessenger,
		"otherBridge":   proxyL1StandardBridge.Address,
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
