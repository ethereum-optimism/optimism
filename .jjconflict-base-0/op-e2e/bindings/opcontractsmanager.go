// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package bindings

import (
	"errors"
	"math/big"
	"strings"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
)

// Reference imports to suppress errors if they are not otherwise used.
var (
	_ = errors.New
	_ = big.NewInt
	_ = strings.NewReader
	_ = ethereum.NotFound
	_ = bind.Bind
	_ = common.Big1
	_ = types.BloomLookup
	_ = event.NewSubscription
	_ = abi.ConvertType
)

// OPContractsManagerAddGameInput is an auto generated low-level Go binding around an user-defined struct.
type OPContractsManagerAddGameInput struct {
	SaltMixer               string
	SystemConfig            common.Address
	DelayedWETH             common.Address
	DisputeGameType         uint32
	DisputeAbsolutePrestate [32]byte
	DisputeMaxGameDepth     *big.Int
	DisputeSplitDepth       *big.Int
	DisputeClockExtension   uint64
	DisputeMaxClockDuration uint64
	InitialBond             *big.Int
	Vm                      common.Address
	Permissioned            bool
}

// OPContractsManagerAddGameOutput is an auto generated low-level Go binding around an user-defined struct.
type OPContractsManagerAddGameOutput struct {
	DelayedWETH      common.Address
	FaultDisputeGame common.Address
}

// OPContractsManagerBlueprints is an auto generated low-level Go binding around an user-defined struct.
type OPContractsManagerBlueprints struct {
	AddressManager        common.Address
	Proxy                 common.Address
	ProxyAdmin            common.Address
	L1ChugSplashProxy     common.Address
	ResolvedDelegateProxy common.Address
}

// OPContractsManagerDeployInput is an auto generated low-level Go binding around an user-defined struct.
type OPContractsManagerDeployInput struct {
	Roles                   OPContractsManagerRoles
	BasefeeScalar           uint32
	BlobBasefeeScalar       uint32
	L2ChainId               *big.Int
	StartingAnchorRoot      []byte
	SaltMixer               string
	GasLimit                uint64
	DisputeGameType         uint32
	DisputeAbsolutePrestate [32]byte
	DisputeMaxGameDepth     *big.Int
	DisputeSplitDepth       *big.Int
	DisputeClockExtension   uint64
	DisputeMaxClockDuration uint64
	UseCustomGasToken       bool
}

// OPContractsManagerDeployOutput is an auto generated low-level Go binding around an user-defined struct.
type OPContractsManagerDeployOutput struct {
	OpChainProxyAdmin                  common.Address
	AddressManager                     common.Address
	L1ERC721BridgeProxy                common.Address
	SystemConfigProxy                  common.Address
	OptimismMintableERC20FactoryProxy  common.Address
	L1StandardBridgeProxy              common.Address
	L1CrossDomainMessengerProxy        common.Address
	EthLockboxProxy                    common.Address
	OptimismPortalProxy                common.Address
	DisputeGameFactoryProxy            common.Address
	AnchorStateRegistryProxy           common.Address
	FaultDisputeGame                   common.Address
	PermissionedDisputeGame            common.Address
	DelayedWETHPermissionedGameProxy   common.Address
	DelayedWETHPermissionlessGameProxy common.Address
}

// OPContractsManagerImplementations is an auto generated low-level Go binding around an user-defined struct.
type OPContractsManagerImplementations struct {
	SuperchainConfigImpl             common.Address
	ProtocolVersionsImpl             common.Address
	L1ERC721BridgeImpl               common.Address
	OptimismPortalImpl               common.Address
	OptimismPortalInteropImpl        common.Address
	EthLockboxImpl                   common.Address
	SystemConfigImpl                 common.Address
	OptimismMintableERC20FactoryImpl common.Address
	L1CrossDomainMessengerImpl       common.Address
	L1StandardBridgeImpl             common.Address
	DisputeGameFactoryImpl           common.Address
	AnchorStateRegistryImpl          common.Address
	DelayedWETHImpl                  common.Address
	MipsImpl                         common.Address
	FaultDisputeGameImpl             common.Address
	PermissionedDisputeGameImpl      common.Address
	SuperFaultDisputeGameImpl        common.Address
	SuperPermissionedDisputeGameImpl common.Address
}

// OPContractsManagerInteropMigratorGameParameters is an auto generated low-level Go binding around an user-defined struct.
type OPContractsManagerInteropMigratorGameParameters struct {
	Proposer         common.Address
	Challenger       common.Address
	MaxGameDepth     *big.Int
	SplitDepth       *big.Int
	InitBond         *big.Int
	ClockExtension   uint64
	MaxClockDuration uint64
}

// OPContractsManagerInteropMigratorMigrateInput is an auto generated low-level Go binding around an user-defined struct.
type OPContractsManagerInteropMigratorMigrateInput struct {
	UsePermissionlessGame bool
	StartingAnchorRoot    Proposal
	GameParameters        OPContractsManagerInteropMigratorGameParameters
	OpChainConfigs        []OPContractsManagerOpChainConfig
}

// OPContractsManagerOpChainConfig is an auto generated low-level Go binding around an user-defined struct.
type OPContractsManagerOpChainConfig struct {
	SystemConfigProxy  common.Address
	CannonPrestate     [32]byte
	CannonKonaPrestate [32]byte
}

// OPContractsManagerRoles is an auto generated low-level Go binding around an user-defined struct.
type OPContractsManagerRoles struct {
	OpChainProxyAdminOwner common.Address
	SystemConfigOwner      common.Address
	Batcher                common.Address
	UnsafeBlockSigner      common.Address
	Proposer               common.Address
	Challenger             common.Address
}

// OPContractsManagerStandardValidatorValidationInput is an auto generated low-level Go binding around an user-defined struct.
type OPContractsManagerStandardValidatorValidationInput struct {
	SysCfg           common.Address
	AbsolutePrestate [32]byte
	L2ChainID        *big.Int
	Proposer         common.Address
}

// OPContractsManagerStandardValidatorValidationInputDev is an auto generated low-level Go binding around an user-defined struct.
type OPContractsManagerStandardValidatorValidationInputDev struct {
	SysCfg             common.Address
	CannonPrestate     [32]byte
	CannonKonaPrestate [32]byte
	L2ChainID          *big.Int
	Proposer           common.Address
}

// OPContractsManagerStandardValidatorValidationOverrides is an auto generated low-level Go binding around an user-defined struct.
type OPContractsManagerStandardValidatorValidationOverrides struct {
	L1PAOMultisig common.Address
	Challenger    common.Address
}

// OPContractsManagerUpdatePrestateInput is an auto generated low-level Go binding around an user-defined struct.
type OPContractsManagerUpdatePrestateInput struct {
	SystemConfigProxy  common.Address
	CannonPrestate     [32]byte
	CannonKonaPrestate [32]byte
}

// Proposal is an auto generated low-level Go binding around an user-defined struct.
type Proposal struct {
	Root             [32]byte
	L2SequenceNumber *big.Int
}

// OPContractsManagerMetaData contains all meta data concerning the OPContractsManager contract.
var OPContractsManagerMetaData = &bind.MetaData{
	ABI: "[{\"type\":\"constructor\",\"inputs\":[{\"name\":\"_opcmGameTypeAdder\",\"type\":\"address\",\"internalType\":\"contractOPContractsManagerGameTypeAdder\"},{\"name\":\"_opcmDeployer\",\"type\":\"address\",\"internalType\":\"contractOPContractsManagerDeployer\"},{\"name\":\"_opcmUpgrader\",\"type\":\"address\",\"internalType\":\"contractOPContractsManagerUpgrader\"},{\"name\":\"_opcmInteropMigrator\",\"type\":\"address\",\"internalType\":\"contractOPContractsManagerInteropMigrator\"},{\"name\":\"_opcmStandardValidator\",\"type\":\"address\",\"internalType\":\"contractOPContractsManagerStandardValidator\"},{\"name\":\"_superchainConfig\",\"type\":\"address\",\"internalType\":\"contractISuperchainConfig\"},{\"name\":\"_protocolVersions\",\"type\":\"address\",\"internalType\":\"contractIProtocolVersions\"}],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"addGameType\",\"inputs\":[{\"name\":\"_gameConfigs\",\"type\":\"tuple[]\",\"internalType\":\"structOPContractsManager.AddGameInput[]\",\"components\":[{\"name\":\"saltMixer\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"systemConfig\",\"type\":\"address\",\"internalType\":\"contractISystemConfig\"},{\"name\":\"delayedWETH\",\"type\":\"address\",\"internalType\":\"contractIDelayedWETH\"},{\"name\":\"disputeGameType\",\"type\":\"uint32\",\"internalType\":\"GameType\"},{\"name\":\"disputeAbsolutePrestate\",\"type\":\"bytes32\",\"internalType\":\"Claim\"},{\"name\":\"disputeMaxGameDepth\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"disputeSplitDepth\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"disputeClockExtension\",\"type\":\"uint64\",\"internalType\":\"Duration\"},{\"name\":\"disputeMaxClockDuration\",\"type\":\"uint64\",\"internalType\":\"Duration\"},{\"name\":\"initialBond\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"vm\",\"type\":\"address\",\"internalType\":\"contractIBigStepper\"},{\"name\":\"permissioned\",\"type\":\"bool\",\"internalType\":\"bool\"}]}],\"outputs\":[{\"name\":\"\",\"type\":\"tuple[]\",\"internalType\":\"structOPContractsManager.AddGameOutput[]\",\"components\":[{\"name\":\"delayedWETH\",\"type\":\"address\",\"internalType\":\"contractIDelayedWETH\"},{\"name\":\"faultDisputeGame\",\"type\":\"address\",\"internalType\":\"contractIFaultDisputeGame\"}]}],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"blueprints\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"tuple\",\"internalType\":\"structOPContractsManager.Blueprints\",\"components\":[{\"name\":\"addressManager\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"proxy\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"proxyAdmin\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"l1ChugSplashProxy\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"resolvedDelegateProxy\",\"type\":\"address\",\"internalType\":\"address\"}]}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"chainIdToBatchInboxAddress\",\"inputs\":[{\"name\":\"_l2ChainId\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"deploy\",\"inputs\":[{\"name\":\"_input\",\"type\":\"tuple\",\"internalType\":\"structOPContractsManager.DeployInput\",\"components\":[{\"name\":\"roles\",\"type\":\"tuple\",\"internalType\":\"structOPContractsManager.Roles\",\"components\":[{\"name\":\"opChainProxyAdminOwner\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"systemConfigOwner\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"batcher\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"unsafeBlockSigner\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"proposer\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"challenger\",\"type\":\"address\",\"internalType\":\"address\"}]},{\"name\":\"basefeeScalar\",\"type\":\"uint32\",\"internalType\":\"uint32\"},{\"name\":\"blobBasefeeScalar\",\"type\":\"uint32\",\"internalType\":\"uint32\"},{\"name\":\"l2ChainId\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"startingAnchorRoot\",\"type\":\"bytes\",\"internalType\":\"bytes\"},{\"name\":\"saltMixer\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"gasLimit\",\"type\":\"uint64\",\"internalType\":\"uint64\"},{\"name\":\"disputeGameType\",\"type\":\"uint32\",\"internalType\":\"GameType\"},{\"name\":\"disputeAbsolutePrestate\",\"type\":\"bytes32\",\"internalType\":\"Claim\"},{\"name\":\"disputeMaxGameDepth\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"disputeSplitDepth\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"disputeClockExtension\",\"type\":\"uint64\",\"internalType\":\"Duration\"},{\"name\":\"disputeMaxClockDuration\",\"type\":\"uint64\",\"internalType\":\"Duration\"},{\"name\":\"useCustomGasToken\",\"type\":\"bool\",\"internalType\":\"bool\"}]}],\"outputs\":[{\"name\":\"\",\"type\":\"tuple\",\"internalType\":\"structOPContractsManager.DeployOutput\",\"components\":[{\"name\":\"opChainProxyAdmin\",\"type\":\"address\",\"internalType\":\"contractIProxyAdmin\"},{\"name\":\"addressManager\",\"type\":\"address\",\"internalType\":\"contractIAddressManager\"},{\"name\":\"l1ERC721BridgeProxy\",\"type\":\"address\",\"internalType\":\"contractIL1ERC721Bridge\"},{\"name\":\"systemConfigProxy\",\"type\":\"address\",\"internalType\":\"contractISystemConfig\"},{\"name\":\"optimismMintableERC20FactoryProxy\",\"type\":\"address\",\"internalType\":\"contractIOptimismMintableERC20Factory\"},{\"name\":\"l1StandardBridgeProxy\",\"type\":\"address\",\"internalType\":\"contractIL1StandardBridge\"},{\"name\":\"l1CrossDomainMessengerProxy\",\"type\":\"address\",\"internalType\":\"contractIL1CrossDomainMessenger\"},{\"name\":\"ethLockboxProxy\",\"type\":\"address\",\"internalType\":\"contractIETHLockbox\"},{\"name\":\"optimismPortalProxy\",\"type\":\"address\",\"internalType\":\"contractIOptimismPortal2\"},{\"name\":\"disputeGameFactoryProxy\",\"type\":\"address\",\"internalType\":\"contractIDisputeGameFactory\"},{\"name\":\"anchorStateRegistryProxy\",\"type\":\"address\",\"internalType\":\"contractIAnchorStateRegistry\"},{\"name\":\"faultDisputeGame\",\"type\":\"address\",\"internalType\":\"contractIFaultDisputeGame\"},{\"name\":\"permissionedDisputeGame\",\"type\":\"address\",\"internalType\":\"contractIPermissionedDisputeGame\"},{\"name\":\"delayedWETHPermissionedGameProxy\",\"type\":\"address\",\"internalType\":\"contractIDelayedWETH\"},{\"name\":\"delayedWETHPermissionlessGameProxy\",\"type\":\"address\",\"internalType\":\"contractIDelayedWETH\"}]}],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"devFeatureBitmap\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"implementations\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"tuple\",\"internalType\":\"structOPContractsManager.Implementations\",\"components\":[{\"name\":\"superchainConfigImpl\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"protocolVersionsImpl\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"l1ERC721BridgeImpl\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"optimismPortalImpl\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"optimismPortalInteropImpl\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"ethLockboxImpl\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"systemConfigImpl\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"optimismMintableERC20FactoryImpl\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"l1CrossDomainMessengerImpl\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"l1StandardBridgeImpl\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"disputeGameFactoryImpl\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"anchorStateRegistryImpl\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"delayedWETHImpl\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"mipsImpl\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"faultDisputeGameImpl\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"permissionedDisputeGameImpl\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"superFaultDisputeGameImpl\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"superPermissionedDisputeGameImpl\",\"type\":\"address\",\"internalType\":\"address\"}]}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"isDevFeatureEnabled\",\"inputs\":[{\"name\":\"_feature\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"outputs\":[{\"name\":\"\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"migrate\",\"inputs\":[{\"name\":\"_input\",\"type\":\"tuple\",\"internalType\":\"structOPContractsManagerInteropMigrator.MigrateInput\",\"components\":[{\"name\":\"usePermissionlessGame\",\"type\":\"bool\",\"internalType\":\"bool\"},{\"name\":\"startingAnchorRoot\",\"type\":\"tuple\",\"internalType\":\"structProposal\",\"components\":[{\"name\":\"root\",\"type\":\"bytes32\",\"internalType\":\"Hash\"},{\"name\":\"l2SequenceNumber\",\"type\":\"uint256\",\"internalType\":\"uint256\"}]},{\"name\":\"gameParameters\",\"type\":\"tuple\",\"internalType\":\"structOPContractsManagerInteropMigrator.GameParameters\",\"components\":[{\"name\":\"proposer\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"challenger\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"maxGameDepth\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"splitDepth\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"initBond\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"clockExtension\",\"type\":\"uint64\",\"internalType\":\"Duration\"},{\"name\":\"maxClockDuration\",\"type\":\"uint64\",\"internalType\":\"Duration\"}]},{\"name\":\"opChainConfigs\",\"type\":\"tuple[]\",\"internalType\":\"structOPContractsManager.OpChainConfig[]\",\"components\":[{\"name\":\"systemConfigProxy\",\"type\":\"address\",\"internalType\":\"contractISystemConfig\"},{\"name\":\"cannonPrestate\",\"type\":\"bytes32\",\"internalType\":\"Claim\"},{\"name\":\"cannonKonaPrestate\",\"type\":\"bytes32\",\"internalType\":\"Claim\"}]}]}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"opcmDeployer\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"contractOPContractsManagerDeployer\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"opcmGameTypeAdder\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"contractOPContractsManagerGameTypeAdder\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"opcmInteropMigrator\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"contractOPContractsManagerInteropMigrator\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"opcmStandardValidator\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"contractOPContractsManagerStandardValidator\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"opcmUpgrader\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"contractOPContractsManagerUpgrader\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"protocolVersions\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"contractIProtocolVersions\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"superchainConfig\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"contractISuperchainConfig\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"updatePrestate\",\"inputs\":[{\"name\":\"_prestateUpdateInputs\",\"type\":\"tuple[]\",\"internalType\":\"structOPContractsManager.UpdatePrestateInput[]\",\"components\":[{\"name\":\"systemConfigProxy\",\"type\":\"address\",\"internalType\":\"contractISystemConfig\"},{\"name\":\"cannonPrestate\",\"type\":\"bytes32\",\"internalType\":\"Claim\"},{\"name\":\"cannonKonaPrestate\",\"type\":\"bytes32\",\"internalType\":\"Claim\"}]}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"upgrade\",\"inputs\":[{\"name\":\"_opChainConfigs\",\"type\":\"tuple[]\",\"internalType\":\"structOPContractsManager.OpChainConfig[]\",\"components\":[{\"name\":\"systemConfigProxy\",\"type\":\"address\",\"internalType\":\"contractISystemConfig\"},{\"name\":\"cannonPrestate\",\"type\":\"bytes32\",\"internalType\":\"Claim\"},{\"name\":\"cannonKonaPrestate\",\"type\":\"bytes32\",\"internalType\":\"Claim\"}]}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"upgradeSuperchainConfig\",\"inputs\":[{\"name\":\"_superchainConfig\",\"type\":\"address\",\"internalType\":\"contractISuperchainConfig\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"validate\",\"inputs\":[{\"name\":\"_input\",\"type\":\"tuple\",\"internalType\":\"structOPContractsManagerStandardValidator.ValidationInput\",\"components\":[{\"name\":\"sysCfg\",\"type\":\"address\",\"internalType\":\"contractISystemConfig\"},{\"name\":\"absolutePrestate\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"l2ChainID\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"proposer\",\"type\":\"address\",\"internalType\":\"address\"}]},{\"name\":\"_allowFailure\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"outputs\":[{\"name\":\"\",\"type\":\"string\",\"internalType\":\"string\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"validate\",\"inputs\":[{\"name\":\"_input\",\"type\":\"tuple\",\"internalType\":\"structOPContractsManagerStandardValidator.ValidationInputDev\",\"components\":[{\"name\":\"sysCfg\",\"type\":\"address\",\"internalType\":\"contractISystemConfig\"},{\"name\":\"cannonPrestate\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"cannonKonaPrestate\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"l2ChainID\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"proposer\",\"type\":\"address\",\"internalType\":\"address\"}]},{\"name\":\"_allowFailure\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"outputs\":[{\"name\":\"\",\"type\":\"string\",\"internalType\":\"string\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"validateWithOverrides\",\"inputs\":[{\"name\":\"_input\",\"type\":\"tuple\",\"internalType\":\"structOPContractsManagerStandardValidator.ValidationInput\",\"components\":[{\"name\":\"sysCfg\",\"type\":\"address\",\"internalType\":\"contractISystemConfig\"},{\"name\":\"absolutePrestate\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"l2ChainID\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"proposer\",\"type\":\"address\",\"internalType\":\"address\"}]},{\"name\":\"_allowFailure\",\"type\":\"bool\",\"internalType\":\"bool\"},{\"name\":\"_overrides\",\"type\":\"tuple\",\"internalType\":\"structOPContractsManagerStandardValidator.ValidationOverrides\",\"components\":[{\"name\":\"l1PAOMultisig\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"challenger\",\"type\":\"address\",\"internalType\":\"address\"}]}],\"outputs\":[{\"name\":\"\",\"type\":\"string\",\"internalType\":\"string\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"validateWithOverrides\",\"inputs\":[{\"name\":\"_input\",\"type\":\"tuple\",\"internalType\":\"structOPContractsManagerStandardValidator.ValidationInputDev\",\"components\":[{\"name\":\"sysCfg\",\"type\":\"address\",\"internalType\":\"contractISystemConfig\"},{\"name\":\"cannonPrestate\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"cannonKonaPrestate\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"l2ChainID\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"proposer\",\"type\":\"address\",\"internalType\":\"address\"}]},{\"name\":\"_allowFailure\",\"type\":\"bool\",\"internalType\":\"bool\"},{\"name\":\"_overrides\",\"type\":\"tuple\",\"internalType\":\"structOPContractsManagerStandardValidator.ValidationOverrides\",\"components\":[{\"name\":\"l1PAOMultisig\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"challenger\",\"type\":\"address\",\"internalType\":\"address\"}]}],\"outputs\":[{\"name\":\"\",\"type\":\"string\",\"internalType\":\"string\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"version\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"string\",\"internalType\":\"string\"}],\"stateMutability\":\"pure\"},{\"type\":\"error\",\"name\":\"AddressHasNoCode\",\"inputs\":[{\"name\":\"who\",\"type\":\"address\",\"internalType\":\"address\"}]},{\"type\":\"error\",\"name\":\"AddressNotFound\",\"inputs\":[{\"name\":\"who\",\"type\":\"address\",\"internalType\":\"address\"}]},{\"type\":\"error\",\"name\":\"AlreadyReleased\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"InvalidChainId\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"InvalidDevFeatureAccess\",\"inputs\":[{\"name\":\"devFeature\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}]},{\"type\":\"error\",\"name\":\"InvalidGameConfigs\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"InvalidRoleAddress\",\"inputs\":[{\"name\":\"role\",\"type\":\"string\",\"internalType\":\"string\"}]},{\"type\":\"error\",\"name\":\"InvalidStartingAnchorRoot\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"LatestReleaseNotSet\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"OPContractsManager_V2Enabled\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"OnlyDelegatecall\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"PrestateNotSet\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"PrestateRequired\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"SuperchainConfigMismatch\",\"inputs\":[{\"name\":\"systemConfig\",\"type\":\"address\",\"internalType\":\"contractISystemConfig\"}]},{\"type\":\"error\",\"name\":\"SuperchainProxyAdminMismatch\",\"inputs\":[]}]",
	Bin: "0x6101806040523480156200001257600080fd5b5060405162002e0a38038062002e0a833981016040819052620000359162000306565b60405163b6a4cd2160e01b81526001600160a01b03838116600483015287169063b6a4cd219060240160006040518083038186803b1580156200007757600080fd5b505afa1580156200008c573d6000803e3d6000fd5b505060405163b6a4cd2160e01b81526001600160a01b0384811660048301528916925063b6a4cd21915060240160006040518083038186803b158015620000d257600080fd5b505afa158015620000e7573d6000803e3d6000fd5b505060405163b6a4cd2160e01b81526001600160a01b038a811660048301528916925063b6a4cd21915060240160006040518083038186803b1580156200012d57600080fd5b505afa15801562000142573d6000803e3d6000fd5b505060405163b6a4cd2160e01b81526001600160a01b03891660048201819052925063b6a4cd21915060240160006040518083038186803b1580156200018757600080fd5b505afa1580156200019c573d6000803e3d6000fd5b505060405163b6a4cd2160e01b81526001600160a01b0388811660048301528916925063b6a4cd21915060240160006040518083038186803b158015620001e257600080fd5b505afa158015620001f7573d6000803e3d6000fd5b505060405163b6a4cd2160e01b81526001600160a01b0387811660048301528916925063b6a4cd21915060240160006040518083038186803b1580156200023d57600080fd5b505afa15801562000252573d6000803e3d6000fd5b505060405163b6a4cd2160e01b81526001600160a01b0386811660048301528916925063b6a4cd21915060240160006040518083038186803b1580156200029857600080fd5b505afa158015620002ad573d6000803e3d6000fd5b5050506001600160a01b039788166080525094861660a05292851660c05290841660e05283166101005282166101205216610140523061016052620003b1565b6001600160a01b03811681146200030357600080fd5b50565b600080600080600080600060e0888a0312156200032257600080fd5b87516200032f81620002ed565b60208901519097506200034281620002ed565b60408901519096506200035581620002ed565b60608901519095506200036881620002ed565b60808901519094506200037b81620002ed565b60a08901519093506200038e81620002ed565b60c0890151909250620003a181620002ed565b8091505092959891949750929550565b60805160a05160c05160e05161010051610120516101405161016051612967620004a3600039600081816108610152818161096901528181610ce101528181610e8f0152610f950152600061032f0152600081816102600152610b50015260008181610416015281816104cb015281816107cc01528181610c9601526110b70152600081816101fb015261092b01526000818161019701528181610f5e015261105f015260008181610308015281816105550152818161066d0152818161072001528181610b2101528181610bf00152610dfd01526000818161043d01528181610a350152610dab01526129676000f3fe608060405234801561001057600080fd5b506004361061018d5760003560e01c8063622d56f1116100e3578063b51f9c2b1161008c578063c993f27c11610066578063c993f27c1461045f578063cbeda5a714610472578063f3edcbe11461048557600080fd5b8063b51f9c2b146103ba578063ba7903db14610411578063becbdf4a1461043857600080fd5b806378ecabce116100bd57806378ecabce146103715780638970ac4414610394578063b23cc044146103a757600080fd5b8063622d56f1146103035780636624856a1461032a5780636ab5f6611461035157600080fd5b8063318b1b801161014557806354fd4d501161011f57806354fd4d501461029557806358084273146102ce578063604aa628146102e357600080fd5b8063318b1b801461024857806335e80ab31461025b57806341fe53851461028257600080fd5b80631481a724116101765780631481a724146101f65780631d8a4e921461021d57806330e9012c1461023357600080fd5b806303dbe68c146101925780630e9d5cb9146101d6575b600080fd5b6101b97f000000000000000000000000000000000000000000000000000000000000000081565b6040516001600160a01b0390911681526020015b60405180910390f35b6101e96101e4366004611393565b610498565b6040516101cd9190611436565b6101b97f000000000000000000000000000000000000000000000000000000000000000081565b610225610551565b6040519081526020016101cd565b61023b6105da565b6040516101cd9190611449565b6101b96102563660046115b1565b6106ee565b6101b97f000000000000000000000000000000000000000000000000000000000000000081565b6101e96102903660046115ca565b610799565b60408051808201909152600581527f362e312e3000000000000000000000000000000000000000000000000000000060208201526101e9565b6102e16102dc366004611602565b61084f565b005b6102f66102f1366004611715565b610955565b6040516101cd91906118aa565b6101b97f000000000000000000000000000000000000000000000000000000000000000081565b6101b97f000000000000000000000000000000000000000000000000000000000000000081565b61036461035f366004611906565b610a70565b6040516101cd9190611942565b61038461037f3660046115b1565b610bbe565b60405190151581526020016101cd565b6101e96103a2366004611b02565b610c63565b6102e16103b5366004611be6565b610ccf565b6103c2610dd0565b6040516101cd919081516001600160a01b039081168252602080840151821690830152604080840151821690830152606080840151821690830152608092830151169181019190915260a00190565b6101b97f000000000000000000000000000000000000000000000000000000000000000081565b6101b97f000000000000000000000000000000000000000000000000000000000000000081565b6102e161046d366004611c2f565b610e7d565b6102e1610480366004611be6565b610f83565b6101e9610493366004611c4c565b611084565b6040517f0e9d5cb90000000000000000000000000000000000000000000000000000000081526060906001600160a01b037f00000000000000000000000000000000000000000000000000000000000000001690630e9d5cb99061050490879087908790600401611c79565b600060405180830381865afa158015610521573d6000803e3d6000fd5b505050506040513d6000823e601f3d908101601f191682016040526105499190810190611cdb565b949350505050565b60007f00000000000000000000000000000000000000000000000000000000000000006001600160a01b0316631d8a4e926040518163ffffffff1660e01b8152600401602060405180830381865afa1580156105b1573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906105d59190611d52565b905090565b6040805161024081018252600080825260208201819052918101829052606081018290526080810182905260a0810182905260c0810182905260e08101829052610100810182905261012081018290526101408101829052610160810182905261018081018290526101a081018290526101c081018290526101e0810182905261020081018290526102208101919091527f00000000000000000000000000000000000000000000000000000000000000006001600160a01b03166330e9012c6040518163ffffffff1660e01b815260040161024060405180830381865afa1580156106ca573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906105d59190611d76565b6040517f318b1b80000000000000000000000000000000000000000000000000000000008152600481018290526000907f00000000000000000000000000000000000000000000000000000000000000006001600160a01b03169063318b1b8090602401602060405180830381865afa15801561076f573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906107939190611ece565b92915050565b6040517f41fe53850000000000000000000000000000000000000000000000000000000081526060906001600160a01b037f000000000000000000000000000000000000000000000000000000000000000016906341fe5385906108039086908690600401611eeb565b600060405180830381865afa158015610820573d6000803e3d6000fd5b505050506040513d6000823e601f3d908101601f191682016040526108489190810190611cdb565b9392505050565b6108576110ee565b6001600160a01b037f00000000000000000000000000000000000000000000000000000000000000001630036108b9576040517f0a57d61d00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b6000816040516024016108cc9190611ffd565b60408051601f198184030181529190526020810180517bffffffffffffffffffffffffffffffffffffffffffffffffffffffff167f580842730000000000000000000000000000000000000000000000000000000017905290506109507f000000000000000000000000000000000000000000000000000000000000000082611133565b505050565b606061095f6110ee565b6001600160a01b037f00000000000000000000000000000000000000000000000000000000000000001630036109c1576040517f0a57d61d00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b6000826040516024016109d491906120f0565b60408051601f198184030181529190526020810180517bffffffffffffffffffffffffffffffffffffffffffffffffffffffff167f604aa6280000000000000000000000000000000000000000000000000000000017905290506000610a5a7f000000000000000000000000000000000000000000000000000000000000000083611133565b9050808060200190518101906105499190612225565b604080516101e081018252600080825260208201819052918101829052606081018290526080810182905260a0810182905260c0810182905260e08101829052610100810182905261012081018290526101408101829052610160810182905261018081018290526101a081018290526101c0810191909152610af16110ee565b6040517fcefe12230000000000000000000000000000000000000000000000000000000081526001600160a01b037f0000000000000000000000000000000000000000000000000000000000000000169063cefe122390610b7a9085907f000000000000000000000000000000000000000000000000000000000000000090339060040161240e565b6101e0604051808303816000875af1158015610b9a573d6000803e3d6000fd5b505050506040513d601f19601f8201168201806040525081019061079391906125e2565b6040517f78ecabce000000000000000000000000000000000000000000000000000000008152600481018290526000907f00000000000000000000000000000000000000000000000000000000000000006001600160a01b0316906378ecabce90602401602060405180830381865afa158015610c3f573d6000803e3d6000fd5b505050506040513d601f19601f8201168201806040525081019061079391906126f9565b6040517f8970ac440000000000000000000000000000000000000000000000000000000081526060906001600160a01b037f00000000000000000000000000000000000000000000000000000000000000001690638970ac449061050490879087908790600401612716565b610cd76110ee565b6001600160a01b037f0000000000000000000000000000000000000000000000000000000000000000163003610d39576040517f0a57d61d00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b600081604051602401610d4c9190612787565b60408051601f198184030181529190526020810180517bffffffffffffffffffffffffffffffffffffffffffffffffffffffff167fb23cc0440000000000000000000000000000000000000000000000000000000017905290506109507f000000000000000000000000000000000000000000000000000000000000000082611133565b6040805160a0810182526000808252602082018190529181018290526060810182905260808101919091527f00000000000000000000000000000000000000000000000000000000000000006001600160a01b031663b51f9c2b6040518163ffffffff1660e01b815260040160a060405180830381865afa158015610e59573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906105d591906127f2565b610e856110ee565b6001600160a01b037f0000000000000000000000000000000000000000000000000000000000000000163003610ee7576040517f0a57d61d00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b6040516001600160a01b038216602482015260009060440160408051601f198184030181529190526020810180517bffffffffffffffffffffffffffffffffffffffffffffffffffffffff167fc993f27c0000000000000000000000000000000000000000000000000000000017905290506109507f000000000000000000000000000000000000000000000000000000000000000082611133565b610f8b6110ee565b6001600160a01b037f0000000000000000000000000000000000000000000000000000000000000000163003610fed576040517f0a57d61d00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b600081604051602401611000919061288a565b60408051601f198184030181529190526020810180517bffffffffffffffffffffffffffffffffffffffffffffffffffffffff167fcbeda5a70000000000000000000000000000000000000000000000000000000017905290506109507f000000000000000000000000000000000000000000000000000000000000000082611133565b6040517ff3edcbe10000000000000000000000000000000000000000000000000000000081526060906001600160a01b037f0000000000000000000000000000000000000000000000000000000000000000169063f3edcbe19061080390869086906004016128e9565b6110fa62010000610bbe565b15611131576040517fe232d67700000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b565b6060600080846001600160a01b031684604051611150919061293e565b600060405180830381855af49150503d806000811461118b576040519150601f19603f3d011682016040523d82523d6000602084013e611190565b606091505b50915091508161054957805160208201fd5b7f4e487b7100000000000000000000000000000000000000000000000000000000600052604160045260246000fd5b6040805190810167ffffffffffffffff811182821017156111f4576111f46111a2565b60405290565b604051610180810167ffffffffffffffff811182821017156111f4576111f46111a2565b604051610240810167ffffffffffffffff811182821017156111f4576111f46111a2565b6040516101e0810167ffffffffffffffff811182821017156111f4576111f46111a2565b604051601f8201601f1916810167ffffffffffffffff8111828210171561128f5761128f6111a2565b604052919050565b6001600160a01b03811681146112ac57600080fd5b50565b80356112ba81611297565b919050565b6000608082840312156112d157600080fd5b6040516080810181811067ffffffffffffffff821117156112f4576112f46111a2565b604052905080823561130581611297565b808252506020830135602082015260408301356040820152606083013561132b81611297565b6060919091015292915050565b80151581146112ac57600080fd5b80356112ba81611338565b60006040828403121561136357600080fd5b61136b6111d1565b9050813561137881611297565b8152602082013561138881611297565b602082015292915050565b600080600060e084860312156113a857600080fd5b6113b285856112bf565b925060808401356113c281611338565b91506113d18560a08601611351565b90509250925092565b60005b838110156113f55781810151838201526020016113dd565b83811115611404576000848401525b50505050565b600081518084526114228160208601602086016113da565b601f01601f19169290920160200192915050565b602081526000610848602083018461140a565b81516001600160a01b031681526102408101602083015161147560208401826001600160a01b03169052565b50604083015161149060408401826001600160a01b03169052565b5060608301516114ab60608401826001600160a01b03169052565b5060808301516114c660808401826001600160a01b03169052565b5060a08301516114e160a08401826001600160a01b03169052565b5060c08301516114fc60c08401826001600160a01b03169052565b5060e083015161151760e08401826001600160a01b03169052565b50610100838101516001600160a01b0390811691840191909152610120808501518216908401526101408085015182169084015261016080850151821690840152610180808501518216908401526101a0808501518216908401526101c0808501518216908401526101e080850151821690840152610200808501518216908401526102208085015191821681850152905b505092915050565b6000602082840312156115c357600080fd5b5035919050565b60008060a083850312156115dd57600080fd5b6115e784846112bf565b915060808301356115f781611338565b809150509250929050565b60006020828403121561161457600080fd5b813567ffffffffffffffff81111561162b57600080fd5b8201610160818503121561084857600080fd5b600067ffffffffffffffff821115611658576116586111a2565b5060051b60200190565b600067ffffffffffffffff82111561167c5761167c6111a2565b50601f01601f191660200190565b600082601f83011261169b57600080fd5b81356116ae6116a982611662565b611266565b8181528460208386010111156116c357600080fd5b816020850160208301376000918101602001919091529392505050565b803563ffffffff811681146112ba57600080fd5b67ffffffffffffffff811681146112ac57600080fd5b80356112ba816116f4565b6000602080838503121561172857600080fd5b823567ffffffffffffffff8082111561174057600080fd5b818501915085601f83011261175457600080fd5b81356117626116a98261163e565b81815260059190911b8301840190848101908883111561178157600080fd5b8585015b8381101561189d5780358581111561179d5760008081fd5b8601610180818c03601f19018113156117b65760008081fd5b6117be6111fa565b89830135888111156117d05760008081fd5b6117de8e8c8387010161168a565b82525060406117ee8185016112af565b8b83015260606117ff8186016112af565b82840152608091506118128286016116e0565b818401525060a0808501358284015260c0915081850135818401525060e08085013582840152610100915061184882860161170a565b9083015261012061185a85820161170a565b82840152610140915081850135818401525061016061187a8186016112af565b82840152611889848601611346565b908301525085525050918601918601611785565b5098975050505050505050565b602080825282518282018190526000919060409081850190868401855b828110156118f957815180516001600160a01b03908116865290870151168685015292840192908501906001016118c7565b5091979650505050505050565b60006020828403121561191857600080fd5b813567ffffffffffffffff81111561192f57600080fd5b8201610260818503121561084857600080fd5b81516001600160a01b031681526101e08101602083015161196e60208401826001600160a01b03169052565b50604083015161198960408401826001600160a01b03169052565b5060608301516119a460608401826001600160a01b03169052565b5060808301516119bf60808401826001600160a01b03169052565b5060a08301516119da60a08401826001600160a01b03169052565b5060c08301516119f560c08401826001600160a01b03169052565b5060e0830151611a1060e08401826001600160a01b03169052565b50610100838101516001600160a01b0390811691840191909152610120808501518216908401526101408085015182169084015261016080850151821690840152610180808501518216908401526101a0808501518216908401526101c08085015191821681850152906115a9565b600060a08284031215611a9157600080fd5b60405160a0810181811067ffffffffffffffff82111715611ab457611ab46111a2565b6040529050808235611ac581611297565b808252506020830135602082015260408301356040820152606083013560608201526080830135611af581611297565b6080919091015292915050565b60008060006101008486031215611b1857600080fd5b611b228585611a7f565b925060a0840135611b3281611338565b91506113d18560c08601611351565b6000611b4f6116a98461163e565b83815290506020808201906060808602850187811115611b6e57600080fd5b855b81811015611bda5782818a031215611b885760008081fd5b6040805184810181811067ffffffffffffffff82111715611bab57611bab6111a2565b82528235611bb881611297565b8152828601358682015281830135918101919091528552938301938201611b70565b50505050509392505050565b600060208284031215611bf857600080fd5b813567ffffffffffffffff811115611c0f57600080fd5b8201601f81018413611c2057600080fd5b61054984823560208401611b41565b600060208284031215611c4157600080fd5b813561084881611297565b60008060c08385031215611c5f57600080fd5b611c698484611a7f565b915060a08301356115f781611338565b60e08101611cb1828680516001600160a01b039081168352602080830151908401526040808301519084015260609182015116910152565b831515608083015282516001600160a01b0390811660a084015260208401511660c0830152610549565b600060208284031215611ced57600080fd5b815167ffffffffffffffff811115611d0457600080fd5b8201601f81018413611d1557600080fd5b8051611d236116a982611662565b818152856020838501011115611d3857600080fd5b611d498260208301602086016113da565b95945050505050565b600060208284031215611d6457600080fd5b5051919050565b80516112ba81611297565b60006102408284031215611d8957600080fd5b611d9161121e565b611d9a83611d6b565b8152611da860208401611d6b565b6020820152611db960408401611d6b565b6040820152611dca60608401611d6b565b6060820152611ddb60808401611d6b565b6080820152611dec60a08401611d6b565b60a0820152611dfd60c08401611d6b565b60c0820152611e0e60e08401611d6b565b60e0820152610100611e21818501611d6b565b90820152610120611e33848201611d6b565b90820152610140611e45848201611d6b565b90820152610160611e57848201611d6b565b90820152610180611e69848201611d6b565b908201526101a0611e7b848201611d6b565b908201526101c0611e8d848201611d6b565b908201526101e0611e9f848201611d6b565b90820152610200611eb1848201611d6b565b90820152610220611ec3848201611d6b565b908201529392505050565b600060208284031215611ee057600080fd5b815161084881611297565b60a08101611f23828580516001600160a01b039081168352602080830151908401526040808301519084015260609182015116910152565b82151560808301529392505050565b60008083357fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe1843603018112611f6757600080fd5b830160208101925035905067ffffffffffffffff811115611f8757600080fd5b606081023603821315611f9957600080fd5b9250929050565b8183526000602080850194508260005b85811015611ff2578135611fc381611297565b6001600160a01b0316875281830135838801526040808301359088015260609687019690910190600101611fb0565b509495945050505050565b602081526000823561200e81611338565b8015156020840152506020830135604083015260408301356060830152606083013561203981611297565b6001600160a01b0380821660808501526080850135915061205982611297565b80821660a0850152505060a083013560c083015260c083013560e083015261010060e084013581840152808401359050612092816116f4565b61012067ffffffffffffffff8216818501526120af81860161170a565b9150506101406120ca8185018367ffffffffffffffff169052565b6120d681860186611f32565b6101608681015292509050611d4961018085018383611fa0565b60006020808301818452808551808352604092508286019150828160051b87010184880160005b83811015612217577fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffc08984030185528151610180815181865261215c8287018261140a565b915050888201516121778a8701826001600160a01b03169052565b50878201516001600160a01b03908116868a015260608084015163ffffffff16908701526080808401519087015260a0808401519087015260c0808401519087015260e08084015167ffffffffffffffff9081169188019190915261010080850151909116908701526101208084015190870152610140808401519091169086015261016091820151151591909401529386019390860190600101612117565b509098975050505050505050565b6000602080838503121561223857600080fd5b825167ffffffffffffffff81111561224f57600080fd5b8301601f8101851361226057600080fd5b805161226e6116a98261163e565b81815260069190911b8201830190838101908783111561228d57600080fd5b928401925b828410156122e357604084890312156122ab5760008081fd5b6122b36111d1565b84516122be81611297565b8152848601516122cd81611297565b8187015282526040939093019290840190612292565b979650505050505050565b80356122f981611297565b6001600160a01b03908116835260208201359061231582611297565b908116602084015260408201359061232c82611297565b908116604084015260608201359061234382611297565b908116606084015260808201359061235a82611297565b908116608084015260a08201359061237182611297565b80821660a085015250505050565b60008083357fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe18436030181126123b457600080fd5b830160208101925035905067ffffffffffffffff8111156123d457600080fd5b803603821315611f9957600080fd5b818352818160208501375060006020828401015260006020601f19601f840116840101905092915050565b6060815261241f60608201856122ee565b600061242d60c086016116e0565b6101206124418185018363ffffffff169052565b61244d60e088016116e0565b91506101406124638186018463ffffffff169052565b61016092506101008801358386015261247e8289018961237f565b9250610260610180818189015261249a6102c0890186856123e3565b94506124a8848c018c61237f565b945092506101a07fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffa089870301818a01526124e38686866123e3565b95506124f0878d0161170a565b96506101c0945061250c858a018867ffffffffffffffff169052565b612517828d016116e0565b96506101e0935061252f848a018863ffffffff169052565b6102009650808c0135878a01525050610220838b0135818901526102409350828b013584890152612561868c0161170a565b67ffffffffffffffff811689840152955061257d818c0161170a565b955050505061259961028086018467ffffffffffffffff169052565b6125a4818901611346565b9250506125b66102a085018315159052565b6001600160a01b038616602085015291506125ce9050565b6001600160a01b0383166040830152610549565b60006101e082840312156125f557600080fd5b6125fd611242565b61260683611d6b565b815261261460208401611d6b565b602082015261262560408401611d6b565b604082015261263660608401611d6b565b606082015261264760808401611d6b565b608082015261265860a08401611d6b565b60a082015261266960c08401611d6b565b60c082015261267a60e08401611d6b565b60e082015261010061268d818501611d6b565b9082015261012061269f848201611d6b565b908201526101406126b1848201611d6b565b908201526101606126c3848201611d6b565b908201526101806126d5848201611d6b565b908201526101a06126e7848201611d6b565b908201526101c0611ec3848201611d6b565b60006020828403121561270b57600080fd5b815161084881611338565b610100810161275d82866001600160a01b03808251168352602082015160208401526040820151604084015260608201516060840152806080830151166080840152505050565b83151560a083015282516001600160a01b0390811660c084015260208401511660e0830152610549565b6020808252825182820181905260009190848201906040850190845b818110156127e6576127d383855180516001600160a01b0316825260208082015190830152604090810151910152565b92840192606092909201916001016127a3565b50909695505050505050565b600060a0828403121561280457600080fd5b60405160a0810181811067ffffffffffffffff82111715612827576128276111a2565b604052825161283581611297565b8152602083015161284581611297565b6020820152604083015161285881611297565b6040820152606083015161286b81611297565b6060820152608083015161287e81611297565b60808201529392505050565b6020808252825182820181905260009190848201906040850190845b818110156127e6576128d683855180516001600160a01b0316825260208082015190830152604090810151910152565b92840192606092909201916001016128a6565b60c0810161292f82856001600160a01b03808251168352602082015160208401526040820151604084015260608201516060840152806080830151166080840152505050565b82151560a08301529392505050565b600082516129508184602087016113da565b919091019291505056fea164736f6c634300080f000a",
}

// OPContractsManagerABI is the input ABI used to generate the binding from.
// Deprecated: Use OPContractsManagerMetaData.ABI instead.
var OPContractsManagerABI = OPContractsManagerMetaData.ABI

// OPContractsManagerBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use OPContractsManagerMetaData.Bin instead.
var OPContractsManagerBin = OPContractsManagerMetaData.Bin

// DeployOPContractsManager deploys a new Ethereum contract, binding an instance of OPContractsManager to it.
func DeployOPContractsManager(auth *bind.TransactOpts, backend bind.ContractBackend, _opcmGameTypeAdder common.Address, _opcmDeployer common.Address, _opcmUpgrader common.Address, _opcmInteropMigrator common.Address, _opcmStandardValidator common.Address, _superchainConfig common.Address, _protocolVersions common.Address) (common.Address, *types.Transaction, *OPContractsManager, error) {
	parsed, err := OPContractsManagerMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(OPContractsManagerBin), backend, _opcmGameTypeAdder, _opcmDeployer, _opcmUpgrader, _opcmInteropMigrator, _opcmStandardValidator, _superchainConfig, _protocolVersions)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &OPContractsManager{OPContractsManagerCaller: OPContractsManagerCaller{contract: contract}, OPContractsManagerTransactor: OPContractsManagerTransactor{contract: contract}, OPContractsManagerFilterer: OPContractsManagerFilterer{contract: contract}}, nil
}

// OPContractsManager is an auto generated Go binding around an Ethereum contract.
type OPContractsManager struct {
	OPContractsManagerCaller     // Read-only binding to the contract
	OPContractsManagerTransactor // Write-only binding to the contract
	OPContractsManagerFilterer   // Log filterer for contract events
}

// OPContractsManagerCaller is an auto generated read-only Go binding around an Ethereum contract.
type OPContractsManagerCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// OPContractsManagerTransactor is an auto generated write-only Go binding around an Ethereum contract.
type OPContractsManagerTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// OPContractsManagerFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type OPContractsManagerFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// OPContractsManagerSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type OPContractsManagerSession struct {
	Contract     *OPContractsManager // Generic contract binding to set the session for
	CallOpts     bind.CallOpts       // Call options to use throughout this session
	TransactOpts bind.TransactOpts   // Transaction auth options to use throughout this session
}

// OPContractsManagerCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type OPContractsManagerCallerSession struct {
	Contract *OPContractsManagerCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts             // Call options to use throughout this session
}

// OPContractsManagerTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type OPContractsManagerTransactorSession struct {
	Contract     *OPContractsManagerTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts             // Transaction auth options to use throughout this session
}

// OPContractsManagerRaw is an auto generated low-level Go binding around an Ethereum contract.
type OPContractsManagerRaw struct {
	Contract *OPContractsManager // Generic contract binding to access the raw methods on
}

// OPContractsManagerCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type OPContractsManagerCallerRaw struct {
	Contract *OPContractsManagerCaller // Generic read-only contract binding to access the raw methods on
}

// OPContractsManagerTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type OPContractsManagerTransactorRaw struct {
	Contract *OPContractsManagerTransactor // Generic write-only contract binding to access the raw methods on
}

// NewOPContractsManager creates a new instance of OPContractsManager, bound to a specific deployed contract.
func NewOPContractsManager(address common.Address, backend bind.ContractBackend) (*OPContractsManager, error) {
	contract, err := bindOPContractsManager(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &OPContractsManager{OPContractsManagerCaller: OPContractsManagerCaller{contract: contract}, OPContractsManagerTransactor: OPContractsManagerTransactor{contract: contract}, OPContractsManagerFilterer: OPContractsManagerFilterer{contract: contract}}, nil
}

// NewOPContractsManagerCaller creates a new read-only instance of OPContractsManager, bound to a specific deployed contract.
func NewOPContractsManagerCaller(address common.Address, caller bind.ContractCaller) (*OPContractsManagerCaller, error) {
	contract, err := bindOPContractsManager(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &OPContractsManagerCaller{contract: contract}, nil
}

// NewOPContractsManagerTransactor creates a new write-only instance of OPContractsManager, bound to a specific deployed contract.
func NewOPContractsManagerTransactor(address common.Address, transactor bind.ContractTransactor) (*OPContractsManagerTransactor, error) {
	contract, err := bindOPContractsManager(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &OPContractsManagerTransactor{contract: contract}, nil
}

// NewOPContractsManagerFilterer creates a new log filterer instance of OPContractsManager, bound to a specific deployed contract.
func NewOPContractsManagerFilterer(address common.Address, filterer bind.ContractFilterer) (*OPContractsManagerFilterer, error) {
	contract, err := bindOPContractsManager(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &OPContractsManagerFilterer{contract: contract}, nil
}

// bindOPContractsManager binds a generic wrapper to an already deployed contract.
func bindOPContractsManager(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := OPContractsManagerMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_OPContractsManager *OPContractsManagerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _OPContractsManager.Contract.OPContractsManagerCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_OPContractsManager *OPContractsManagerRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _OPContractsManager.Contract.OPContractsManagerTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_OPContractsManager *OPContractsManagerRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _OPContractsManager.Contract.OPContractsManagerTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_OPContractsManager *OPContractsManagerCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _OPContractsManager.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_OPContractsManager *OPContractsManagerTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _OPContractsManager.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_OPContractsManager *OPContractsManagerTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _OPContractsManager.Contract.contract.Transact(opts, method, params...)
}

// Blueprints is a free data retrieval call binding the contract method 0xb51f9c2b.
//
// Solidity: function blueprints() view returns((address,address,address,address,address))
func (_OPContractsManager *OPContractsManagerCaller) Blueprints(opts *bind.CallOpts) (OPContractsManagerBlueprints, error) {
	var out []interface{}
	err := _OPContractsManager.contract.Call(opts, &out, "blueprints")

	if err != nil {
		return *new(OPContractsManagerBlueprints), err
	}

	out0 := *abi.ConvertType(out[0], new(OPContractsManagerBlueprints)).(*OPContractsManagerBlueprints)

	return out0, err

}

// Blueprints is a free data retrieval call binding the contract method 0xb51f9c2b.
//
// Solidity: function blueprints() view returns((address,address,address,address,address))
func (_OPContractsManager *OPContractsManagerSession) Blueprints() (OPContractsManagerBlueprints, error) {
	return _OPContractsManager.Contract.Blueprints(&_OPContractsManager.CallOpts)
}

// Blueprints is a free data retrieval call binding the contract method 0xb51f9c2b.
//
// Solidity: function blueprints() view returns((address,address,address,address,address))
func (_OPContractsManager *OPContractsManagerCallerSession) Blueprints() (OPContractsManagerBlueprints, error) {
	return _OPContractsManager.Contract.Blueprints(&_OPContractsManager.CallOpts)
}

// ChainIdToBatchInboxAddress is a free data retrieval call binding the contract method 0x318b1b80.
//
// Solidity: function chainIdToBatchInboxAddress(uint256 _l2ChainId) view returns(address)
func (_OPContractsManager *OPContractsManagerCaller) ChainIdToBatchInboxAddress(opts *bind.CallOpts, _l2ChainId *big.Int) (common.Address, error) {
	var out []interface{}
	err := _OPContractsManager.contract.Call(opts, &out, "chainIdToBatchInboxAddress", _l2ChainId)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// ChainIdToBatchInboxAddress is a free data retrieval call binding the contract method 0x318b1b80.
//
// Solidity: function chainIdToBatchInboxAddress(uint256 _l2ChainId) view returns(address)
func (_OPContractsManager *OPContractsManagerSession) ChainIdToBatchInboxAddress(_l2ChainId *big.Int) (common.Address, error) {
	return _OPContractsManager.Contract.ChainIdToBatchInboxAddress(&_OPContractsManager.CallOpts, _l2ChainId)
}

// ChainIdToBatchInboxAddress is a free data retrieval call binding the contract method 0x318b1b80.
//
// Solidity: function chainIdToBatchInboxAddress(uint256 _l2ChainId) view returns(address)
func (_OPContractsManager *OPContractsManagerCallerSession) ChainIdToBatchInboxAddress(_l2ChainId *big.Int) (common.Address, error) {
	return _OPContractsManager.Contract.ChainIdToBatchInboxAddress(&_OPContractsManager.CallOpts, _l2ChainId)
}

// DevFeatureBitmap is a free data retrieval call binding the contract method 0x1d8a4e92.
//
// Solidity: function devFeatureBitmap() view returns(bytes32)
func (_OPContractsManager *OPContractsManagerCaller) DevFeatureBitmap(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _OPContractsManager.contract.Call(opts, &out, "devFeatureBitmap")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// DevFeatureBitmap is a free data retrieval call binding the contract method 0x1d8a4e92.
//
// Solidity: function devFeatureBitmap() view returns(bytes32)
func (_OPContractsManager *OPContractsManagerSession) DevFeatureBitmap() ([32]byte, error) {
	return _OPContractsManager.Contract.DevFeatureBitmap(&_OPContractsManager.CallOpts)
}

// DevFeatureBitmap is a free data retrieval call binding the contract method 0x1d8a4e92.
//
// Solidity: function devFeatureBitmap() view returns(bytes32)
func (_OPContractsManager *OPContractsManagerCallerSession) DevFeatureBitmap() ([32]byte, error) {
	return _OPContractsManager.Contract.DevFeatureBitmap(&_OPContractsManager.CallOpts)
}

// Implementations is a free data retrieval call binding the contract method 0x30e9012c.
//
// Solidity: function implementations() view returns((address,address,address,address,address,address,address,address,address,address,address,address,address,address,address,address,address,address))
func (_OPContractsManager *OPContractsManagerCaller) Implementations(opts *bind.CallOpts) (OPContractsManagerImplementations, error) {
	var out []interface{}
	err := _OPContractsManager.contract.Call(opts, &out, "implementations")

	if err != nil {
		return *new(OPContractsManagerImplementations), err
	}

	out0 := *abi.ConvertType(out[0], new(OPContractsManagerImplementations)).(*OPContractsManagerImplementations)

	return out0, err

}

// Implementations is a free data retrieval call binding the contract method 0x30e9012c.
//
// Solidity: function implementations() view returns((address,address,address,address,address,address,address,address,address,address,address,address,address,address,address,address,address,address))
func (_OPContractsManager *OPContractsManagerSession) Implementations() (OPContractsManagerImplementations, error) {
	return _OPContractsManager.Contract.Implementations(&_OPContractsManager.CallOpts)
}

// Implementations is a free data retrieval call binding the contract method 0x30e9012c.
//
// Solidity: function implementations() view returns((address,address,address,address,address,address,address,address,address,address,address,address,address,address,address,address,address,address))
func (_OPContractsManager *OPContractsManagerCallerSession) Implementations() (OPContractsManagerImplementations, error) {
	return _OPContractsManager.Contract.Implementations(&_OPContractsManager.CallOpts)
}

// IsDevFeatureEnabled is a free data retrieval call binding the contract method 0x78ecabce.
//
// Solidity: function isDevFeatureEnabled(bytes32 _feature) view returns(bool)
func (_OPContractsManager *OPContractsManagerCaller) IsDevFeatureEnabled(opts *bind.CallOpts, _feature [32]byte) (bool, error) {
	var out []interface{}
	err := _OPContractsManager.contract.Call(opts, &out, "isDevFeatureEnabled", _feature)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsDevFeatureEnabled is a free data retrieval call binding the contract method 0x78ecabce.
//
// Solidity: function isDevFeatureEnabled(bytes32 _feature) view returns(bool)
func (_OPContractsManager *OPContractsManagerSession) IsDevFeatureEnabled(_feature [32]byte) (bool, error) {
	return _OPContractsManager.Contract.IsDevFeatureEnabled(&_OPContractsManager.CallOpts, _feature)
}

// IsDevFeatureEnabled is a free data retrieval call binding the contract method 0x78ecabce.
//
// Solidity: function isDevFeatureEnabled(bytes32 _feature) view returns(bool)
func (_OPContractsManager *OPContractsManagerCallerSession) IsDevFeatureEnabled(_feature [32]byte) (bool, error) {
	return _OPContractsManager.Contract.IsDevFeatureEnabled(&_OPContractsManager.CallOpts, _feature)
}

// OpcmDeployer is a free data retrieval call binding the contract method 0x622d56f1.
//
// Solidity: function opcmDeployer() view returns(address)
func (_OPContractsManager *OPContractsManagerCaller) OpcmDeployer(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _OPContractsManager.contract.Call(opts, &out, "opcmDeployer")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// OpcmDeployer is a free data retrieval call binding the contract method 0x622d56f1.
//
// Solidity: function opcmDeployer() view returns(address)
func (_OPContractsManager *OPContractsManagerSession) OpcmDeployer() (common.Address, error) {
	return _OPContractsManager.Contract.OpcmDeployer(&_OPContractsManager.CallOpts)
}

// OpcmDeployer is a free data retrieval call binding the contract method 0x622d56f1.
//
// Solidity: function opcmDeployer() view returns(address)
func (_OPContractsManager *OPContractsManagerCallerSession) OpcmDeployer() (common.Address, error) {
	return _OPContractsManager.Contract.OpcmDeployer(&_OPContractsManager.CallOpts)
}

// OpcmGameTypeAdder is a free data retrieval call binding the contract method 0xbecbdf4a.
//
// Solidity: function opcmGameTypeAdder() view returns(address)
func (_OPContractsManager *OPContractsManagerCaller) OpcmGameTypeAdder(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _OPContractsManager.contract.Call(opts, &out, "opcmGameTypeAdder")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// OpcmGameTypeAdder is a free data retrieval call binding the contract method 0xbecbdf4a.
//
// Solidity: function opcmGameTypeAdder() view returns(address)
func (_OPContractsManager *OPContractsManagerSession) OpcmGameTypeAdder() (common.Address, error) {
	return _OPContractsManager.Contract.OpcmGameTypeAdder(&_OPContractsManager.CallOpts)
}

// OpcmGameTypeAdder is a free data retrieval call binding the contract method 0xbecbdf4a.
//
// Solidity: function opcmGameTypeAdder() view returns(address)
func (_OPContractsManager *OPContractsManagerCallerSession) OpcmGameTypeAdder() (common.Address, error) {
	return _OPContractsManager.Contract.OpcmGameTypeAdder(&_OPContractsManager.CallOpts)
}

// OpcmInteropMigrator is a free data retrieval call binding the contract method 0x1481a724.
//
// Solidity: function opcmInteropMigrator() view returns(address)
func (_OPContractsManager *OPContractsManagerCaller) OpcmInteropMigrator(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _OPContractsManager.contract.Call(opts, &out, "opcmInteropMigrator")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// OpcmInteropMigrator is a free data retrieval call binding the contract method 0x1481a724.
//
// Solidity: function opcmInteropMigrator() view returns(address)
func (_OPContractsManager *OPContractsManagerSession) OpcmInteropMigrator() (common.Address, error) {
	return _OPContractsManager.Contract.OpcmInteropMigrator(&_OPContractsManager.CallOpts)
}

// OpcmInteropMigrator is a free data retrieval call binding the contract method 0x1481a724.
//
// Solidity: function opcmInteropMigrator() view returns(address)
func (_OPContractsManager *OPContractsManagerCallerSession) OpcmInteropMigrator() (common.Address, error) {
	return _OPContractsManager.Contract.OpcmInteropMigrator(&_OPContractsManager.CallOpts)
}

// OpcmStandardValidator is a free data retrieval call binding the contract method 0xba7903db.
//
// Solidity: function opcmStandardValidator() view returns(address)
func (_OPContractsManager *OPContractsManagerCaller) OpcmStandardValidator(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _OPContractsManager.contract.Call(opts, &out, "opcmStandardValidator")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// OpcmStandardValidator is a free data retrieval call binding the contract method 0xba7903db.
//
// Solidity: function opcmStandardValidator() view returns(address)
func (_OPContractsManager *OPContractsManagerSession) OpcmStandardValidator() (common.Address, error) {
	return _OPContractsManager.Contract.OpcmStandardValidator(&_OPContractsManager.CallOpts)
}

// OpcmStandardValidator is a free data retrieval call binding the contract method 0xba7903db.
//
// Solidity: function opcmStandardValidator() view returns(address)
func (_OPContractsManager *OPContractsManagerCallerSession) OpcmStandardValidator() (common.Address, error) {
	return _OPContractsManager.Contract.OpcmStandardValidator(&_OPContractsManager.CallOpts)
}

// OpcmUpgrader is a free data retrieval call binding the contract method 0x03dbe68c.
//
// Solidity: function opcmUpgrader() view returns(address)
func (_OPContractsManager *OPContractsManagerCaller) OpcmUpgrader(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _OPContractsManager.contract.Call(opts, &out, "opcmUpgrader")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// OpcmUpgrader is a free data retrieval call binding the contract method 0x03dbe68c.
//
// Solidity: function opcmUpgrader() view returns(address)
func (_OPContractsManager *OPContractsManagerSession) OpcmUpgrader() (common.Address, error) {
	return _OPContractsManager.Contract.OpcmUpgrader(&_OPContractsManager.CallOpts)
}

// OpcmUpgrader is a free data retrieval call binding the contract method 0x03dbe68c.
//
// Solidity: function opcmUpgrader() view returns(address)
func (_OPContractsManager *OPContractsManagerCallerSession) OpcmUpgrader() (common.Address, error) {
	return _OPContractsManager.Contract.OpcmUpgrader(&_OPContractsManager.CallOpts)
}

// ProtocolVersions is a free data retrieval call binding the contract method 0x6624856a.
//
// Solidity: function protocolVersions() view returns(address)
func (_OPContractsManager *OPContractsManagerCaller) ProtocolVersions(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _OPContractsManager.contract.Call(opts, &out, "protocolVersions")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// ProtocolVersions is a free data retrieval call binding the contract method 0x6624856a.
//
// Solidity: function protocolVersions() view returns(address)
func (_OPContractsManager *OPContractsManagerSession) ProtocolVersions() (common.Address, error) {
	return _OPContractsManager.Contract.ProtocolVersions(&_OPContractsManager.CallOpts)
}

// ProtocolVersions is a free data retrieval call binding the contract method 0x6624856a.
//
// Solidity: function protocolVersions() view returns(address)
func (_OPContractsManager *OPContractsManagerCallerSession) ProtocolVersions() (common.Address, error) {
	return _OPContractsManager.Contract.ProtocolVersions(&_OPContractsManager.CallOpts)
}

// SuperchainConfig is a free data retrieval call binding the contract method 0x35e80ab3.
//
// Solidity: function superchainConfig() view returns(address)
func (_OPContractsManager *OPContractsManagerCaller) SuperchainConfig(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _OPContractsManager.contract.Call(opts, &out, "superchainConfig")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// SuperchainConfig is a free data retrieval call binding the contract method 0x35e80ab3.
//
// Solidity: function superchainConfig() view returns(address)
func (_OPContractsManager *OPContractsManagerSession) SuperchainConfig() (common.Address, error) {
	return _OPContractsManager.Contract.SuperchainConfig(&_OPContractsManager.CallOpts)
}

// SuperchainConfig is a free data retrieval call binding the contract method 0x35e80ab3.
//
// Solidity: function superchainConfig() view returns(address)
func (_OPContractsManager *OPContractsManagerCallerSession) SuperchainConfig() (common.Address, error) {
	return _OPContractsManager.Contract.SuperchainConfig(&_OPContractsManager.CallOpts)
}

// Validate is a free data retrieval call binding the contract method 0x41fe5385.
//
// Solidity: function validate((address,bytes32,uint256,address) _input, bool _allowFailure) view returns(string)
func (_OPContractsManager *OPContractsManagerCaller) Validate(opts *bind.CallOpts, _input OPContractsManagerStandardValidatorValidationInput, _allowFailure bool) (string, error) {
	var out []interface{}
	err := _OPContractsManager.contract.Call(opts, &out, "validate", _input, _allowFailure)

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Validate is a free data retrieval call binding the contract method 0x41fe5385.
//
// Solidity: function validate((address,bytes32,uint256,address) _input, bool _allowFailure) view returns(string)
func (_OPContractsManager *OPContractsManagerSession) Validate(_input OPContractsManagerStandardValidatorValidationInput, _allowFailure bool) (string, error) {
	return _OPContractsManager.Contract.Validate(&_OPContractsManager.CallOpts, _input, _allowFailure)
}

// Validate is a free data retrieval call binding the contract method 0x41fe5385.
//
// Solidity: function validate((address,bytes32,uint256,address) _input, bool _allowFailure) view returns(string)
func (_OPContractsManager *OPContractsManagerCallerSession) Validate(_input OPContractsManagerStandardValidatorValidationInput, _allowFailure bool) (string, error) {
	return _OPContractsManager.Contract.Validate(&_OPContractsManager.CallOpts, _input, _allowFailure)
}

// Validate0 is a free data retrieval call binding the contract method 0xf3edcbe1.
//
// Solidity: function validate((address,bytes32,bytes32,uint256,address) _input, bool _allowFailure) view returns(string)
func (_OPContractsManager *OPContractsManagerCaller) Validate0(opts *bind.CallOpts, _input OPContractsManagerStandardValidatorValidationInputDev, _allowFailure bool) (string, error) {
	var out []interface{}
	err := _OPContractsManager.contract.Call(opts, &out, "validate0", _input, _allowFailure)

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Validate0 is a free data retrieval call binding the contract method 0xf3edcbe1.
//
// Solidity: function validate((address,bytes32,bytes32,uint256,address) _input, bool _allowFailure) view returns(string)
func (_OPContractsManager *OPContractsManagerSession) Validate0(_input OPContractsManagerStandardValidatorValidationInputDev, _allowFailure bool) (string, error) {
	return _OPContractsManager.Contract.Validate0(&_OPContractsManager.CallOpts, _input, _allowFailure)
}

// Validate0 is a free data retrieval call binding the contract method 0xf3edcbe1.
//
// Solidity: function validate((address,bytes32,bytes32,uint256,address) _input, bool _allowFailure) view returns(string)
func (_OPContractsManager *OPContractsManagerCallerSession) Validate0(_input OPContractsManagerStandardValidatorValidationInputDev, _allowFailure bool) (string, error) {
	return _OPContractsManager.Contract.Validate0(&_OPContractsManager.CallOpts, _input, _allowFailure)
}

// ValidateWithOverrides is a free data retrieval call binding the contract method 0x0e9d5cb9.
//
// Solidity: function validateWithOverrides((address,bytes32,uint256,address) _input, bool _allowFailure, (address,address) _overrides) view returns(string)
func (_OPContractsManager *OPContractsManagerCaller) ValidateWithOverrides(opts *bind.CallOpts, _input OPContractsManagerStandardValidatorValidationInput, _allowFailure bool, _overrides OPContractsManagerStandardValidatorValidationOverrides) (string, error) {
	var out []interface{}
	err := _OPContractsManager.contract.Call(opts, &out, "validateWithOverrides", _input, _allowFailure, _overrides)

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// ValidateWithOverrides is a free data retrieval call binding the contract method 0x0e9d5cb9.
//
// Solidity: function validateWithOverrides((address,bytes32,uint256,address) _input, bool _allowFailure, (address,address) _overrides) view returns(string)
func (_OPContractsManager *OPContractsManagerSession) ValidateWithOverrides(_input OPContractsManagerStandardValidatorValidationInput, _allowFailure bool, _overrides OPContractsManagerStandardValidatorValidationOverrides) (string, error) {
	return _OPContractsManager.Contract.ValidateWithOverrides(&_OPContractsManager.CallOpts, _input, _allowFailure, _overrides)
}

// ValidateWithOverrides is a free data retrieval call binding the contract method 0x0e9d5cb9.
//
// Solidity: function validateWithOverrides((address,bytes32,uint256,address) _input, bool _allowFailure, (address,address) _overrides) view returns(string)
func (_OPContractsManager *OPContractsManagerCallerSession) ValidateWithOverrides(_input OPContractsManagerStandardValidatorValidationInput, _allowFailure bool, _overrides OPContractsManagerStandardValidatorValidationOverrides) (string, error) {
	return _OPContractsManager.Contract.ValidateWithOverrides(&_OPContractsManager.CallOpts, _input, _allowFailure, _overrides)
}

// ValidateWithOverrides0 is a free data retrieval call binding the contract method 0x8970ac44.
//
// Solidity: function validateWithOverrides((address,bytes32,bytes32,uint256,address) _input, bool _allowFailure, (address,address) _overrides) view returns(string)
func (_OPContractsManager *OPContractsManagerCaller) ValidateWithOverrides0(opts *bind.CallOpts, _input OPContractsManagerStandardValidatorValidationInputDev, _allowFailure bool, _overrides OPContractsManagerStandardValidatorValidationOverrides) (string, error) {
	var out []interface{}
	err := _OPContractsManager.contract.Call(opts, &out, "validateWithOverrides0", _input, _allowFailure, _overrides)

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// ValidateWithOverrides0 is a free data retrieval call binding the contract method 0x8970ac44.
//
// Solidity: function validateWithOverrides((address,bytes32,bytes32,uint256,address) _input, bool _allowFailure, (address,address) _overrides) view returns(string)
func (_OPContractsManager *OPContractsManagerSession) ValidateWithOverrides0(_input OPContractsManagerStandardValidatorValidationInputDev, _allowFailure bool, _overrides OPContractsManagerStandardValidatorValidationOverrides) (string, error) {
	return _OPContractsManager.Contract.ValidateWithOverrides0(&_OPContractsManager.CallOpts, _input, _allowFailure, _overrides)
}

// ValidateWithOverrides0 is a free data retrieval call binding the contract method 0x8970ac44.
//
// Solidity: function validateWithOverrides((address,bytes32,bytes32,uint256,address) _input, bool _allowFailure, (address,address) _overrides) view returns(string)
func (_OPContractsManager *OPContractsManagerCallerSession) ValidateWithOverrides0(_input OPContractsManagerStandardValidatorValidationInputDev, _allowFailure bool, _overrides OPContractsManagerStandardValidatorValidationOverrides) (string, error) {
	return _OPContractsManager.Contract.ValidateWithOverrides0(&_OPContractsManager.CallOpts, _input, _allowFailure, _overrides)
}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() pure returns(string)
func (_OPContractsManager *OPContractsManagerCaller) Version(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _OPContractsManager.contract.Call(opts, &out, "version")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() pure returns(string)
func (_OPContractsManager *OPContractsManagerSession) Version() (string, error) {
	return _OPContractsManager.Contract.Version(&_OPContractsManager.CallOpts)
}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() pure returns(string)
func (_OPContractsManager *OPContractsManagerCallerSession) Version() (string, error) {
	return _OPContractsManager.Contract.Version(&_OPContractsManager.CallOpts)
}

// AddGameType is a paid mutator transaction binding the contract method 0x604aa628.
//
// Solidity: function addGameType((string,address,address,uint32,bytes32,uint256,uint256,uint64,uint64,uint256,address,bool)[] _gameConfigs) returns((address,address)[])
func (_OPContractsManager *OPContractsManagerTransactor) AddGameType(opts *bind.TransactOpts, _gameConfigs []OPContractsManagerAddGameInput) (*types.Transaction, error) {
	return _OPContractsManager.contract.Transact(opts, "addGameType", _gameConfigs)
}

// AddGameType is a paid mutator transaction binding the contract method 0x604aa628.
//
// Solidity: function addGameType((string,address,address,uint32,bytes32,uint256,uint256,uint64,uint64,uint256,address,bool)[] _gameConfigs) returns((address,address)[])
func (_OPContractsManager *OPContractsManagerSession) AddGameType(_gameConfigs []OPContractsManagerAddGameInput) (*types.Transaction, error) {
	return _OPContractsManager.Contract.AddGameType(&_OPContractsManager.TransactOpts, _gameConfigs)
}

// AddGameType is a paid mutator transaction binding the contract method 0x604aa628.
//
// Solidity: function addGameType((string,address,address,uint32,bytes32,uint256,uint256,uint64,uint64,uint256,address,bool)[] _gameConfigs) returns((address,address)[])
func (_OPContractsManager *OPContractsManagerTransactorSession) AddGameType(_gameConfigs []OPContractsManagerAddGameInput) (*types.Transaction, error) {
	return _OPContractsManager.Contract.AddGameType(&_OPContractsManager.TransactOpts, _gameConfigs)
}

// Deploy is a paid mutator transaction binding the contract method 0x6ab5f661.
//
// Solidity: function deploy(((address,address,address,address,address,address),uint32,uint32,uint256,bytes,string,uint64,uint32,bytes32,uint256,uint256,uint64,uint64,bool) _input) returns((address,address,address,address,address,address,address,address,address,address,address,address,address,address,address))
func (_OPContractsManager *OPContractsManagerTransactor) Deploy(opts *bind.TransactOpts, _input OPContractsManagerDeployInput) (*types.Transaction, error) {
	return _OPContractsManager.contract.Transact(opts, "deploy", _input)
}

// Deploy is a paid mutator transaction binding the contract method 0x6ab5f661.
//
// Solidity: function deploy(((address,address,address,address,address,address),uint32,uint32,uint256,bytes,string,uint64,uint32,bytes32,uint256,uint256,uint64,uint64,bool) _input) returns((address,address,address,address,address,address,address,address,address,address,address,address,address,address,address))
func (_OPContractsManager *OPContractsManagerSession) Deploy(_input OPContractsManagerDeployInput) (*types.Transaction, error) {
	return _OPContractsManager.Contract.Deploy(&_OPContractsManager.TransactOpts, _input)
}

// Deploy is a paid mutator transaction binding the contract method 0x6ab5f661.
//
// Solidity: function deploy(((address,address,address,address,address,address),uint32,uint32,uint256,bytes,string,uint64,uint32,bytes32,uint256,uint256,uint64,uint64,bool) _input) returns((address,address,address,address,address,address,address,address,address,address,address,address,address,address,address))
func (_OPContractsManager *OPContractsManagerTransactorSession) Deploy(_input OPContractsManagerDeployInput) (*types.Transaction, error) {
	return _OPContractsManager.Contract.Deploy(&_OPContractsManager.TransactOpts, _input)
}

// Migrate is a paid mutator transaction binding the contract method 0x58084273.
//
// Solidity: function migrate((bool,(bytes32,uint256),(address,address,uint256,uint256,uint256,uint64,uint64),(address,bytes32,bytes32)[]) _input) returns()
func (_OPContractsManager *OPContractsManagerTransactor) Migrate(opts *bind.TransactOpts, _input OPContractsManagerInteropMigratorMigrateInput) (*types.Transaction, error) {
	return _OPContractsManager.contract.Transact(opts, "migrate", _input)
}

// Migrate is a paid mutator transaction binding the contract method 0x58084273.
//
// Solidity: function migrate((bool,(bytes32,uint256),(address,address,uint256,uint256,uint256,uint64,uint64),(address,bytes32,bytes32)[]) _input) returns()
func (_OPContractsManager *OPContractsManagerSession) Migrate(_input OPContractsManagerInteropMigratorMigrateInput) (*types.Transaction, error) {
	return _OPContractsManager.Contract.Migrate(&_OPContractsManager.TransactOpts, _input)
}

// Migrate is a paid mutator transaction binding the contract method 0x58084273.
//
// Solidity: function migrate((bool,(bytes32,uint256),(address,address,uint256,uint256,uint256,uint64,uint64),(address,bytes32,bytes32)[]) _input) returns()
func (_OPContractsManager *OPContractsManagerTransactorSession) Migrate(_input OPContractsManagerInteropMigratorMigrateInput) (*types.Transaction, error) {
	return _OPContractsManager.Contract.Migrate(&_OPContractsManager.TransactOpts, _input)
}

// UpdatePrestate is a paid mutator transaction binding the contract method 0xb23cc044.
//
// Solidity: function updatePrestate((address,bytes32,bytes32)[] _prestateUpdateInputs) returns()
func (_OPContractsManager *OPContractsManagerTransactor) UpdatePrestate(opts *bind.TransactOpts, _prestateUpdateInputs []OPContractsManagerUpdatePrestateInput) (*types.Transaction, error) {
	return _OPContractsManager.contract.Transact(opts, "updatePrestate", _prestateUpdateInputs)
}

// UpdatePrestate is a paid mutator transaction binding the contract method 0xb23cc044.
//
// Solidity: function updatePrestate((address,bytes32,bytes32)[] _prestateUpdateInputs) returns()
func (_OPContractsManager *OPContractsManagerSession) UpdatePrestate(_prestateUpdateInputs []OPContractsManagerUpdatePrestateInput) (*types.Transaction, error) {
	return _OPContractsManager.Contract.UpdatePrestate(&_OPContractsManager.TransactOpts, _prestateUpdateInputs)
}

// UpdatePrestate is a paid mutator transaction binding the contract method 0xb23cc044.
//
// Solidity: function updatePrestate((address,bytes32,bytes32)[] _prestateUpdateInputs) returns()
func (_OPContractsManager *OPContractsManagerTransactorSession) UpdatePrestate(_prestateUpdateInputs []OPContractsManagerUpdatePrestateInput) (*types.Transaction, error) {
	return _OPContractsManager.Contract.UpdatePrestate(&_OPContractsManager.TransactOpts, _prestateUpdateInputs)
}

// Upgrade is a paid mutator transaction binding the contract method 0xcbeda5a7.
//
// Solidity: function upgrade((address,bytes32,bytes32)[] _opChainConfigs) returns()
func (_OPContractsManager *OPContractsManagerTransactor) Upgrade(opts *bind.TransactOpts, _opChainConfigs []OPContractsManagerOpChainConfig) (*types.Transaction, error) {
	return _OPContractsManager.contract.Transact(opts, "upgrade", _opChainConfigs)
}

// Upgrade is a paid mutator transaction binding the contract method 0xcbeda5a7.
//
// Solidity: function upgrade((address,bytes32,bytes32)[] _opChainConfigs) returns()
func (_OPContractsManager *OPContractsManagerSession) Upgrade(_opChainConfigs []OPContractsManagerOpChainConfig) (*types.Transaction, error) {
	return _OPContractsManager.Contract.Upgrade(&_OPContractsManager.TransactOpts, _opChainConfigs)
}

// Upgrade is a paid mutator transaction binding the contract method 0xcbeda5a7.
//
// Solidity: function upgrade((address,bytes32,bytes32)[] _opChainConfigs) returns()
func (_OPContractsManager *OPContractsManagerTransactorSession) Upgrade(_opChainConfigs []OPContractsManagerOpChainConfig) (*types.Transaction, error) {
	return _OPContractsManager.Contract.Upgrade(&_OPContractsManager.TransactOpts, _opChainConfigs)
}

// UpgradeSuperchainConfig is a paid mutator transaction binding the contract method 0xc993f27c.
//
// Solidity: function upgradeSuperchainConfig(address _superchainConfig) returns()
func (_OPContractsManager *OPContractsManagerTransactor) UpgradeSuperchainConfig(opts *bind.TransactOpts, _superchainConfig common.Address) (*types.Transaction, error) {
	return _OPContractsManager.contract.Transact(opts, "upgradeSuperchainConfig", _superchainConfig)
}

// UpgradeSuperchainConfig is a paid mutator transaction binding the contract method 0xc993f27c.
//
// Solidity: function upgradeSuperchainConfig(address _superchainConfig) returns()
func (_OPContractsManager *OPContractsManagerSession) UpgradeSuperchainConfig(_superchainConfig common.Address) (*types.Transaction, error) {
	return _OPContractsManager.Contract.UpgradeSuperchainConfig(&_OPContractsManager.TransactOpts, _superchainConfig)
}

// UpgradeSuperchainConfig is a paid mutator transaction binding the contract method 0xc993f27c.
//
// Solidity: function upgradeSuperchainConfig(address _superchainConfig) returns()
func (_OPContractsManager *OPContractsManagerTransactorSession) UpgradeSuperchainConfig(_superchainConfig common.Address) (*types.Transaction, error) {
	return _OPContractsManager.Contract.UpgradeSuperchainConfig(&_OPContractsManager.TransactOpts, _superchainConfig)
}
