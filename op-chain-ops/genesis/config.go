package genesis

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"os"
	"path/filepath"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum-optimism/optimism/op-bindings/hardhat"
	"github.com/ethereum-optimism/optimism/op-bindings/predeploys"
	"github.com/ethereum-optimism/optimism/op-chain-ops/immutables"
	"github.com/ethereum-optimism/optimism/op-chain-ops/state"
	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
)

var (
	ErrInvalidDeployConfig     = errors.New("invalid deploy config")
	ErrInvalidImmutablesConfig = errors.New("invalid immutables config")
)

// DeployConfig represents the deployment configuration for Optimism
type DeployConfig struct {
	L1StartingBlockTag *MarshalableRPCBlockNumberOrHash `json:"l1StartingBlockTag"`
	L1ChainID          uint64                           `json:"l1ChainID"`
	L2ChainID          uint64                           `json:"l2ChainID"`
	L2BlockTime        uint64                           `json:"l2BlockTime"`

	FinalizationPeriodSeconds uint64         `json:"finalizationPeriodSeconds"`
	MaxSequencerDrift         uint64         `json:"maxSequencerDrift"`
	SequencerWindowSize       uint64         `json:"sequencerWindowSize"`
	ChannelTimeout            uint64         `json:"channelTimeout"`
	P2PSequencerAddress       common.Address `json:"p2pSequencerAddress"`
	BatchInboxAddress         common.Address `json:"batchInboxAddress"`
	BatchSenderAddress        common.Address `json:"batchSenderAddress"`

	L2OutputOracleSubmissionInterval uint64         `json:"l2OutputOracleSubmissionInterval"`
	L2OutputOracleStartingTimestamp  int            `json:"l2OutputOracleStartingTimestamp"`
	L2OutputOracleProposer           common.Address `json:"l2OutputOracleProposer"`
	L2OutputOracleChallenger         common.Address `json:"l2OutputOracleChallenger"`

	L1BlockTime                 uint64         `json:"l1BlockTime"`
	L1GenesisBlockTimestamp     hexutil.Uint64 `json:"l1GenesisBlockTimestamp"`
	L1GenesisBlockNonce         hexutil.Uint64 `json:"l1GenesisBlockNonce"`
	CliqueSignerAddress         common.Address `json:"cliqueSignerAddress"` // proof of stake genesis if left zeroed.
	L1GenesisBlockGasLimit      hexutil.Uint64 `json:"l1GenesisBlockGasLimit"`
	L1GenesisBlockDifficulty    *hexutil.Big   `json:"l1GenesisBlockDifficulty"`
	L1GenesisBlockMixHash       common.Hash    `json:"l1GenesisBlockMixHash"`
	L1GenesisBlockCoinbase      common.Address `json:"l1GenesisBlockCoinbase"`
	L1GenesisBlockNumber        hexutil.Uint64 `json:"l1GenesisBlockNumber"`
	L1GenesisBlockGasUsed       hexutil.Uint64 `json:"l1GenesisBlockGasUsed"`
	L1GenesisBlockParentHash    common.Hash    `json:"l1GenesisBlockParentHash"`
	L1GenesisBlockBaseFeePerGas *hexutil.Big   `json:"l1GenesisBlockBaseFeePerGas"`

	L2GenesisBlockNonce         hexutil.Uint64 `json:"l2GenesisBlockNonce"`
	L2GenesisBlockGasLimit      hexutil.Uint64 `json:"l2GenesisBlockGasLimit"`
	L2GenesisBlockDifficulty    *hexutil.Big   `json:"l2GenesisBlockDifficulty"`
	L2GenesisBlockMixHash       common.Hash    `json:"l2GenesisBlockMixHash"`
	L2GenesisBlockNumber        hexutil.Uint64 `json:"l2GenesisBlockNumber"`
	L2GenesisBlockGasUsed       hexutil.Uint64 `json:"l2GenesisBlockGasUsed"`
	L2GenesisBlockParentHash    common.Hash    `json:"l2GenesisBlockParentHash"`
	L2GenesisBlockBaseFeePerGas *hexutil.Big   `json:"l2GenesisBlockBaseFeePerGas"`

	// Seconds after genesis block that Regolith hard fork activates. 0 to activate at genesis. Nil to disable regolith
	L2GenesisRegolithTimeOffset *hexutil.Uint64 `json:"l2GenesisRegolithTimeOffset,omitempty"`

	// Owner of the ProxyAdmin predeploy
	ProxyAdminOwner common.Address `json:"proxyAdminOwner"`
	// Owner of the system on L1
	FinalSystemOwner common.Address `json:"finalSystemOwner"`
	// GUARDIAN account in the OptimismPortal
	PortalGuardian common.Address `json:"portalGuardian"`
	// L1 recipient of fees accumulated in the BaseFeeVault
	BaseFeeVaultRecipient common.Address `json:"baseFeeVaultRecipient"`
	// L1 recipient of fees accumulated in the L1FeeVault
	L1FeeVaultRecipient common.Address `json:"l1FeeVaultRecipient"`
	// L1 recipient of fees accumulated in the SequencerFeeVault
	SequencerFeeVaultRecipient common.Address `json:"sequencerFeeVaultRecipient"`
	// L1StandardBridge proxy address on L1
	L1StandardBridgeProxy common.Address `json:"l1StandardBridgeProxy"`
	// L1CrossDomainMessenger proxy address on L1
	L1CrossDomainMessengerProxy common.Address `json:"l1CrossDomainMessengerProxy"`
	// L1ERC721Bridge proxy address on L1
	L1ERC721BridgeProxy common.Address `json:"l1ERC721BridgeProxy"`
	// SystemConfig proxy address on L1
	SystemConfigProxy common.Address `json:"systemConfigProxy"`
	// OptimismPortal proxy address on L1
	OptimismPortalProxy common.Address `json:"optimismPortalProxy"`
	// The initial value of the gas overhead
	GasPriceOracleOverhead uint64 `json:"gasPriceOracleOverhead"`
	// The initial value of the gas scalar
	GasPriceOracleScalar uint64 `json:"gasPriceOracleScalar"`
	// The ERC20 symbol of the GovernanceToken
	GovernanceTokenSymbol string `json:"governanceTokenSymbol"`
	// The ERC20 name of the GovernanceToken
	GovernanceTokenName string `json:"governanceTokenName"`
	// The owner of the GovernanceToken
	GovernanceTokenOwner common.Address `json:"governanceTokenOwner"`

	DeploymentWaitConfirmations int `json:"deploymentWaitConfirmations"`

	EIP1559Elasticity  uint64 `json:"eip1559Elasticity"`
	EIP1559Denominator uint64 `json:"eip1559Denominator"`

	FundDevAccounts bool `json:"fundDevAccounts"`
}

// Check will ensure that the config is sane and return an error when it is not
func (d *DeployConfig) Check() error {
	if d.L1StartingBlockTag == nil {
		return fmt.Errorf("%w: L2StartingBlockTag cannot be nil", ErrInvalidDeployConfig)
	}
	if d.L1ChainID == 0 {
		return fmt.Errorf("%w: L1ChainID cannot be 0", ErrInvalidDeployConfig)
	}
	if d.L2ChainID == 0 {
		return fmt.Errorf("%w: L2ChainID cannot be 0", ErrInvalidDeployConfig)
	}
	if d.L2BlockTime == 0 {
		return fmt.Errorf("%w: L2BlockTime cannot be 0", ErrInvalidDeployConfig)
	}
	if d.FinalizationPeriodSeconds == 0 {
		return fmt.Errorf("%w: FinalizationPeriodSeconds cannot be 0", ErrInvalidDeployConfig)
	}
	if d.PortalGuardian == (common.Address{}) {
		return fmt.Errorf("%w: PortalGuardian cannot be address(0)", ErrInvalidDeployConfig)
	}
	if d.MaxSequencerDrift == 0 {
		return fmt.Errorf("%w: MaxSequencerDrift cannot be 0", ErrInvalidDeployConfig)
	}
	if d.SequencerWindowSize == 0 {
		return fmt.Errorf("%w: SequencerWindowSize cannot be 0", ErrInvalidDeployConfig)
	}
	if d.ChannelTimeout == 0 {
		return fmt.Errorf("%w: ChannelTimeout cannot be 0", ErrInvalidDeployConfig)
	}
	if d.P2PSequencerAddress == (common.Address{}) {
		return fmt.Errorf("%w: P2PSequencerAddress cannot be address(0)", ErrInvalidDeployConfig)
	}
	if d.BatchInboxAddress == (common.Address{}) {
		return fmt.Errorf("%w: BatchInboxAddress cannot be address(0)", ErrInvalidDeployConfig)
	}
	if d.BatchSenderAddress == (common.Address{}) {
		return fmt.Errorf("%w: BatchSenderAddress cannot be address(0)", ErrInvalidDeployConfig)
	}
	if d.L2OutputOracleSubmissionInterval == 0 {
		return fmt.Errorf("%w: L2OutputOracleSubmissionInterval cannot be 0", ErrInvalidDeployConfig)
	}
	if d.L2OutputOracleStartingTimestamp == 0 {
		log.Warn("L2OutputOracleStartingTimestamp is 0")
	}
	if d.L2OutputOracleProposer == (common.Address{}) {
		return fmt.Errorf("%w: L2OutputOracleProposer cannot be address(0)", ErrInvalidDeployConfig)
	}
	if d.L2OutputOracleChallenger == (common.Address{}) {
		return fmt.Errorf("%w: L2OutputOracleChallenger cannot be address(0)", ErrInvalidDeployConfig)
	}
	if d.FinalSystemOwner == (common.Address{}) {
		return fmt.Errorf("%w: FinalSystemOwner cannot be address(0)", ErrInvalidDeployConfig)
	}
	if d.ProxyAdminOwner == (common.Address{}) {
		return fmt.Errorf("%w: ProxyAdminOwner cannot be address(0)", ErrInvalidDeployConfig)
	}
	if d.BaseFeeVaultRecipient == (common.Address{}) {
		return fmt.Errorf("%w: BaseFeeVaultRecipient cannot be address(0)", ErrInvalidDeployConfig)
	}
	if d.L1FeeVaultRecipient == (common.Address{}) {
		return fmt.Errorf("%w: L1FeeVaultRecipient cannot be address(0)", ErrInvalidDeployConfig)
	}
	if d.SequencerFeeVaultRecipient == (common.Address{}) {
		return fmt.Errorf("%w: SequencerFeeVaultRecipient cannot be address(0)", ErrInvalidDeployConfig)
	}
	if d.GasPriceOracleOverhead == 0 {
		log.Warn("GasPriceOracleOverhead is 0")
	}
	if d.GasPriceOracleScalar == 0 {
		return fmt.Errorf("%w: GasPriceOracleScalar cannot be 0", ErrInvalidDeployConfig)
	}
	if d.L1StandardBridgeProxy == (common.Address{}) {
		return fmt.Errorf("%w: L1StandardBridgeProxy cannot be address(0)", ErrInvalidDeployConfig)
	}
	if d.L1CrossDomainMessengerProxy == (common.Address{}) {
		return fmt.Errorf("%w: L1CrossDomainMessengerProxy cannot be address(0)", ErrInvalidDeployConfig)
	}
	if d.L1ERC721BridgeProxy == (common.Address{}) {
		return fmt.Errorf("%w: L1ERC721BridgeProxy cannot be address(0)", ErrInvalidDeployConfig)
	}
	if d.SystemConfigProxy == (common.Address{}) {
		return fmt.Errorf("%w: SystemConfigProxy cannot be address(0)", ErrInvalidDeployConfig)
	}
	if d.OptimismPortalProxy == (common.Address{}) {
		return fmt.Errorf("%w: OptimismPortalProxy cannot be address(0)", ErrInvalidDeployConfig)
	}
	if d.EIP1559Denominator == 0 {
		return fmt.Errorf("%w: EIP1559Denominator cannot be 0", ErrInvalidDeployConfig)
	}
	if d.EIP1559Elasticity == 0 {
		return fmt.Errorf("%w: EIP1559Elasticity cannot be 0", ErrInvalidDeployConfig)
	}
	if d.L2GenesisBlockGasLimit == 0 {
		return fmt.Errorf("%w: L2 genesis block gas limit cannot be 0", ErrInvalidDeployConfig)
	}
	// When the initial resource config is made to be configurable by the DeployConfig, ensure
	// that this check is updated to use the values from the DeployConfig instead of the defaults.
	if uint64(d.L2GenesisBlockGasLimit) < uint64(defaultResourceConfig.MaxResourceLimit+defaultResourceConfig.SystemTxMaxGas) {
		return fmt.Errorf("%w: L2 genesis block gas limit is too small", ErrInvalidDeployConfig)
	}
	if d.L2GenesisBlockBaseFeePerGas == nil {
		return fmt.Errorf("%w: L2 genesis block base fee per gas cannot be nil", ErrInvalidDeployConfig)
	}
	if d.GovernanceTokenName == "" {
		return fmt.Errorf("%w: GovernanceToken.name cannot be empty", ErrInvalidDeployConfig)
	}
	if d.GovernanceTokenSymbol == "" {
		return fmt.Errorf("%w: GovernanceToken.symbol cannot be empty", ErrInvalidDeployConfig)
	}
	if d.GovernanceTokenOwner == (common.Address{}) {
		return fmt.Errorf("%w: GovernanceToken owner cannot be address(0)", ErrInvalidDeployConfig)
	}
	return nil
}

// GetDeployedAddresses will get the deployed addresses of deployed L1 contracts
// required for the L2 genesis creation. Legacy systems use the `Proxy__` prefix
// while modern systems use the `Proxy` suffix. First check for the legacy
// deployments so that this works with upgrading a system.
func (d *DeployConfig) GetDeployedAddresses(hh *hardhat.Hardhat) error {
	var err error

	if d.L1StandardBridgeProxy == (common.Address{}) {
		var l1StandardBridgeProxyDeployment *hardhat.Deployment
		l1StandardBridgeProxyDeployment, err = hh.GetDeployment("Proxy__OVM_L1StandardBridge")
		if errors.Is(err, hardhat.ErrCannotFindDeployment) {
			l1StandardBridgeProxyDeployment, err = hh.GetDeployment("L1StandardBridgeProxy")
			if err != nil {
				return err
			}
		}
		d.L1StandardBridgeProxy = l1StandardBridgeProxyDeployment.Address
	}

	if d.L1CrossDomainMessengerProxy == (common.Address{}) {
		var l1CrossDomainMessengerProxyDeployment *hardhat.Deployment
		l1CrossDomainMessengerProxyDeployment, err = hh.GetDeployment("Proxy__OVM_L1CrossDomainMessenger")
		if errors.Is(err, hardhat.ErrCannotFindDeployment) {
			l1CrossDomainMessengerProxyDeployment, err = hh.GetDeployment("L1CrossDomainMessengerProxy")
			if err != nil {
				return err
			}
		}
		d.L1CrossDomainMessengerProxy = l1CrossDomainMessengerProxyDeployment.Address
	}

	if d.L1ERC721BridgeProxy == (common.Address{}) {
		// There is no legacy deployment of this contract
		l1ERC721BridgeProxyDeployment, err := hh.GetDeployment("L1ERC721BridgeProxy")
		if err != nil {
			return err
		}
		d.L1ERC721BridgeProxy = l1ERC721BridgeProxyDeployment.Address
	}

	if d.SystemConfigProxy == (common.Address{}) {
		systemConfigProxyDeployment, err := hh.GetDeployment("SystemConfigProxy")
		if err != nil {
			return err
		}
		d.SystemConfigProxy = systemConfigProxyDeployment.Address
	}

	if d.OptimismPortalProxy == (common.Address{}) {
		optimismPortalProxyDeployment, err := hh.GetDeployment("OptimismPortalProxy")
		if err != nil {
			return err
		}
		d.OptimismPortalProxy = optimismPortalProxyDeployment.Address
	}

	return nil
}

// InitDeveloperDeployedAddresses will set the dev addresses on the DeployConfig
func (d *DeployConfig) InitDeveloperDeployedAddresses() error {
	d.L1StandardBridgeProxy = predeploys.DevL1StandardBridgeAddr
	d.L1CrossDomainMessengerProxy = predeploys.DevL1CrossDomainMessengerAddr
	d.L1ERC721BridgeProxy = predeploys.DevL1ERC721BridgeAddr
	d.OptimismPortalProxy = predeploys.DevOptimismPortalAddr
	d.SystemConfigProxy = predeploys.DevSystemConfigAddr
	return nil
}

func (d *DeployConfig) RegolithTime(genesisTime uint64) *uint64 {
	if d.L2GenesisRegolithTimeOffset == nil {
		return nil
	}
	v := uint64(0)
	if offset := *d.L2GenesisRegolithTimeOffset; offset > 0 {
		v = genesisTime + uint64(offset)
	}
	return &v
}

// RollupConfig converts a DeployConfig to a rollup.Config
func (d *DeployConfig) RollupConfig(l1StartBlock *types.Block, l2GenesisBlockHash common.Hash, l2GenesisBlockNumber uint64) (*rollup.Config, error) {
	if d.OptimismPortalProxy == (common.Address{}) {
		return nil, errors.New("OptimismPortalProxy cannot be address(0)")
	}
	if d.SystemConfigProxy == (common.Address{}) {
		return nil, errors.New("SystemConfigProxy cannot be address(0)")
	}

	return &rollup.Config{
		Genesis: rollup.Genesis{
			L1: eth.BlockID{
				Hash:   l1StartBlock.Hash(),
				Number: l1StartBlock.NumberU64(),
			},
			L2: eth.BlockID{
				Hash:   l2GenesisBlockHash,
				Number: l2GenesisBlockNumber,
			},
			L2Time: l1StartBlock.Time(),
			SystemConfig: eth.SystemConfig{
				BatcherAddr: d.BatchSenderAddress,
				Overhead:    eth.Bytes32(common.BigToHash(new(big.Int).SetUint64(d.GasPriceOracleOverhead))),
				Scalar:      eth.Bytes32(common.BigToHash(new(big.Int).SetUint64(d.GasPriceOracleScalar))),
				GasLimit:    uint64(d.L2GenesisBlockGasLimit),
			},
		},
		BlockTime:              d.L2BlockTime,
		MaxSequencerDrift:      d.MaxSequencerDrift,
		SeqWindowSize:          d.SequencerWindowSize,
		ChannelTimeout:         d.ChannelTimeout,
		L1ChainID:              new(big.Int).SetUint64(d.L1ChainID),
		L2ChainID:              new(big.Int).SetUint64(d.L2ChainID),
		BatchInboxAddress:      d.BatchInboxAddress,
		DepositContractAddress: d.OptimismPortalProxy,
		L1SystemConfigAddress:  d.SystemConfigProxy,
		RegolithTime:           d.RegolithTime(l1StartBlock.Time()),
	}, nil
}

// NewDeployConfig reads a config file given a path on the filesystem.
func NewDeployConfig(path string) (*DeployConfig, error) {
	file, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("deploy config at %s not found: %w", path, err)
	}

	var config DeployConfig
	if err := json.Unmarshal(file, &config); err != nil {
		return nil, fmt.Errorf("cannot unmarshal deploy config: %w", err)
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
// DeployConfig and a block.
func NewL2ImmutableConfig(config *DeployConfig, block *types.Block) (immutables.ImmutableConfig, error) {
	immutable := make(immutables.ImmutableConfig)

	if config.L1StandardBridgeProxy == (common.Address{}) {
		return immutable, fmt.Errorf("L1StandardBridgeProxy cannot be address(0): %w", ErrInvalidImmutablesConfig)
	}
	if config.L1CrossDomainMessengerProxy == (common.Address{}) {
		return immutable, fmt.Errorf("L1CrossDomainMessengerProxy cannot be address(0): %w", ErrInvalidImmutablesConfig)
	}
	if config.L1ERC721BridgeProxy == (common.Address{}) {
		return immutable, fmt.Errorf("L1ERC721BridgeProxy cannot be address(0): %w", ErrInvalidImmutablesConfig)
	}
	if config.SequencerFeeVaultRecipient == (common.Address{}) {
		return immutable, fmt.Errorf("SequencerFeeVaultRecipient cannot be address(0): %w", ErrInvalidImmutablesConfig)
	}
	if config.BaseFeeVaultRecipient == (common.Address{}) {
		return immutable, fmt.Errorf("BaseFeeVaultRecipient cannot be address(0): %w", ErrInvalidImmutablesConfig)
	}
	if config.L1FeeVaultRecipient == (common.Address{}) {
		return immutable, fmt.Errorf("L1FeeVaultRecipient cannot be address(0): %w", ErrInvalidImmutablesConfig)
	}

	immutable["L2StandardBridge"] = immutables.ImmutableValues{
		"otherBridge": config.L1StandardBridgeProxy,
	}
	immutable["L2CrossDomainMessenger"] = immutables.ImmutableValues{
		"otherMessenger": config.L1CrossDomainMessengerProxy,
	}
	immutable["L2ERC721Bridge"] = immutables.ImmutableValues{
		"messenger":   predeploys.L2CrossDomainMessengerAddr,
		"otherBridge": config.L1ERC721BridgeProxy,
	}
	immutable["OptimismMintableERC721Factory"] = immutables.ImmutableValues{
		"bridge":        predeploys.L2ERC721BridgeAddr,
		"remoteChainId": new(big.Int).SetUint64(config.L1ChainID),
	}
	immutable["SequencerFeeVault"] = immutables.ImmutableValues{
		"recipient": config.SequencerFeeVaultRecipient,
	}
	immutable["L1FeeVault"] = immutables.ImmutableValues{
		"recipient": config.L1FeeVaultRecipient,
	}
	immutable["BaseFeeVault"] = immutables.ImmutableValues{
		"recipient": config.BaseFeeVaultRecipient,
	}

	return immutable, nil
}

// NewL2StorageConfig will create a StorageConfig given an instance of a
// Hardhat and a DeployConfig.
func NewL2StorageConfig(config *DeployConfig, block *types.Block) (state.StorageConfig, error) {
	storage := make(state.StorageConfig)

	if block.Number() == nil {
		return storage, errors.New("block number not set")
	}
	if block.BaseFee() == nil {
		return storage, errors.New("block base fee not set")
	}

	storage["L2ToL1MessagePasser"] = state.StorageValues{
		"msgNonce": 0,
	}
	storage["L2CrossDomainMessenger"] = state.StorageValues{
		"_initialized":     1,
		"_initializing":    false,
		"xDomainMsgSender": "0x000000000000000000000000000000000000dEaD",
		"msgNonce":         0,
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
		"_name":   "Ether",
		"_symbol": "ETH",
	}
	storage["WETH9"] = state.StorageValues{
		"name":     "Wrapped Ether",
		"symbol":   "WETH",
		"decimals": 18,
	}
	storage["GovernanceToken"] = state.StorageValues{
		"_name":   config.GovernanceTokenName,
		"_symbol": config.GovernanceTokenSymbol,
		"_owner":  config.GovernanceTokenOwner,
	}
	storage["ProxyAdmin"] = state.StorageValues{
		"_owner": config.ProxyAdminOwner,
	}
	return storage, nil
}

type MarshalableRPCBlockNumberOrHash rpc.BlockNumberOrHash

func (m *MarshalableRPCBlockNumberOrHash) MarshalJSON() ([]byte, error) {
	r := rpc.BlockNumberOrHash(*m)
	if hash, ok := r.Hash(); ok {
		return json.Marshal(hash)
	}
	if num, ok := r.Number(); ok {
		// never errors
		text, _ := num.MarshalText()
		return json.Marshal(string(text))
	}
	return json.Marshal(nil)
}

func (m *MarshalableRPCBlockNumberOrHash) UnmarshalJSON(b []byte) error {
	var r rpc.BlockNumberOrHash
	if err := json.Unmarshal(b, &r); err != nil {
		return err
	}

	asMarshalable := MarshalableRPCBlockNumberOrHash(r)
	*m = asMarshalable
	return nil
}

// Number wraps the rpc.BlockNumberOrHash Number method.
func (m *MarshalableRPCBlockNumberOrHash) Number() (rpc.BlockNumber, bool) {
	return (*rpc.BlockNumberOrHash)(m).Number()
}

// Hash wraps the rpc.BlockNumberOrHash Hash method.
func (m *MarshalableRPCBlockNumberOrHash) Hash() (common.Hash, bool) {
	return (*rpc.BlockNumberOrHash)(m).Hash()
}

// String wraps the rpc.BlockNumberOrHash String method.
func (m *MarshalableRPCBlockNumberOrHash) String() string {
	return (*rpc.BlockNumberOrHash)(m).String()
}
