package genesis

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"os"
	"path/filepath"

	"github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum-optimism/optimism/op-bindings/predeploys"
	"github.com/ethereum-optimism/optimism/op-chain-ops/immutables"
	"github.com/ethereum-optimism/optimism/op-chain-ops/state"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
)

// DeployConfig represents the deployment configuration for Optimism
type DeployConfig struct {
	JSONDeployConfig

	L1StartingBlockTag *rpc.BlockNumberOrHash `json:"-"`
}

func (d *DeployConfig) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.JSONDeployConfig)
}

func (d *DeployConfig) UnmarshalJSON(b []byte) error {
	if err := json.Unmarshal(b, &d.JSONDeployConfig); err != nil {
		return nil
	}

	return json.Unmarshal(d.RawL1StartingBlockTag, &d.L1StartingBlockTag)
}

func (d *DeployConfig) SetL1StartingBlockTag(tag *rpc.BlockNumberOrHash) {
	d.L1StartingBlockTag = tag

	var val any
	if n, ok := tag.Number(); ok {
		val = n
	} else if hash, ok := tag.Hash(); ok {
		val = hash
	} else {
		panic("invalid tag")
	}

	d.RawL1StartingBlockTag, _ = json.Marshal(val)
}

type JSONDeployConfig struct {
	RawL1StartingBlockTag json.RawMessage `json:"l1StartingBlockTag"`

	L1ChainID   uint64 `json:"l1ChainID"`
	L2ChainID   uint64 `json:"l2ChainID"`
	L2BlockTime uint64 `json:"l2BlockTime"`

	FinalizationPeriodSeconds uint64         `json:"finalizationPeriodSeconds,omitempty"`
	MaxSequencerDrift         uint64         `json:"maxSequencerDrift,omitempty"`
	SequencerWindowSize       uint64         `json:"sequencerWindowSize,omitempty"`
	ChannelTimeout            uint64         `json:"channelTimeout,omitempty"`
	P2PSequencerAddress       common.Address `json:"p2pSequencerAddress,omitempty"`
	BatchInboxAddress         common.Address `json:"batchInboxAddress,omitempty"`
	BatchSenderAddress        common.Address `json:"batchSenderAddress,omitempty"`

	L2OutputOracleSubmissionInterval uint64         `json:"l2OutputOracleSubmissionInterval,omitempty"`
	L2OutputOracleStartingTimestamp  int            `json:"l2OutputOracleStartingTimestamp,omitempty"`
	L2OutputOracleProposer           common.Address `json:"l2OutputOracleProposer,omitempty"`
	L2OutputOracleOwner              common.Address `json:"l2OutputOracleOwner,omitempty"`
	L2OutputOracleGenesisL2Output    common.Hash    `json:"l2OutputOracleGenesisL2Output,omitempty"`

	SystemConfigOwner common.Address `json:"systemConfigOwner,omitempty"`

	L1BlockTime                 uint64         `json:"l1BlockTime,omitempty"`
	L1GenesisBlockTimestamp     hexutil.Uint64 `json:"l1GenesisBlockTimestamp,omitempty"`
	L1GenesisBlockNonce         hexutil.Uint64 `json:"l1GenesisBlockNonce,omitempty"`
	CliqueSignerAddress         common.Address `json:"cliqueSignerAddress,omitempty"` // proof of stake genesis if left zeroed.
	L1GenesisBlockGasLimit      hexutil.Uint64 `json:"l1GenesisBlockGasLimit,omitempty"`
	L1GenesisBlockDifficulty    *hexutil.Big   `json:"l1GenesisBlockDifficulty,omitempty"`
	L1GenesisBlockMixHash       common.Hash    `json:"l1GenesisBlockMixHash,omitempty"`
	L1GenesisBlockCoinbase      common.Address `json:"l1GenesisBlockCoinbase,omitempty"`
	L1GenesisBlockNumber        hexutil.Uint64 `json:"l1GenesisBlockNumber,omitempty"`
	L1GenesisBlockGasUsed       hexutil.Uint64 `json:"l1GenesisBlockGasUsed,omitempty"`
	L1GenesisBlockParentHash    common.Hash    `json:"l1GenesisBlockParentHash,omitempty"`
	L1GenesisBlockBaseFeePerGas *hexutil.Big   `json:"l1GenesisBlockBaseFeePerGas,omitempty"`

	L2GenesisBlockNonce         hexutil.Uint64 `json:"l2GenesisBlockNonce,omitempty"`
	L2GenesisBlockExtraData     hexutil.Bytes  `json:"l2GenesisBlockExtraData,omitempty"`
	L2GenesisBlockGasLimit      hexutil.Uint64 `json:"l2GenesisBlockGasLimit,omitempty"`
	L2GenesisBlockDifficulty    *hexutil.Big   `json:"l2GenesisBlockDifficulty,omitempty"`
	L2GenesisBlockMixHash       common.Hash    `json:"l2GenesisBlockMixHash,omitempty"`
	L2GenesisBlockCoinbase      common.Address `json:"l2GenesisBlockCoinbase,omitempty"`
	L2GenesisBlockNumber        hexutil.Uint64 `json:"l2GenesisBlockNumber,omitempty"`
	L2GenesisBlockGasUsed       hexutil.Uint64 `json:"l2GenesisBlockGasUsed,omitempty"`
	L2GenesisBlockParentHash    common.Hash    `json:"l2GenesisBlockParentHash,omitempty"`
	L2GenesisBlockBaseFeePerGas *hexutil.Big   `json:"l2GenesisBlockBaseFeePerGas,omitempty"`

	ProxyAdminOwner             common.Address `json:"proxyAdminOwner,omitempty"`
	L2CrossDomainMessengerOwner common.Address `json:"l2CrossDomainMessengerOwner,omitempty"`
	OptimismBaseFeeRecipient    common.Address `json:"optimismBaseFeeRecipient,omitempty"`
	OptimismL1FeeRecipient      common.Address `json:"optimismL1FeeRecipient,omitempty"`

	GasPriceOracleOwner    common.Address `json:"gasPriceOracleOwner,omitempty"`
	GasPriceOracleOverhead uint64         `json:"gasPriceOracleOverhead,omitempty"`
	GasPriceOracleScalar   uint64         `json:"gasPriceOracleScalar,omitempty"`

	DeploymentWaitConfirmations int `json:"deploymentWaitConfirmations,omitempty"`

	EIP1559Elasticity  uint64 `json:"eip1559Elasticity,omitempty"`
	EIP1559Denominator uint64 `json:"eip1559Denominator,omitempty"`

	FundDevAccounts bool `json:"fundDevAccounts"`
}

// NewDeployConfig reads a config file given a path on the filesystem.
func NewDeployConfig(path string) (*DeployConfig, error) {
	file, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("deploy config at %s not found: %w", path, err)
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
func NewL2ImmutableConfig(config *DeployConfig, block *types.Block, l2Addrs *L2Addresses) (immutables.ImmutableConfig, error) {
	immutable := make(immutables.ImmutableConfig)

	if l2Addrs == nil {
		return immutable, errors.New("must pass L1 contract addresses")
	}

	if l2Addrs.L1ERC721BridgeProxy == (common.Address{}) {
		return immutable, errors.New("L1ERC721BridgeProxy cannot be address(0)")
	}

	immutable["L2StandardBridge"] = immutables.ImmutableValues{
		"otherBridge": l2Addrs.L1StandardBridgeProxy,
	}
	immutable["L2CrossDomainMessenger"] = immutables.ImmutableValues{
		"otherMessenger": l2Addrs.L1CrossDomainMessengerProxy,
	}
	immutable["L2ERC721Bridge"] = immutables.ImmutableValues{
		"messenger":   predeploys.L2CrossDomainMessengerAddr,
		"otherBridge": l2Addrs.L1ERC721BridgeProxy,
	}
	immutable["OptimismMintableERC721Factory"] = immutables.ImmutableValues{
		"bridge":        predeploys.L2ERC721BridgeAddr,
		"remoteChainId": new(big.Int).SetUint64(config.L1ChainID),
	}
	immutable["SequencerFeeVault"] = immutables.ImmutableValues{
		"recipient": l2Addrs.SequencerFeeVaultRecipient,
	}
	immutable["L1FeeVault"] = immutables.ImmutableValues{
		"recipient": l2Addrs.L1FeeVaultRecipient,
	}
	immutable["BaseFeeVault"] = immutables.ImmutableValues{
		"recipient": l2Addrs.BaseFeeVaultRecipient,
	}

	return immutable, nil
}

// NewL2StorageConfig will create a StorageConfig given an instance of a
// Hardhat and a DeployConfig.
func NewL2StorageConfig(config *DeployConfig, block *types.Block, l2Addrs *L2Addresses) (state.StorageConfig, error) {
	storage := make(state.StorageConfig)

	if block.Number() == nil {
		return storage, errors.New("block number not set")
	}
	if block.BaseFee() == nil {
		return storage, errors.New("block base fee not set")
	}
	if l2Addrs == nil {
		return storage, errors.New("must pass L1 address info")
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
	}
	storage["GasPriceOracle"] = state.StorageValues{
		// TODO: remove this in the future
		"_owner": config.GasPriceOracleOwner,
	}
	storage["L1Block"] = state.StorageValues{
		"number":         block.Number(),
		"timestamp":      block.Time(),
		"basefee":        block.BaseFee(),
		"hash":           block.Hash(),
		"sequenceNumber": 0,
		"batcherHash":    config.BatchSenderAddress.Hash(),
		"l1FeeOverhead":  config.GasPriceOracleOverhead,
		"l1FeeScalar":    config.GasPriceOracleScalar,
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
	storage["ProxyAdmin"] = state.StorageValues{
		"owner": l2Addrs.ProxyAdminOwner,
	}
	return storage, nil
}
