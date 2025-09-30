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
	ProxyAdmin              common.Address
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
	AddressManager                  common.Address
	Proxy                           common.Address
	ProxyAdmin                      common.Address
	L1ChugSplashProxy               common.Address
	ResolvedDelegateProxy           common.Address
	PermissionedDisputeGame1        common.Address
	PermissionedDisputeGame2        common.Address
	PermissionlessDisputeGame1      common.Address
	PermissionlessDisputeGame2      common.Address
	SuperPermissionedDisputeGame1   common.Address
	SuperPermissionedDisputeGame2   common.Address
	SuperPermissionlessDisputeGame1 common.Address
	SuperPermissionlessDisputeGame2 common.Address
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
	SystemConfigProxy common.Address
	ProxyAdmin        common.Address
	AbsolutePrestate  [32]byte
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
	ProxyAdmin       common.Address
	SysCfg           common.Address
	AbsolutePrestate [32]byte
	L2ChainID        *big.Int
}

// OPContractsManagerStandardValidatorValidationOverrides is an auto generated low-level Go binding around an user-defined struct.
type OPContractsManagerStandardValidatorValidationOverrides struct {
	L1PAOMultisig common.Address
	Challenger    common.Address
}

// Proposal is an auto generated low-level Go binding around an user-defined struct.
type Proposal struct {
	Root             [32]byte
	L2SequenceNumber *big.Int
}

// OPContractsManagerMetaData contains all meta data concerning the OPContractsManager contract.
var OPContractsManagerMetaData = &bind.MetaData{
	ABI: "[{\"type\":\"constructor\",\"inputs\":[{\"name\":\"_opcmGameTypeAdder\",\"type\":\"address\",\"internalType\":\"contractOPContractsManagerGameTypeAdder\"},{\"name\":\"_opcmDeployer\",\"type\":\"address\",\"internalType\":\"contractOPContractsManagerDeployer\"},{\"name\":\"_opcmUpgrader\",\"type\":\"address\",\"internalType\":\"contractOPContractsManagerUpgrader\"},{\"name\":\"_opcmInteropMigrator\",\"type\":\"address\",\"internalType\":\"contractOPContractsManagerInteropMigrator\"},{\"name\":\"_opcmStandardValidator\",\"type\":\"address\",\"internalType\":\"contractOPContractsManagerStandardValidator\"},{\"name\":\"_superchainConfig\",\"type\":\"address\",\"internalType\":\"contractISuperchainConfig\"},{\"name\":\"_protocolVersions\",\"type\":\"address\",\"internalType\":\"contractIProtocolVersions\"},{\"name\":\"_superchainProxyAdmin\",\"type\":\"address\",\"internalType\":\"contractIProxyAdmin\"}],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"addGameType\",\"inputs\":[{\"name\":\"_gameConfigs\",\"type\":\"tuple[]\",\"internalType\":\"structOPContractsManager.AddGameInput[]\",\"components\":[{\"name\":\"saltMixer\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"systemConfig\",\"type\":\"address\",\"internalType\":\"contractISystemConfig\"},{\"name\":\"proxyAdmin\",\"type\":\"address\",\"internalType\":\"contractIProxyAdmin\"},{\"name\":\"delayedWETH\",\"type\":\"address\",\"internalType\":\"contractIDelayedWETH\"},{\"name\":\"disputeGameType\",\"type\":\"uint32\",\"internalType\":\"GameType\"},{\"name\":\"disputeAbsolutePrestate\",\"type\":\"bytes32\",\"internalType\":\"Claim\"},{\"name\":\"disputeMaxGameDepth\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"disputeSplitDepth\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"disputeClockExtension\",\"type\":\"uint64\",\"internalType\":\"Duration\"},{\"name\":\"disputeMaxClockDuration\",\"type\":\"uint64\",\"internalType\":\"Duration\"},{\"name\":\"initialBond\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"vm\",\"type\":\"address\",\"internalType\":\"contractIBigStepper\"},{\"name\":\"permissioned\",\"type\":\"bool\",\"internalType\":\"bool\"}]}],\"outputs\":[{\"name\":\"\",\"type\":\"tuple[]\",\"internalType\":\"structOPContractsManager.AddGameOutput[]\",\"components\":[{\"name\":\"delayedWETH\",\"type\":\"address\",\"internalType\":\"contractIDelayedWETH\"},{\"name\":\"faultDisputeGame\",\"type\":\"address\",\"internalType\":\"contractIFaultDisputeGame\"}]}],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"blueprints\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"tuple\",\"internalType\":\"structOPContractsManager.Blueprints\",\"components\":[{\"name\":\"addressManager\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"proxy\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"proxyAdmin\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"l1ChugSplashProxy\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"resolvedDelegateProxy\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"permissionedDisputeGame1\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"permissionedDisputeGame2\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"permissionlessDisputeGame1\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"permissionlessDisputeGame2\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"superPermissionedDisputeGame1\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"superPermissionedDisputeGame2\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"superPermissionlessDisputeGame1\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"superPermissionlessDisputeGame2\",\"type\":\"address\",\"internalType\":\"address\"}]}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"chainIdToBatchInboxAddress\",\"inputs\":[{\"name\":\"_l2ChainId\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"deploy\",\"inputs\":[{\"name\":\"_input\",\"type\":\"tuple\",\"internalType\":\"structOPContractsManager.DeployInput\",\"components\":[{\"name\":\"roles\",\"type\":\"tuple\",\"internalType\":\"structOPContractsManager.Roles\",\"components\":[{\"name\":\"opChainProxyAdminOwner\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"systemConfigOwner\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"batcher\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"unsafeBlockSigner\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"proposer\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"challenger\",\"type\":\"address\",\"internalType\":\"address\"}]},{\"name\":\"basefeeScalar\",\"type\":\"uint32\",\"internalType\":\"uint32\"},{\"name\":\"blobBasefeeScalar\",\"type\":\"uint32\",\"internalType\":\"uint32\"},{\"name\":\"l2ChainId\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"startingAnchorRoot\",\"type\":\"bytes\",\"internalType\":\"bytes\"},{\"name\":\"saltMixer\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"gasLimit\",\"type\":\"uint64\",\"internalType\":\"uint64\"},{\"name\":\"disputeGameType\",\"type\":\"uint32\",\"internalType\":\"GameType\"},{\"name\":\"disputeAbsolutePrestate\",\"type\":\"bytes32\",\"internalType\":\"Claim\"},{\"name\":\"disputeMaxGameDepth\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"disputeSplitDepth\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"disputeClockExtension\",\"type\":\"uint64\",\"internalType\":\"Duration\"},{\"name\":\"disputeMaxClockDuration\",\"type\":\"uint64\",\"internalType\":\"Duration\"}]}],\"outputs\":[{\"name\":\"\",\"type\":\"tuple\",\"internalType\":\"structOPContractsManager.DeployOutput\",\"components\":[{\"name\":\"opChainProxyAdmin\",\"type\":\"address\",\"internalType\":\"contractIProxyAdmin\"},{\"name\":\"addressManager\",\"type\":\"address\",\"internalType\":\"contractIAddressManager\"},{\"name\":\"l1ERC721BridgeProxy\",\"type\":\"address\",\"internalType\":\"contractIL1ERC721Bridge\"},{\"name\":\"systemConfigProxy\",\"type\":\"address\",\"internalType\":\"contractISystemConfig\"},{\"name\":\"optimismMintableERC20FactoryProxy\",\"type\":\"address\",\"internalType\":\"contractIOptimismMintableERC20Factory\"},{\"name\":\"l1StandardBridgeProxy\",\"type\":\"address\",\"internalType\":\"contractIL1StandardBridge\"},{\"name\":\"l1CrossDomainMessengerProxy\",\"type\":\"address\",\"internalType\":\"contractIL1CrossDomainMessenger\"},{\"name\":\"ethLockboxProxy\",\"type\":\"address\",\"internalType\":\"contractIETHLockbox\"},{\"name\":\"optimismPortalProxy\",\"type\":\"address\",\"internalType\":\"contractIOptimismPortal2\"},{\"name\":\"disputeGameFactoryProxy\",\"type\":\"address\",\"internalType\":\"contractIDisputeGameFactory\"},{\"name\":\"anchorStateRegistryProxy\",\"type\":\"address\",\"internalType\":\"contractIAnchorStateRegistry\"},{\"name\":\"faultDisputeGame\",\"type\":\"address\",\"internalType\":\"contractIFaultDisputeGame\"},{\"name\":\"permissionedDisputeGame\",\"type\":\"address\",\"internalType\":\"contractIPermissionedDisputeGame\"},{\"name\":\"delayedWETHPermissionedGameProxy\",\"type\":\"address\",\"internalType\":\"contractIDelayedWETH\"},{\"name\":\"delayedWETHPermissionlessGameProxy\",\"type\":\"address\",\"internalType\":\"contractIDelayedWETH\"}]}],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"devFeatureBitmap\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"implementations\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"tuple\",\"internalType\":\"structOPContractsManager.Implementations\",\"components\":[{\"name\":\"superchainConfigImpl\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"protocolVersionsImpl\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"l1ERC721BridgeImpl\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"optimismPortalImpl\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"optimismPortalInteropImpl\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"ethLockboxImpl\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"systemConfigImpl\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"optimismMintableERC20FactoryImpl\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"l1CrossDomainMessengerImpl\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"l1StandardBridgeImpl\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"disputeGameFactoryImpl\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"anchorStateRegistryImpl\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"delayedWETHImpl\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"mipsImpl\",\"type\":\"address\",\"internalType\":\"address\"}]}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"isDevFeatureEnabled\",\"inputs\":[{\"name\":\"_feature\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"outputs\":[{\"name\":\"\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"migrate\",\"inputs\":[{\"name\":\"_input\",\"type\":\"tuple\",\"internalType\":\"structOPContractsManagerInteropMigrator.MigrateInput\",\"components\":[{\"name\":\"usePermissionlessGame\",\"type\":\"bool\",\"internalType\":\"bool\"},{\"name\":\"startingAnchorRoot\",\"type\":\"tuple\",\"internalType\":\"structProposal\",\"components\":[{\"name\":\"root\",\"type\":\"bytes32\",\"internalType\":\"Hash\"},{\"name\":\"l2SequenceNumber\",\"type\":\"uint256\",\"internalType\":\"uint256\"}]},{\"name\":\"gameParameters\",\"type\":\"tuple\",\"internalType\":\"structOPContractsManagerInteropMigrator.GameParameters\",\"components\":[{\"name\":\"proposer\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"challenger\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"maxGameDepth\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"splitDepth\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"initBond\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"clockExtension\",\"type\":\"uint64\",\"internalType\":\"Duration\"},{\"name\":\"maxClockDuration\",\"type\":\"uint64\",\"internalType\":\"Duration\"}]},{\"name\":\"opChainConfigs\",\"type\":\"tuple[]\",\"internalType\":\"structOPContractsManager.OpChainConfig[]\",\"components\":[{\"name\":\"systemConfigProxy\",\"type\":\"address\",\"internalType\":\"contractISystemConfig\"},{\"name\":\"proxyAdmin\",\"type\":\"address\",\"internalType\":\"contractIProxyAdmin\"},{\"name\":\"absolutePrestate\",\"type\":\"bytes32\",\"internalType\":\"Claim\"}]}]}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"opcmDeployer\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"contractOPContractsManagerDeployer\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"opcmGameTypeAdder\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"contractOPContractsManagerGameTypeAdder\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"opcmInteropMigrator\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"contractOPContractsManagerInteropMigrator\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"opcmStandardValidator\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"contractOPContractsManagerStandardValidator\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"opcmUpgrader\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"contractOPContractsManagerUpgrader\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"protocolVersions\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"contractIProtocolVersions\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"superchainConfig\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"contractISuperchainConfig\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"superchainProxyAdmin\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"contractIProxyAdmin\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"updatePrestate\",\"inputs\":[{\"name\":\"_prestateUpdateInputs\",\"type\":\"tuple[]\",\"internalType\":\"structOPContractsManager.OpChainConfig[]\",\"components\":[{\"name\":\"systemConfigProxy\",\"type\":\"address\",\"internalType\":\"contractISystemConfig\"},{\"name\":\"proxyAdmin\",\"type\":\"address\",\"internalType\":\"contractIProxyAdmin\"},{\"name\":\"absolutePrestate\",\"type\":\"bytes32\",\"internalType\":\"Claim\"}]}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"upgrade\",\"inputs\":[{\"name\":\"_opChainConfigs\",\"type\":\"tuple[]\",\"internalType\":\"structOPContractsManager.OpChainConfig[]\",\"components\":[{\"name\":\"systemConfigProxy\",\"type\":\"address\",\"internalType\":\"contractISystemConfig\"},{\"name\":\"proxyAdmin\",\"type\":\"address\",\"internalType\":\"contractIProxyAdmin\"},{\"name\":\"absolutePrestate\",\"type\":\"bytes32\",\"internalType\":\"Claim\"}]}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"validate\",\"inputs\":[{\"name\":\"_input\",\"type\":\"tuple\",\"internalType\":\"structOPContractsManagerStandardValidator.ValidationInput\",\"components\":[{\"name\":\"proxyAdmin\",\"type\":\"address\",\"internalType\":\"contractIProxyAdmin\"},{\"name\":\"sysCfg\",\"type\":\"address\",\"internalType\":\"contractISystemConfig\"},{\"name\":\"absolutePrestate\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"l2ChainID\",\"type\":\"uint256\",\"internalType\":\"uint256\"}]},{\"name\":\"_allowFailure\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"outputs\":[{\"name\":\"\",\"type\":\"string\",\"internalType\":\"string\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"validateWithOverrides\",\"inputs\":[{\"name\":\"_input\",\"type\":\"tuple\",\"internalType\":\"structOPContractsManagerStandardValidator.ValidationInput\",\"components\":[{\"name\":\"proxyAdmin\",\"type\":\"address\",\"internalType\":\"contractIProxyAdmin\"},{\"name\":\"sysCfg\",\"type\":\"address\",\"internalType\":\"contractISystemConfig\"},{\"name\":\"absolutePrestate\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"l2ChainID\",\"type\":\"uint256\",\"internalType\":\"uint256\"}]},{\"name\":\"_allowFailure\",\"type\":\"bool\",\"internalType\":\"bool\"},{\"name\":\"_overrides\",\"type\":\"tuple\",\"internalType\":\"structOPContractsManagerStandardValidator.ValidationOverrides\",\"components\":[{\"name\":\"l1PAOMultisig\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"challenger\",\"type\":\"address\",\"internalType\":\"address\"}]}],\"outputs\":[{\"name\":\"\",\"type\":\"string\",\"internalType\":\"string\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"version\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"string\",\"internalType\":\"string\"}],\"stateMutability\":\"pure\"},{\"type\":\"error\",\"name\":\"AddressHasNoCode\",\"inputs\":[{\"name\":\"who\",\"type\":\"address\",\"internalType\":\"address\"}]},{\"type\":\"error\",\"name\":\"AddressNotFound\",\"inputs\":[{\"name\":\"who\",\"type\":\"address\",\"internalType\":\"address\"}]},{\"type\":\"error\",\"name\":\"AlreadyReleased\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"InvalidChainId\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"InvalidGameConfigs\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"InvalidRoleAddress\",\"inputs\":[{\"name\":\"role\",\"type\":\"string\",\"internalType\":\"string\"}]},{\"type\":\"error\",\"name\":\"InvalidStartingAnchorRoot\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"LatestReleaseNotSet\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"OnlyDelegatecall\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"PrestateNotSet\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"PrestateRequired\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"SuperchainConfigMismatch\",\"inputs\":[{\"name\":\"systemConfig\",\"type\":\"address\",\"internalType\":\"contractISystemConfig\"}]},{\"type\":\"error\",\"name\":\"SuperchainProxyAdminMismatch\",\"inputs\":[]}]",
	Bin: "0x6101a06040523480156200001257600080fd5b5060405162002bc838038062002bc883398101604081905262000035916200030c565b60405163b6a4cd2160e01b81526001600160a01b03848116600483015288169063b6a4cd219060240160006040518083038186803b1580156200007757600080fd5b505afa1580156200008c573d6000803e3d6000fd5b505060405163b6a4cd2160e01b81526001600160a01b0385811660048301528a16925063b6a4cd21915060240160006040518083038186803b158015620000d257600080fd5b505afa158015620000e7573d6000803e3d6000fd5b505060405163b6a4cd2160e01b81526001600160a01b038b811660048301528a16925063b6a4cd21915060240160006040518083038186803b1580156200012d57600080fd5b505afa15801562000142573d6000803e3d6000fd5b505060405163b6a4cd2160e01b81526001600160a01b038a1660048201819052925063b6a4cd21915060240160006040518083038186803b1580156200018757600080fd5b505afa1580156200019c573d6000803e3d6000fd5b505060405163b6a4cd2160e01b81526001600160a01b0389811660048301528a16925063b6a4cd21915060240160006040518083038186803b158015620001e257600080fd5b505afa158015620001f7573d6000803e3d6000fd5b505060405163b6a4cd2160e01b81526001600160a01b0388811660048301528a16925063b6a4cd21915060240160006040518083038186803b1580156200023d57600080fd5b505afa15801562000252573d6000803e3d6000fd5b505060405163b6a4cd2160e01b81526001600160a01b0387811660048301528a16925063b6a4cd21915060240160006040518083038186803b1580156200029857600080fd5b505afa158015620002ad573d6000803e3d6000fd5b5050506001600160a01b039889166080525095871660a05293861660c05291851660e05284166101005283166101205282166101405216610160523061018052620003cd565b6001600160a01b03811681146200030957600080fd5b50565b600080600080600080600080610100898b0312156200032a57600080fd5b88516200033781620002f3565b60208a01519098506200034a81620002f3565b60408a01519097506200035d81620002f3565b60608a01519096506200037081620002f3565b60808a01519095506200038381620002f3565b60a08a01519094506200039681620002f3565b60c08a0151909350620003a981620002f3565b60e08a0151909250620003bc81620002f3565b809150509295985092959890939650565b60805160a05160c05160e051610100516101205161014051610160516101805161270c620004bc600039600081816104580152818161084f01528181610be501528181610cde0152610ed60152600061022d0152600061035801526000818161029c0152610a170152600081816103f0015281816106230152610ab80152600081816101d0015261091901526000818161018c01528181610db50152610fa00152600081816103310152818161056b01528181610719015281816107cc015281816109e001528181610b680152610e4b015260008181610417015281816105240152610caf015261270c6000f3fe608060405234801561001057600080fd5b50600436106101825760003560e01c8063613e827b116100d85780639a72745b1161008c578063ba7903db11610066578063ba7903db146103eb578063becbdf4a14610412578063ff2dd5a11461043957600080fd5b80639a72745b146103b0578063b0b807eb146103c3578063b51f9c2b146103d657600080fd5b80636624856a116100bd5780636624856a1461035357806367cda69c1461037a57806378ecabce1461038d57600080fd5b8063613e827b1461030c578063622d56f11461032c57600080fd5b806330d148881161013a57806335e80ab31161011457806335e80ab3146102975780633fe13f3f146102be57806354fd4d50146102d357600080fd5b806330d148881461024f57806330e9012c1461026f578063318b1b801461028457600080fd5b80631661a2e91161016b5780631661a2e9146101f25780631d8a4e92146102125780632b96b8391461022857600080fd5b806303dbe68c146101875780631481a724146101cb575b600080fd5b6101ae7f000000000000000000000000000000000000000000000000000000000000000081565b6040516001600160a01b0390911681526020015b60405180910390f35b6101ae7f000000000000000000000000000000000000000000000000000000000000000081565b610205610200366004611260565b61044c565b6040516101c29190611408565b61021a610567565b6040519081526020016101c2565b6101ae7f000000000000000000000000000000000000000000000000000000000000000081565b61026261025d3660046114dc565b6105f0565b6040516101c2919061156c565b6102776106a6565b6040516101c2919061157f565b6101ae6102923660046116b3565b61079a565b6101ae7f000000000000000000000000000000000000000000000000000000000000000081565b6102d16102cc3660046116cc565b610845565b005b60408051808201909152600581527f332e352e310000000000000000000000000000000000000000000000000000006020820152610262565b61031f61031a366004611708565b610943565b6040516101c29190611744565b6101ae7f000000000000000000000000000000000000000000000000000000000000000081565b6101ae7f000000000000000000000000000000000000000000000000000000000000000081565b610262610388366004611881565b610a85565b6103a061039b3660046116b3565b610b36565b60405190151581526020016101c2565b6102d16103be36600461191b565b610bdb565b6102d16103d13660046119eb565b610cd4565b6103de610de0565b6040516101c29190611a19565b6101ae7f000000000000000000000000000000000000000000000000000000000000000081565b6101ae7f000000000000000000000000000000000000000000000000000000000000000081565b6102d161044736600461191b565b610ecc565b60606001600160a01b037f00000000000000000000000000000000000000000000000000000000000000001630036104b0576040517f0a57d61d00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b6000826040516024016104c39190611b3c565b60408051601f198184030181529190526020810180517bffffffffffffffffffffffffffffffffffffffffffffffffffffffff167f1661a2e900000000000000000000000000000000000000000000000000000000179052905060006105497f000000000000000000000000000000000000000000000000000000000000000083610fc1565b90508080602001905181019061055f9190611c89565b949350505050565b60007f00000000000000000000000000000000000000000000000000000000000000006001600160a01b0316631d8a4e926040518163ffffffff1660e01b8152600401602060405180830381865afa1580156105c7573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906105eb9190611d52565b905090565b6040517f30d148880000000000000000000000000000000000000000000000000000000081526060906001600160a01b037f000000000000000000000000000000000000000000000000000000000000000016906330d148889061065a9086908690600401611d6b565b600060405180830381865afa158015610677573d6000803e3d6000fd5b505050506040513d6000823e601f3d908101601f1916820160405261069f9190810190611db6565b9392505050565b604080516101c081018252600080825260208201819052918101829052606081018290526080810182905260a0810182905260c0810182905260e08101829052610100810182905261012081018290526101408101829052610160810182905261018081018290526101a08101919091527f00000000000000000000000000000000000000000000000000000000000000006001600160a01b03166330e9012c6040518163ffffffff1660e01b81526004016101c060405180830381865afa158015610776573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906105eb9190611e2d565b6040517f318b1b80000000000000000000000000000000000000000000000000000000008152600481018290526000907f00000000000000000000000000000000000000000000000000000000000000006001600160a01b03169063318b1b8090602401602060405180830381865afa15801561081b573d6000803e3d6000fd5b505050506040513d601f19601f8201168201806040525081019061083f9190611f3d565b92915050565b6001600160a01b037f00000000000000000000000000000000000000000000000000000000000000001630036108a7576040517f0a57d61d00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b6000816040516024016108ba9190612032565b60408051601f198184030181529190526020810180517bffffffffffffffffffffffffffffffffffffffffffffffffffffffff167f3fe13f3f00000000000000000000000000000000000000000000000000000000179052905061093e7f000000000000000000000000000000000000000000000000000000000000000082610fc1565b505050565b604080516101e081018252600080825260208201819052818301819052606082018190526080820181905260a0820181905260c0820181905260e08201819052610100820181905261012082018190526101408201819052610160820181905261018082018190526101a082018190526101c082015290517fb2e48a3f0000000000000000000000000000000000000000000000000000000081527f00000000000000000000000000000000000000000000000000000000000000006001600160a01b03169063b2e48a3f90610a419085907f0000000000000000000000000000000000000000000000000000000000000000903390600401612245565b6101e0604051808303816000875af1158015610a61573d6000803e3d6000fd5b505050506040513d601f19601f8201168201806040525081019061083f91906123fa565b6040517f67cda69c0000000000000000000000000000000000000000000000000000000081526060906001600160a01b037f000000000000000000000000000000000000000000000000000000000000000016906367cda69c90610af190879087908790600401612511565b600060405180830381865afa158015610b0e573d6000803e3d6000fd5b505050506040513d6000823e601f3d908101601f1916820160405261055f9190810190611db6565b6040517f78ecabce000000000000000000000000000000000000000000000000000000008152600481018290526000907f00000000000000000000000000000000000000000000000000000000000000006001600160a01b0316906378ecabce90602401602060405180830381865afa158015610bb7573d6000803e3d6000fd5b505050506040513d601f19601f8201168201806040525081019061083f919061257b565b6001600160a01b037f0000000000000000000000000000000000000000000000000000000000000000163003610c3d576040517f0a57d61d00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b600081604051602401610c509190612598565b60408051601f198184030181529190526020810180517bffffffffffffffffffffffffffffffffffffffffffffffffffffffff167f9a72745b00000000000000000000000000000000000000000000000000000000179052905061093e7f000000000000000000000000000000000000000000000000000000000000000082610fc1565b6001600160a01b037f0000000000000000000000000000000000000000000000000000000000000000163003610d36576040517f0a57d61d00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b6040516001600160a01b0380841660248301528216604482015260009060640160408051601f198184030181529190526020810180517bffffffffffffffffffffffffffffffffffffffffffffffffffffffff167fb0b807eb000000000000000000000000000000000000000000000000000000001790529050610dda7f000000000000000000000000000000000000000000000000000000000000000082610fc1565b50505050565b604080516101a081018252600080825260208201819052918101829052606081018290526080810182905260a0810182905260c0810182905260e0810182905261010081018290526101208101829052610140810182905261016081018290526101808101919091527f00000000000000000000000000000000000000000000000000000000000000006001600160a01b031663b51f9c2b6040518163ffffffff1660e01b81526004016101a060405180830381865afa158015610ea8573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906105eb91906125f0565b6001600160a01b037f0000000000000000000000000000000000000000000000000000000000000000163003610f2e576040517f0a57d61d00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b600081604051602401610f419190612598565b60408051601f198184030181529190526020810180517bffffffffffffffffffffffffffffffffffffffffffffffffffffffff167fff2dd5a100000000000000000000000000000000000000000000000000000000179052905061093e7f0000000000000000000000000000000000000000000000000000000000000000825b6060600080846001600160a01b031684604051610fde91906126e3565b600060405180830381855af49150503d8060008114611019576040519150601f19603f3d011682016040523d82523d6000602084013e61101e565b606091505b50915091508161055f57805160208201fd5b7f4e487b7100000000000000000000000000000000000000000000000000000000600052604160045260246000fd5b6040516101a0810167ffffffffffffffff8111828210171561108357611083611030565b60405290565b6040805190810167ffffffffffffffff8111828210171561108357611083611030565b6040516060810167ffffffffffffffff8111828210171561108357611083611030565b6040516101c0810167ffffffffffffffff8111828210171561108357611083611030565b6040516101e0810167ffffffffffffffff8111828210171561108357611083611030565b604051601f8201601f1916810167ffffffffffffffff8111828210171561114057611140611030565b604052919050565b600067ffffffffffffffff82111561116257611162611030565b5060051b60200190565b600067ffffffffffffffff82111561118657611186611030565b50601f01601f191660200190565b600082601f8301126111a557600080fd5b81356111b86111b38261116c565b611117565b8181528460208386010111156111cd57600080fd5b816020850160208301376000918101602001919091529392505050565b6001600160a01b03811681146111ff57600080fd5b50565b803561120d816111ea565b919050565b803563ffffffff8116811461120d57600080fd5b67ffffffffffffffff811681146111ff57600080fd5b803561120d81611226565b80151581146111ff57600080fd5b803561120d81611247565b6000602080838503121561127357600080fd5b823567ffffffffffffffff8082111561128b57600080fd5b818501915085601f83011261129f57600080fd5b81356112ad6111b382611148565b81815260059190911b830184019084810190888311156112cc57600080fd5b8585015b838110156113fb578035858111156112e85760008081fd5b86016101a0818c03601f19018113156113015760008081fd5b61130961105f565b898301358881111561131b5760008081fd5b6113298e8c83870101611194565b8252506040611339818501611202565b8b830152606061134a818601611202565b828401526080915061135d828601611202565b9083015260a061136e858201611212565b8284015260c0915081850135818401525060e0808501358284015261010091508185013581840152506101206113a581860161123c565b8284015261014091506113b982860161123c565b8184015250610160808501358284015261018091506113d9828601611202565b908301526113e8848401611255565b90820152855250509186019186016112d0565b5098975050505050505050565b602080825282518282018190526000919060409081850190868401855b8281101561145757815180516001600160a01b0390811686529087015116868501529284019290850190600101611425565b5091979650505050505050565b60006080828403121561147657600080fd5b6040516080810181811067ffffffffffffffff8211171561149957611499611030565b60405290508082356114aa816111ea565b815260208301356114ba816111ea565b8060208301525060408301356040820152606083013560608201525092915050565b60008060a083850312156114ef57600080fd5b6114f98484611464565b9150608083013561150981611247565b809150509250929050565b60005b8381101561152f578181015183820152602001611517565b83811115610dda5750506000910152565b60008151808452611558816020860160208601611514565b601f01601f19169290920160200192915050565b60208152600061069f6020830184611540565b81516001600160a01b031681526101c0810160208301516115ab60208401826001600160a01b03169052565b5060408301516115c660408401826001600160a01b03169052565b5060608301516115e160608401826001600160a01b03169052565b5060808301516115fc60808401826001600160a01b03169052565b5060a083015161161760a08401826001600160a01b03169052565b5060c083015161163260c08401826001600160a01b03169052565b5060e083015161164d60e08401826001600160a01b03169052565b50610100838101516001600160a01b0390811691840191909152610120808501518216908401526101408085015182169084015261016080850151821690840152610180808501518216908401526101a08085015191821681850152905b505092915050565b6000602082840312156116c557600080fd5b5035919050565b6000602082840312156116de57600080fd5b813567ffffffffffffffff8111156116f557600080fd5b8201610160818503121561069f57600080fd5b60006020828403121561171a57600080fd5b813567ffffffffffffffff81111561173157600080fd5b8201610240818503121561069f57600080fd5b81516001600160a01b031681526101e08101602083015161177060208401826001600160a01b03169052565b50604083015161178b60408401826001600160a01b03169052565b5060608301516117a660608401826001600160a01b03169052565b5060808301516117c160808401826001600160a01b03169052565b5060a08301516117dc60a08401826001600160a01b03169052565b5060c08301516117f760c08401826001600160a01b03169052565b5060e083015161181260e08401826001600160a01b03169052565b50610100838101516001600160a01b0390811691840191909152610120808501518216908401526101408085015182169084015261016080850151821690840152610180808501518216908401526101a0808501518216908401526101c08085015191821681850152906116ab565b600080600083850360e081121561189757600080fd5b6118a18686611464565b935060808501356118b181611247565b925060407fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff60820112156118e357600080fd5b506118ec611089565b60a08501356118fa816111ea565b815260c085013561190a816111ea565b602082015292959194509192509050565b6000602080838503121561192e57600080fd5b823567ffffffffffffffff81111561194557600080fd5b8301601f8101851361195657600080fd5b80356119646111b382611148565b8181526060918202830184019184820191908884111561198357600080fd5b938501935b838510156119df5780858a0312156119a05760008081fd5b6119a86110ac565b85356119b3816111ea565b8152858701356119c2816111ea565b818801526040868101359082015283529384019391850191611988565b50979650505050505050565b600080604083850312156119fe57600080fd5b8235611a09816111ea565b91506020830135611509816111ea565b81516001600160a01b031681526101a081016020830151611a4560208401826001600160a01b03169052565b506040830151611a6060408401826001600160a01b03169052565b506060830151611a7b60608401826001600160a01b03169052565b506080830151611a9660808401826001600160a01b03169052565b5060a0830151611ab160a08401826001600160a01b03169052565b5060c0830151611acc60c08401826001600160a01b03169052565b5060e0830151611ae760e08401826001600160a01b03169052565b50610100838101516001600160a01b03908116918401919091526101208085015182169084015261014080850151821690840152610160808501518216908401526101808085015191821681850152906116ab565b60006020808301818452808551808352604092508286019150828160051b87010184880160005b83811015611c70577fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffc089840301855281516101a08151818652611ba882870182611540565b91505088820151611bc38a8701826001600160a01b03169052565b50878201516001600160a01b03908116868a015260608084015182169087015260808084015163ffffffff169087015260a0808401519087015260c0808401519087015260e080840151908701526101008084015167ffffffffffffffff9081169188019190915261012080850151909116908701526101408084015190870152610160808401519091169086015261018091820151151591909401529386019390860190600101611b63565b509098975050505050505050565b805161120d816111ea565b60006020808385031215611c9c57600080fd5b825167ffffffffffffffff811115611cb357600080fd5b8301601f81018513611cc457600080fd5b8051611cd26111b382611148565b81815260069190911b82018301908381019087831115611cf157600080fd5b928401925b82841015611d475760408489031215611d0f5760008081fd5b611d17611089565b8451611d22816111ea565b815284860151611d31816111ea565b8187015282526040939093019290840190611cf6565b979650505050505050565b600060208284031215611d6457600080fd5b5051919050565b60a08101611da782856001600160a01b038082511683528060208301511660208401525060408101516040830152606081015160608301525050565b82151560808301529392505050565b600060208284031215611dc857600080fd5b815167ffffffffffffffff811115611ddf57600080fd5b8201601f81018413611df057600080fd5b8051611dfe6111b38261116c565b818152856020838501011115611e1357600080fd5b611e24826020830160208601611514565b95945050505050565b60006101c08284031215611e4057600080fd5b611e486110cf565b611e5183611c7e565b8152611e5f60208401611c7e565b6020820152611e7060408401611c7e565b6040820152611e8160608401611c7e565b6060820152611e9260808401611c7e565b6080820152611ea360a08401611c7e565b60a0820152611eb460c08401611c7e565b60c0820152611ec560e08401611c7e565b60e0820152610100611ed8818501611c7e565b90820152610120611eea848201611c7e565b90820152610140611efc848201611c7e565b90820152610160611f0e848201611c7e565b90820152610180611f20848201611c7e565b908201526101a0611f32848201611c7e565b908201529392505050565b600060208284031215611f4f57600080fd5b815161069f816111ea565b60008083357fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe1843603018112611f8f57600080fd5b830160208101925035905067ffffffffffffffff811115611faf57600080fd5b606081023603821315611fc157600080fd5b9250929050565b8183526000602080850194508260005b85811015612027578135611feb816111ea565b6001600160a01b0390811688528284013590612006826111ea565b16878401526040828101359088015260609687019690910190600101611fd8565b509495945050505050565b602081526000823561204381611247565b8015156020840152506020830135604083015260408301356060830152606083013561206e816111ea565b6001600160a01b0380821660808501526080850135915061208e826111ea565b80821660a0850152505060a083013560c083015260c083013560e083015261010060e0840135818401528084013590506120c781611226565b61012067ffffffffffffffff8216818501526120e481860161123c565b9150506101406120ff8185018367ffffffffffffffff169052565b61210b81860186611f5a565b6101608681015292509050611e2461018085018383611fc8565b8035612130816111ea565b6001600160a01b03908116835260208201359061214c826111ea565b9081166020840152604082013590612163826111ea565b908116604084015260608201359061217a826111ea565b9081166060840152608082013590612191826111ea565b908116608084015260a0820135906121a8826111ea565b80821660a085015250505050565b60008083357fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe18436030181126121eb57600080fd5b830160208101925035905067ffffffffffffffff81111561220b57600080fd5b803603821315611fc157600080fd5b818352818160208501375060006020828401015260006020601f19601f840116840101905092915050565b606081526122566060820185612125565b600061226460c08601611212565b6101206122788185018363ffffffff169052565b61228460e08801611212565b915061014061229a8186018463ffffffff169052565b6101609250610100880135838601526122b5828901896121b6565b925061024061018081818901526122d16102a08901868561221a565b94506122df848c018c6121b6565b945092506101a07fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffa089870301818a015261231a86868661221a565b9550612327878d0161123c565b96506101c09450612343858a018867ffffffffffffffff169052565b61234e828d01611212565b96506101e09350612366848a018863ffffffff169052565b6102009650808c0135878a01525050610220838b013581890152828b013582890152612393868c0161123c565b67ffffffffffffffff81166102608a015295506123b1818c0161123c565b9550505050506123ce61028085018367ffffffffffffffff169052565b6001600160a01b038616602085015291506123e69050565b6001600160a01b038316604083015261055f565b60006101e0828403121561240d57600080fd5b6124156110f3565b61241e83611c7e565b815261242c60208401611c7e565b602082015261243d60408401611c7e565b604082015261244e60608401611c7e565b606082015261245f60808401611c7e565b608082015261247060a08401611c7e565b60a082015261248160c08401611c7e565b60c082015261249260e08401611c7e565b60e08201526101006124a5818501611c7e565b908201526101206124b7848201611c7e565b908201526101406124c9848201611c7e565b908201526101606124db848201611c7e565b908201526101806124ed848201611c7e565b908201526101a06124ff848201611c7e565b908201526101c0611f32848201611c7e565b60e0810161254d82866001600160a01b038082511683528060208301511660208401525060408101516040830152606081015160608301525050565b83151560808301526001600160a01b038084511660a08401528060208501511660c084015250949350505050565b60006020828403121561258d57600080fd5b815161069f81611247565b602080825282518282018190526000919060409081850190868401855b8281101561145757815180516001600160a01b03908116865287820151168786015285015185850152606090930192908501906001016125b5565b60006101a0828403121561260357600080fd5b61260b61105f565b61261483611c7e565b815261262260208401611c7e565b602082015261263360408401611c7e565b604082015261264460608401611c7e565b606082015261265560808401611c7e565b608082015261266660a08401611c7e565b60a082015261267760c08401611c7e565b60c082015261268860e08401611c7e565b60e082015261010061269b818501611c7e565b908201526101206126ad848201611c7e565b908201526101406126bf848201611c7e565b908201526101606126d1848201611c7e565b90820152610180611f32848201611c7e565b600082516126f5818460208701611514565b919091019291505056fea164736f6c634300080f000a",
}

// OPContractsManagerABI is the input ABI used to generate the binding from.
// Deprecated: Use OPContractsManagerMetaData.ABI instead.
var OPContractsManagerABI = OPContractsManagerMetaData.ABI

// OPContractsManagerBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use OPContractsManagerMetaData.Bin instead.
var OPContractsManagerBin = OPContractsManagerMetaData.Bin

// DeployOPContractsManager deploys a new Ethereum contract, binding an instance of OPContractsManager to it.
func DeployOPContractsManager(auth *bind.TransactOpts, backend bind.ContractBackend, _opcmGameTypeAdder common.Address, _opcmDeployer common.Address, _opcmUpgrader common.Address, _opcmInteropMigrator common.Address, _opcmStandardValidator common.Address, _superchainConfig common.Address, _protocolVersions common.Address, _superchainProxyAdmin common.Address, _l1PAO common.Address) (common.Address, *types.Transaction, *OPContractsManager, error) {
	parsed, err := OPContractsManagerMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(OPContractsManagerBin), backend, _opcmGameTypeAdder, _opcmDeployer, _opcmUpgrader, _opcmInteropMigrator, _opcmStandardValidator, _superchainConfig, _protocolVersions, _superchainProxyAdmin, _l1PAO)
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
// Solidity: function blueprints() view returns((address,address,address,address,address,address,address,address,address,address,address,address,address))
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
// Solidity: function blueprints() view returns((address,address,address,address,address,address,address,address,address,address,address,address,address))
func (_OPContractsManager *OPContractsManagerSession) Blueprints() (OPContractsManagerBlueprints, error) {
	return _OPContractsManager.Contract.Blueprints(&_OPContractsManager.CallOpts)
}

// Blueprints is a free data retrieval call binding the contract method 0xb51f9c2b.
//
// Solidity: function blueprints() view returns((address,address,address,address,address,address,address,address,address,address,address,address,address))
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
// Solidity: function implementations() view returns((address,address,address,address,address,address,address,address,address,address,address,address,address,address))
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
// Solidity: function implementations() view returns((address,address,address,address,address,address,address,address,address,address,address,address,address,address))
func (_OPContractsManager *OPContractsManagerSession) Implementations() (OPContractsManagerImplementations, error) {
	return _OPContractsManager.Contract.Implementations(&_OPContractsManager.CallOpts)
}

// Implementations is a free data retrieval call binding the contract method 0x30e9012c.
//
// Solidity: function implementations() view returns((address,address,address,address,address,address,address,address,address,address,address,address,address,address))
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

// SuperchainProxyAdmin is a free data retrieval call binding the contract method 0x2b96b839.
//
// Solidity: function superchainProxyAdmin() view returns(address)
func (_OPContractsManager *OPContractsManagerCaller) SuperchainProxyAdmin(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _OPContractsManager.contract.Call(opts, &out, "superchainProxyAdmin")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// SuperchainProxyAdmin is a free data retrieval call binding the contract method 0x2b96b839.
//
// Solidity: function superchainProxyAdmin() view returns(address)
func (_OPContractsManager *OPContractsManagerSession) SuperchainProxyAdmin() (common.Address, error) {
	return _OPContractsManager.Contract.SuperchainProxyAdmin(&_OPContractsManager.CallOpts)
}

// SuperchainProxyAdmin is a free data retrieval call binding the contract method 0x2b96b839.
//
// Solidity: function superchainProxyAdmin() view returns(address)
func (_OPContractsManager *OPContractsManagerCallerSession) SuperchainProxyAdmin() (common.Address, error) {
	return _OPContractsManager.Contract.SuperchainProxyAdmin(&_OPContractsManager.CallOpts)
}

// Validate is a free data retrieval call binding the contract method 0x30d14888.
//
// Solidity: function validate((address,address,bytes32,uint256) _input, bool _allowFailure) view returns(string)
func (_OPContractsManager *OPContractsManagerCaller) Validate(opts *bind.CallOpts, _input OPContractsManagerStandardValidatorValidationInput, _allowFailure bool) (string, error) {
	var out []interface{}
	err := _OPContractsManager.contract.Call(opts, &out, "validate", _input, _allowFailure)

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Validate is a free data retrieval call binding the contract method 0x30d14888.
//
// Solidity: function validate((address,address,bytes32,uint256) _input, bool _allowFailure) view returns(string)
func (_OPContractsManager *OPContractsManagerSession) Validate(_input OPContractsManagerStandardValidatorValidationInput, _allowFailure bool) (string, error) {
	return _OPContractsManager.Contract.Validate(&_OPContractsManager.CallOpts, _input, _allowFailure)
}

// Validate is a free data retrieval call binding the contract method 0x30d14888.
//
// Solidity: function validate((address,address,bytes32,uint256) _input, bool _allowFailure) view returns(string)
func (_OPContractsManager *OPContractsManagerCallerSession) Validate(_input OPContractsManagerStandardValidatorValidationInput, _allowFailure bool) (string, error) {
	return _OPContractsManager.Contract.Validate(&_OPContractsManager.CallOpts, _input, _allowFailure)
}

// ValidateWithOverrides is a free data retrieval call binding the contract method 0x67cda69c.
//
// Solidity: function validateWithOverrides((address,address,bytes32,uint256) _input, bool _allowFailure, (address,address) _overrides) view returns(string)
func (_OPContractsManager *OPContractsManagerCaller) ValidateWithOverrides(opts *bind.CallOpts, _input OPContractsManagerStandardValidatorValidationInput, _allowFailure bool, _overrides OPContractsManagerStandardValidatorValidationOverrides) (string, error) {
	var out []interface{}
	err := _OPContractsManager.contract.Call(opts, &out, "validateWithOverrides", _input, _allowFailure, _overrides)

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// ValidateWithOverrides is a free data retrieval call binding the contract method 0x67cda69c.
//
// Solidity: function validateWithOverrides((address,address,bytes32,uint256) _input, bool _allowFailure, (address,address) _overrides) view returns(string)
func (_OPContractsManager *OPContractsManagerSession) ValidateWithOverrides(_input OPContractsManagerStandardValidatorValidationInput, _allowFailure bool, _overrides OPContractsManagerStandardValidatorValidationOverrides) (string, error) {
	return _OPContractsManager.Contract.ValidateWithOverrides(&_OPContractsManager.CallOpts, _input, _allowFailure, _overrides)
}

// ValidateWithOverrides is a free data retrieval call binding the contract method 0x67cda69c.
//
// Solidity: function validateWithOverrides((address,address,bytes32,uint256) _input, bool _allowFailure, (address,address) _overrides) view returns(string)
func (_OPContractsManager *OPContractsManagerCallerSession) ValidateWithOverrides(_input OPContractsManagerStandardValidatorValidationInput, _allowFailure bool, _overrides OPContractsManagerStandardValidatorValidationOverrides) (string, error) {
	return _OPContractsManager.Contract.ValidateWithOverrides(&_OPContractsManager.CallOpts, _input, _allowFailure, _overrides)
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

// AddGameType is a paid mutator transaction binding the contract method 0x1661a2e9.
//
// Solidity: function addGameType((string,address,address,address,uint32,bytes32,uint256,uint256,uint64,uint64,uint256,address,bool)[] _gameConfigs) returns((address,address)[])
func (_OPContractsManager *OPContractsManagerTransactor) AddGameType(opts *bind.TransactOpts, _gameConfigs []OPContractsManagerAddGameInput) (*types.Transaction, error) {
	return _OPContractsManager.contract.Transact(opts, "addGameType", _gameConfigs)
}

// AddGameType is a paid mutator transaction binding the contract method 0x1661a2e9.
//
// Solidity: function addGameType((string,address,address,address,uint32,bytes32,uint256,uint256,uint64,uint64,uint256,address,bool)[] _gameConfigs) returns((address,address)[])
func (_OPContractsManager *OPContractsManagerSession) AddGameType(_gameConfigs []OPContractsManagerAddGameInput) (*types.Transaction, error) {
	return _OPContractsManager.Contract.AddGameType(&_OPContractsManager.TransactOpts, _gameConfigs)
}

// AddGameType is a paid mutator transaction binding the contract method 0x1661a2e9.
//
// Solidity: function addGameType((string,address,address,address,uint32,bytes32,uint256,uint256,uint64,uint64,uint256,address,bool)[] _gameConfigs) returns((address,address)[])
func (_OPContractsManager *OPContractsManagerTransactorSession) AddGameType(_gameConfigs []OPContractsManagerAddGameInput) (*types.Transaction, error) {
	return _OPContractsManager.Contract.AddGameType(&_OPContractsManager.TransactOpts, _gameConfigs)
}

// Deploy is a paid mutator transaction binding the contract method 0x613e827b.
//
// Solidity: function deploy(((address,address,address,address,address,address),uint32,uint32,uint256,bytes,string,uint64,uint32,bytes32,uint256,uint256,uint64,uint64) _input) returns((address,address,address,address,address,address,address,address,address,address,address,address,address,address,address))
func (_OPContractsManager *OPContractsManagerTransactor) Deploy(opts *bind.TransactOpts, _input OPContractsManagerDeployInput) (*types.Transaction, error) {
	return _OPContractsManager.contract.Transact(opts, "deploy", _input)
}

// Deploy is a paid mutator transaction binding the contract method 0x613e827b.
//
// Solidity: function deploy(((address,address,address,address,address,address),uint32,uint32,uint256,bytes,string,uint64,uint32,bytes32,uint256,uint256,uint64,uint64) _input) returns((address,address,address,address,address,address,address,address,address,address,address,address,address,address,address))
func (_OPContractsManager *OPContractsManagerSession) Deploy(_input OPContractsManagerDeployInput) (*types.Transaction, error) {
	return _OPContractsManager.Contract.Deploy(&_OPContractsManager.TransactOpts, _input)
}

// Deploy is a paid mutator transaction binding the contract method 0x613e827b.
//
// Solidity: function deploy(((address,address,address,address,address,address),uint32,uint32,uint256,bytes,string,uint64,uint32,bytes32,uint256,uint256,uint64,uint64) _input) returns((address,address,address,address,address,address,address,address,address,address,address,address,address,address,address))
func (_OPContractsManager *OPContractsManagerTransactorSession) Deploy(_input OPContractsManagerDeployInput) (*types.Transaction, error) {
	return _OPContractsManager.Contract.Deploy(&_OPContractsManager.TransactOpts, _input)
}

// Migrate is a paid mutator transaction binding the contract method 0x3fe13f3f.
//
// Solidity: function migrate((bool,(bytes32,uint256),(address,address,uint256,uint256,uint256,uint64,uint64),(address,address,bytes32)[]) _input) returns()
func (_OPContractsManager *OPContractsManagerTransactor) Migrate(opts *bind.TransactOpts, _input OPContractsManagerInteropMigratorMigrateInput) (*types.Transaction, error) {
	return _OPContractsManager.contract.Transact(opts, "migrate", _input)
}

// Migrate is a paid mutator transaction binding the contract method 0x3fe13f3f.
//
// Solidity: function migrate((bool,(bytes32,uint256),(address,address,uint256,uint256,uint256,uint64,uint64),(address,address,bytes32)[]) _input) returns()
func (_OPContractsManager *OPContractsManagerSession) Migrate(_input OPContractsManagerInteropMigratorMigrateInput) (*types.Transaction, error) {
	return _OPContractsManager.Contract.Migrate(&_OPContractsManager.TransactOpts, _input)
}

// Migrate is a paid mutator transaction binding the contract method 0x3fe13f3f.
//
// Solidity: function migrate((bool,(bytes32,uint256),(address,address,uint256,uint256,uint256,uint64,uint64),(address,address,bytes32)[]) _input) returns()
func (_OPContractsManager *OPContractsManagerTransactorSession) Migrate(_input OPContractsManagerInteropMigratorMigrateInput) (*types.Transaction, error) {
	return _OPContractsManager.Contract.Migrate(&_OPContractsManager.TransactOpts, _input)
}

// UpdatePrestate is a paid mutator transaction binding the contract method 0x9a72745b.
//
// Solidity: function updatePrestate((address,address,bytes32)[] _prestateUpdateInputs) returns()
func (_OPContractsManager *OPContractsManagerTransactor) UpdatePrestate(opts *bind.TransactOpts, _prestateUpdateInputs []OPContractsManagerOpChainConfig) (*types.Transaction, error) {
	return _OPContractsManager.contract.Transact(opts, "updatePrestate", _prestateUpdateInputs)
}

// UpdatePrestate is a paid mutator transaction binding the contract method 0x9a72745b.
//
// Solidity: function updatePrestate((address,address,bytes32)[] _prestateUpdateInputs) returns()
func (_OPContractsManager *OPContractsManagerSession) UpdatePrestate(_prestateUpdateInputs []OPContractsManagerOpChainConfig) (*types.Transaction, error) {
	return _OPContractsManager.Contract.UpdatePrestate(&_OPContractsManager.TransactOpts, _prestateUpdateInputs)
}

// UpdatePrestate is a paid mutator transaction binding the contract method 0x9a72745b.
//
// Solidity: function updatePrestate((address,address,bytes32)[] _prestateUpdateInputs) returns()
func (_OPContractsManager *OPContractsManagerTransactorSession) UpdatePrestate(_prestateUpdateInputs []OPContractsManagerOpChainConfig) (*types.Transaction, error) {
	return _OPContractsManager.Contract.UpdatePrestate(&_OPContractsManager.TransactOpts, _prestateUpdateInputs)
}

// Upgrade is a paid mutator transaction binding the contract method 0xff2dd5a1.
//
// Solidity: function upgrade((address,address,bytes32)[] _opChainConfigs) returns()
func (_OPContractsManager *OPContractsManagerTransactor) Upgrade(opts *bind.TransactOpts, _opChainConfigs []OPContractsManagerOpChainConfig) (*types.Transaction, error) {
	return _OPContractsManager.contract.Transact(opts, "upgrade", _opChainConfigs)
}

// Upgrade is a paid mutator transaction binding the contract method 0xff2dd5a1.
//
// Solidity: function upgrade((address,address,bytes32)[] _opChainConfigs) returns()
func (_OPContractsManager *OPContractsManagerSession) Upgrade(_opChainConfigs []OPContractsManagerOpChainConfig) (*types.Transaction, error) {
	return _OPContractsManager.Contract.Upgrade(&_OPContractsManager.TransactOpts, _opChainConfigs)
}

// Upgrade is a paid mutator transaction binding the contract method 0xff2dd5a1.
//
// Solidity: function upgrade((address,address,bytes32)[] _opChainConfigs) returns()
func (_OPContractsManager *OPContractsManagerTransactorSession) Upgrade(_opChainConfigs []OPContractsManagerOpChainConfig) (*types.Transaction, error) {
	return _OPContractsManager.Contract.Upgrade(&_OPContractsManager.TransactOpts, _opChainConfigs)
}
