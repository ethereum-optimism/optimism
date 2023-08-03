package genesis

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"os"
	"path/filepath"

	"reflect"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	gstate "github.com/ethereum/go-ethereum/core/state"
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

// DeployConfig represents the deployment configuration for an OP Stack chain.
type DeployConfig struct {
	// L1StartingBlockTag is used to fill in the storage of the L1Block info predeploy. The rollup
	// config script uses this to fill the L1 genesis info for the rollup. The Output oracle deploy
	// script may use it if the L2 starting timestamp is nil, assuming the L2 genesis is set up
	// with this.
	L1StartingBlockTag *MarshalableRPCBlockNumberOrHash `json:"l1StartingBlockTag"`
	// L1ChainID is the chain ID of the L1 chain.
	L1ChainID uint64 `json:"l1ChainID"`
	// L2ChainID is the chain ID of the L2 chain.
	L2ChainID uint64 `json:"l2ChainID"`
	// L2BlockTime is the number of seconds between each L2 block.
	L2BlockTime uint64 `json:"l2BlockTime"`
	// FinalizationPeriodSeconds represents the number of seconds before an output is considered
	// finalized. This impacts the amount of time that withdrawals take to finalize and is
	// generally set to 1 week.
	FinalizationPeriodSeconds uint64 `json:"finalizationPeriodSeconds"`
	// MaxSequencerDrift is the number of seconds after the L1 timestamp of the end of the
	// sequencing window that batches must be included, otherwise L2 blocks including
	// deposits are force included.
	MaxSequencerDrift uint64 `json:"maxSequencerDrift"`
	// SequencerWindowSize is the number of L1 blocks per sequencing window.
	SequencerWindowSize uint64 `json:"sequencerWindowSize"`
	// ChannelTimeout is the number of L1 blocks that a frame stays valid when included in L1.
	ChannelTimeout uint64 `json:"channelTimeout"`
	// P2PSequencerAddress is the address of the key the sequencer uses to sign blocks on the P2P layer.
	P2PSequencerAddress common.Address `json:"p2pSequencerAddress"`
	// BatchInboxAddress is the L1 account that batches are sent to.
	BatchInboxAddress common.Address `json:"batchInboxAddress"`
	// BatchSenderAddress represents the initial sequencer account that authorizes batches.
	// Transactions sent from this account to the batch inbox address are considered valid.
	BatchSenderAddress common.Address `json:"batchSenderAddress"`
	// L2OutputOracleSubmissionInterval is the number of L2 blocks between outputs that are submitted
	// to the L2OutputOracle contract located on L1.
	L2OutputOracleSubmissionInterval uint64 `json:"l2OutputOracleSubmissionInterval"`
	// L2OutputOracleStartingTimestamp is the starting timestamp for the L2OutputOracle.
	// MUST be the same as the timestamp of the L2OO start block.
	L2OutputOracleStartingTimestamp int `json:"l2OutputOracleStartingTimestamp"`
	// L2OutputOracleStartingBlockNumber is the starting block number for the L2OutputOracle.
	// Must be greater than or equal to the first Bedrock block. The first L2 output will correspond
	// to this value plus the submission interval.
	L2OutputOracleStartingBlockNumber uint64 `json:"l2OutputOracleStartingBlockNumber"`
	// L2OutputOracleProposer is the address of the account that proposes L2 outputs.
	L2OutputOracleProposer common.Address `json:"l2OutputOracleProposer"`
	// L2OutputOracleChallenger is the address of the account that challenges L2 outputs.
	L2OutputOracleChallenger common.Address `json:"l2OutputOracleChallenger"`

	// CliqueSignerAddress represents the signer address for the clique consensus engine.
	// It is used in the multi-process devnet to sign blocks.
	CliqueSignerAddress common.Address `json:"cliqueSignerAddress"`
	// L1UseClique represents whether or not to use the clique consensus engine.
	L1UseClique bool `json:"l1UseClique"`

	L1BlockTime                 uint64         `json:"l1BlockTime"`
	L1GenesisBlockTimestamp     hexutil.Uint64 `json:"l1GenesisBlockTimestamp"`
	L1GenesisBlockNonce         hexutil.Uint64 `json:"l1GenesisBlockNonce"`
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

	// L2GenesisRegolithTimeOffset is the number of seconds after genesis block that Regolith hard fork activates.
	// Set it to 0 to activate at genesis. Nil to disable regolith.
	L2GenesisRegolithTimeOffset *hexutil.Uint64 `json:"l2GenesisRegolithTimeOffset,omitempty"`
	// L2GenesisBlockExtraData is configurable extradata. Will default to []byte("BEDROCK") if left unspecified.
	L2GenesisBlockExtraData []byte `json:"l2GenesisBlockExtraData"`
	// ProxyAdminOwner represents the owner of the ProxyAdmin predeploy on L2.
	ProxyAdminOwner common.Address `json:"proxyAdminOwner"`
	// FinalSystemOwner is the owner of the system on L1. Any L1 contract that is ownable has
	// this account set as its owner.
	FinalSystemOwner common.Address `json:"finalSystemOwner"`
	// PortalGuardian represents the GUARDIAN account in the OptimismPortal. Has the ability to pause withdrawals.
	PortalGuardian common.Address `json:"portalGuardian"`
	// BaseFeeVaultRecipient represents the recipient of fees accumulated in the BaseFeeVault.
	// Can be an account on L1 or L2, depending on the BaseFeeVaultWithdrawalNetwork value.
	BaseFeeVaultRecipient common.Address `json:"baseFeeVaultRecipient"`
	// L1FeeVaultRecipient represents the recipient of fees accumulated in the L1FeeVault.
	// Can be an account on L1 or L2, depending on the L1FeeVaultWithdrawalNetwork value.
	L1FeeVaultRecipient common.Address `json:"l1FeeVaultRecipient"`
	// SequencerFeeVaultRecipient represents the recipient of fees accumulated in the SequencerFeeVault.
	// Can be an account on L1 or L2, depending on the SequencerFeeVaultWithdrawalNetwork value.
	SequencerFeeVaultRecipient common.Address `json:"sequencerFeeVaultRecipient"`
	// BaseFeeVaultMinimumWithdrawalAmount represents the minimum withdrawal amount for the BaseFeeVault.
	BaseFeeVaultMinimumWithdrawalAmount *hexutil.Big `json:"baseFeeVaultMinimumWithdrawalAmount"`
	// L1FeeVaultMinimumWithdrawalAmount represents the minimum withdrawal amount for the L1FeeVault.
	L1FeeVaultMinimumWithdrawalAmount *hexutil.Big `json:"l1FeeVaultMinimumWithdrawalAmount"`
	// SequencerFeeVaultMinimumWithdrawalAmount represents the minimum withdrawal amount for the SequencerFeeVault.
	SequencerFeeVaultMinimumWithdrawalAmount *hexutil.Big `json:"sequencerFeeVaultMinimumWithdrawalAmount"`
	// BaseFeeVaultWithdrawalNetwork represents the withdrawal network for the BaseFeeVault.
	BaseFeeVaultWithdrawalNetwork WithdrawalNetwork `json:"baseFeeVaultWithdrawalNetwork"`
	// L1FeeVaultWithdrawalNetwork represents the withdrawal network for the L1FeeVault.
	L1FeeVaultWithdrawalNetwork WithdrawalNetwork `json:"l1FeeVaultWithdrawalNetwork"`
	// SequencerFeeVaultWithdrawalNetwork represents the withdrawal network for the SequencerFeeVault.
	SequencerFeeVaultWithdrawalNetwork WithdrawalNetwork `json:"sequencerFeeVaultWithdrawalNetwork"`
	// L1StandardBridgeProxy represents the address of the L1StandardBridgeProxy on L1 and is used
	// as part of building the L2 genesis state.
	L1StandardBridgeProxy common.Address `json:"l1StandardBridgeProxy"`
	// L1CrossDomainMessengerProxy represents the address of the L1CrossDomainMessengerProxy on L1 and is used
	// as part of building the L2 genesis state.
	L1CrossDomainMessengerProxy common.Address `json:"l1CrossDomainMessengerProxy"`
	// L1ERC721BridgeProxy represents the address of the L1ERC721Bridge on L1 and is used
	// as part of building the L2 genesis state.
	L1ERC721BridgeProxy common.Address `json:"l1ERC721BridgeProxy"`
	// SystemConfigProxy represents the address of the SystemConfigProxy on L1 and is used
	// as part of the derivation pipeline.
	SystemConfigProxy common.Address `json:"systemConfigProxy"`
	// OptimismPortalProxy represents the address of the OptimismPortalProxy on L1 and is used
	// as part of the derivation pipeline.
	OptimismPortalProxy common.Address `json:"optimismPortalProxy"`
	// GasPriceOracleOverhead represents the initial value of the gas overhead in the GasPriceOracle predeploy.
	GasPriceOracleOverhead uint64 `json:"gasPriceOracleOverhead"`
	// GasPriceOracleScalar represents the initial value of the gas scalar in the GasPriceOracle predeploy.
	GasPriceOracleScalar uint64 `json:"gasPriceOracleScalar"`
	// EnableGovernance configures whether or not include governance token predeploy.
	EnableGovernance bool `json:"enableGovernance"`
	// GovernanceTokenSymbol represents the  ERC20 symbol of the GovernanceToken.
	GovernanceTokenSymbol string `json:"governanceTokenSymbol"`
	// GovernanceTokenName represents the ERC20 name of the GovernanceToken
	GovernanceTokenName string `json:"governanceTokenName"`
	// GovernanceTokenOwner represents the owner of the GovernanceToken. Has the ability
	// to mint and burn tokens.
	GovernanceTokenOwner common.Address `json:"governanceTokenOwner"`
	// DeploymentWaitConfirmations is the number of confirmations to wait during
	// deployment. This is DEPRECATED and should be removed in a future PR.
	DeploymentWaitConfirmations int `json:"deploymentWaitConfirmations"`
	// EIP1559Elasticity is the elasticity of the EIP1559 fee market.
	EIP1559Elasticity uint64 `json:"eip1559Elasticity"`
	// EIP1559Denominator is the denominator of EIP1559 base fee market.
	EIP1559Denominator uint64 `json:"eip1559Denominator"`
	// FundDevAccounts configures whether or not to fund the dev accounts. Should only be used
	// during devnet deployments.
	FundDevAccounts bool `json:"fundDevAccounts"`
}

// Copy will deeply copy the DeployConfig. This does a JSON roundtrip to copy
// which makes it easier to maintain, we do not need efficiency in this case.
func (d *DeployConfig) Copy() *DeployConfig {
	raw, err := json.Marshal(d)
	if err != nil {
		panic(err)
	}

	cpy := DeployConfig{}
	if err = json.Unmarshal(raw, &cpy); err != nil {
		panic(err)
	}
	return &cpy
}

// Check will ensure that the config is sane and return an error when it is not
func (d *DeployConfig) Check() error {
	if d.L1StartingBlockTag == nil {
		return fmt.Errorf("%w: L1StartingBlockTag cannot be nil", ErrInvalidDeployConfig)
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
	if d.L2OutputOracleStartingBlockNumber == 0 {
		log.Warn("L2OutputOracleStartingBlockNumber is 0, should only be 0 for fresh chains")
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
	if !d.BaseFeeVaultWithdrawalNetwork.Valid() {
		return fmt.Errorf("%w: BaseFeeVaultWithdrawalNetwork can only be 0 (L1) or 1 (L2)", ErrInvalidDeployConfig)
	}
	if !d.L1FeeVaultWithdrawalNetwork.Valid() {
		return fmt.Errorf("%w: L1FeeVaultWithdrawalNetwork can only be 0 (L1) or 1 (L2)", ErrInvalidDeployConfig)
	}
	if !d.SequencerFeeVaultWithdrawalNetwork.Valid() {
		return fmt.Errorf("%w: SequencerFeeVaultWithdrawalNetwork can only be 0 (L1) or 1 (L2)", ErrInvalidDeployConfig)
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
	if d.EnableGovernance {
		if d.GovernanceTokenName == "" {
			return fmt.Errorf("%w: GovernanceToken.name cannot be empty", ErrInvalidDeployConfig)
		}
		if d.GovernanceTokenSymbol == "" {
			return fmt.Errorf("%w: GovernanceToken.symbol cannot be empty", ErrInvalidDeployConfig)
		}
		if d.GovernanceTokenOwner == (common.Address{}) {
			return fmt.Errorf("%w: GovernanceToken owner cannot be address(0)", ErrInvalidDeployConfig)
		}
	}
	// L2 block time must always be smaller than L1 block time
	if d.L1BlockTime < d.L2BlockTime {
		return fmt.Errorf("L2 block time (%d) is larger than L1 block time (%d)", d.L2BlockTime, d.L1BlockTime)
	}
	return nil
}

// SetDeployments will merge a Deployments into a DeployConfig.
func (d *DeployConfig) SetDeployments(deployments *L1Deployments) {
	d.L1StandardBridgeProxy = deployments.L1StandardBridgeProxy
	d.L1CrossDomainMessengerProxy = deployments.L1CrossDomainMessengerProxy
	d.L1ERC721BridgeProxy = deployments.L1ERC721BridgeProxy
	d.SystemConfigProxy = deployments.SystemConfigProxy
	d.OptimismPortalProxy = deployments.OptimismPortalProxy
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

// L1Deployments represents a set of L1 contracts that are deployed.
type L1Deployments struct {
	AddressManager                    common.Address `json:"AddressManager"`
	DisputeGameFactory                common.Address `json:"DisputeGameFactory"`
	DisputeGameFactoryProxy           common.Address `json:"DisputeGameFactoryProxy"`
	L1CrossDomainMessenger            common.Address `json:"L1CrossDomainMessenger"`
	L1CrossDomainMessengerProxy       common.Address `json:"L1CrossDomainMessengerProxy"`
	L1ERC721Bridge                    common.Address `json:"L1ERC721Bridge"`
	L1ERC721BridgeProxy               common.Address `json:"L1ERC721BridgeProxy"`
	L1StandardBridge                  common.Address `json:"L1StandardBridge"`
	L1StandardBridgeProxy             common.Address `json:"L1StandardBridgeProxy"`
	L2OutputOracle                    common.Address `json:"L2OutputOracle"`
	L2OutputOracleProxy               common.Address `json:"L2OutputOracleProxy"`
	OptimismMintableERC20Factory      common.Address `json:"OptimismMintableERC20Factory"`
	OptimismMintableERC20FactoryProxy common.Address `json:"OptimismMintableERC20FactoryProxy"`
	OptimismPortal                    common.Address `json:"OptimismPortal"`
	OptimismPortalProxy               common.Address `json:"OptimismPortalProxy"`
	ProxyAdmin                        common.Address `json:"ProxyAdmin"`
	SystemConfig                      common.Address `json:"SystemConfig"`
	SystemConfigProxy                 common.Address `json:"SystemConfigProxy"`
}

// GetName will return the name of the contract given an address.
func (d *L1Deployments) GetName(addr common.Address) string {
	val := reflect.ValueOf(d)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	for i := 0; i < val.NumField(); i++ {
		if addr == val.Field(i).Interface().(common.Address) {
			return val.Type().Field(i).Name
		}
	}
	return ""
}

// Check will ensure that the L1Deployments are sane
func (d *L1Deployments) Check() error {
	val := reflect.ValueOf(d)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	for i := 0; i < val.NumField(); i++ {
		name := val.Type().Field(i).Name
		// Skip the non production ready contracts
		if name == "DisputeGameFactory" || name == "DisputeGameFactoryProxy" {
			continue
		}
		if val.Field(i).Interface().(common.Address) == (common.Address{}) {
			return fmt.Errorf("%s is not set", name)
		}
	}
	return nil
}

// ForEach will iterate over each contract in the L1Deployments
func (d *L1Deployments) ForEach(cb func(name string, addr common.Address)) {
	val := reflect.ValueOf(d)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	for i := 0; i < val.NumField(); i++ {
		name := val.Type().Field(i).Name
		cb(name, val.Field(i).Interface().(common.Address))
	}
}

// Copy will copy the L1Deployments struct
func (d *L1Deployments) Copy() *L1Deployments {
	cpy := L1Deployments{}
	data, err := json.Marshal(d)
	if err != nil {
		panic(err)
	}
	if err := json.Unmarshal(data, &cpy); err != nil {
		panic(err)
	}
	return &cpy
}

// NewL1Deployments will create a new L1Deployments from a JSON file on disk
// at the given path.
func NewL1Deployments(path string) (*L1Deployments, error) {
	file, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("L1 deployments at %s not found: %w", path, err)
	}

	var deployments L1Deployments
	if err := json.Unmarshal(file, &deployments); err != nil {
		return nil, fmt.Errorf("cannot unmarshal L1 deployements: %w", err)
	}

	return &deployments, nil
}

// NewStateDump will read a Dump JSON file from disk
func NewStateDump(path string) (*gstate.Dump, error) {
	file, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("dump at %s not found: %w", path, err)
	}

	var dump gstate.Dump
	if err := json.Unmarshal(file, &dump); err != nil {
		return nil, fmt.Errorf("cannot unmarshal dump: %w", err)
	}
	return &dump, nil
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
		"recipient":               config.SequencerFeeVaultRecipient,
		"minimumWithdrawalAmount": config.SequencerFeeVaultMinimumWithdrawalAmount,
		"withdrawalNetwork":       config.SequencerFeeVaultWithdrawalNetwork.ToUint8(),
	}
	immutable["L1FeeVault"] = immutables.ImmutableValues{
		"recipient":               config.L1FeeVaultRecipient,
		"minimumWithdrawalAmount": config.L1FeeVaultMinimumWithdrawalAmount,
		"withdrawalNetwork":       config.L1FeeVaultWithdrawalNetwork.ToUint8(),
	}
	immutable["BaseFeeVault"] = immutables.ImmutableValues{
		"recipient":               config.BaseFeeVaultRecipient,
		"minimumWithdrawalAmount": config.BaseFeeVaultMinimumWithdrawalAmount,
		"withdrawalNetwork":       config.BaseFeeVaultWithdrawalNetwork.ToUint8(),
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
	if config.EnableGovernance {
		storage["GovernanceToken"] = state.StorageValues{
			"_name":   config.GovernanceTokenName,
			"_symbol": config.GovernanceTokenSymbol,
			"_owner":  config.GovernanceTokenOwner,
		}
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
