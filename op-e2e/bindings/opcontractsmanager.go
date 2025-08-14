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
	ABI: "[{\"type\":\"constructor\",\"inputs\":[{\"name\":\"_opcmGameTypeAdder\",\"type\":\"address\",\"internalType\":\"contractOPContractsManagerGameTypeAdder\"},{\"name\":\"_opcmDeployer\",\"type\":\"address\",\"internalType\":\"contractOPContractsManagerDeployer\"},{\"name\":\"_opcmUpgrader\",\"type\":\"address\",\"internalType\":\"contractOPContractsManagerUpgrader\"},{\"name\":\"_opcmInteropMigrator\",\"type\":\"address\",\"internalType\":\"contractOPContractsManagerInteropMigrator\"},{\"name\":\"_opcmStandardValidator\",\"type\":\"address\",\"internalType\":\"contractOPContractsManagerStandardValidator\"},{\"name\":\"_superchainConfig\",\"type\":\"address\",\"internalType\":\"contractISuperchainConfig\"},{\"name\":\"_protocolVersions\",\"type\":\"address\",\"internalType\":\"contractIProtocolVersions\"},{\"name\":\"_superchainProxyAdmin\",\"type\":\"address\",\"internalType\":\"contractIProxyAdmin\"},{\"name\":\"_upgradeController\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"addGameType\",\"inputs\":[{\"name\":\"_gameConfigs\",\"type\":\"tuple[]\",\"internalType\":\"structOPContractsManager.AddGameInput[]\",\"components\":[{\"name\":\"saltMixer\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"systemConfig\",\"type\":\"address\",\"internalType\":\"contractISystemConfig\"},{\"name\":\"proxyAdmin\",\"type\":\"address\",\"internalType\":\"contractIProxyAdmin\"},{\"name\":\"delayedWETH\",\"type\":\"address\",\"internalType\":\"contractIDelayedWETH\"},{\"name\":\"disputeGameType\",\"type\":\"uint32\",\"internalType\":\"GameType\"},{\"name\":\"disputeAbsolutePrestate\",\"type\":\"bytes32\",\"internalType\":\"Claim\"},{\"name\":\"disputeMaxGameDepth\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"disputeSplitDepth\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"disputeClockExtension\",\"type\":\"uint64\",\"internalType\":\"Duration\"},{\"name\":\"disputeMaxClockDuration\",\"type\":\"uint64\",\"internalType\":\"Duration\"},{\"name\":\"initialBond\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"vm\",\"type\":\"address\",\"internalType\":\"contractIBigStepper\"},{\"name\":\"permissioned\",\"type\":\"bool\",\"internalType\":\"bool\"}]}],\"outputs\":[{\"name\":\"\",\"type\":\"tuple[]\",\"internalType\":\"structOPContractsManager.AddGameOutput[]\",\"components\":[{\"name\":\"delayedWETH\",\"type\":\"address\",\"internalType\":\"contractIDelayedWETH\"},{\"name\":\"faultDisputeGame\",\"type\":\"address\",\"internalType\":\"contractIFaultDisputeGame\"}]}],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"blueprints\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"tuple\",\"internalType\":\"structOPContractsManager.Blueprints\",\"components\":[{\"name\":\"addressManager\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"proxy\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"proxyAdmin\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"l1ChugSplashProxy\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"resolvedDelegateProxy\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"permissionedDisputeGame1\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"permissionedDisputeGame2\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"permissionlessDisputeGame1\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"permissionlessDisputeGame2\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"superPermissionedDisputeGame1\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"superPermissionedDisputeGame2\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"superPermissionlessDisputeGame1\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"superPermissionlessDisputeGame2\",\"type\":\"address\",\"internalType\":\"address\"}]}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"chainIdToBatchInboxAddress\",\"inputs\":[{\"name\":\"_l2ChainId\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"deploy\",\"inputs\":[{\"name\":\"_input\",\"type\":\"tuple\",\"internalType\":\"structOPContractsManager.DeployInput\",\"components\":[{\"name\":\"roles\",\"type\":\"tuple\",\"internalType\":\"structOPContractsManager.Roles\",\"components\":[{\"name\":\"opChainProxyAdminOwner\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"systemConfigOwner\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"batcher\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"unsafeBlockSigner\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"proposer\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"challenger\",\"type\":\"address\",\"internalType\":\"address\"}]},{\"name\":\"basefeeScalar\",\"type\":\"uint32\",\"internalType\":\"uint32\"},{\"name\":\"blobBasefeeScalar\",\"type\":\"uint32\",\"internalType\":\"uint32\"},{\"name\":\"l2ChainId\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"startingAnchorRoot\",\"type\":\"bytes\",\"internalType\":\"bytes\"},{\"name\":\"saltMixer\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"gasLimit\",\"type\":\"uint64\",\"internalType\":\"uint64\"},{\"name\":\"disputeGameType\",\"type\":\"uint32\",\"internalType\":\"GameType\"},{\"name\":\"disputeAbsolutePrestate\",\"type\":\"bytes32\",\"internalType\":\"Claim\"},{\"name\":\"disputeMaxGameDepth\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"disputeSplitDepth\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"disputeClockExtension\",\"type\":\"uint64\",\"internalType\":\"Duration\"},{\"name\":\"disputeMaxClockDuration\",\"type\":\"uint64\",\"internalType\":\"Duration\"}]}],\"outputs\":[{\"name\":\"\",\"type\":\"tuple\",\"internalType\":\"structOPContractsManager.DeployOutput\",\"components\":[{\"name\":\"opChainProxyAdmin\",\"type\":\"address\",\"internalType\":\"contractIProxyAdmin\"},{\"name\":\"addressManager\",\"type\":\"address\",\"internalType\":\"contractIAddressManager\"},{\"name\":\"l1ERC721BridgeProxy\",\"type\":\"address\",\"internalType\":\"contractIL1ERC721Bridge\"},{\"name\":\"systemConfigProxy\",\"type\":\"address\",\"internalType\":\"contractISystemConfig\"},{\"name\":\"optimismMintableERC20FactoryProxy\",\"type\":\"address\",\"internalType\":\"contractIOptimismMintableERC20Factory\"},{\"name\":\"l1StandardBridgeProxy\",\"type\":\"address\",\"internalType\":\"contractIL1StandardBridge\"},{\"name\":\"l1CrossDomainMessengerProxy\",\"type\":\"address\",\"internalType\":\"contractIL1CrossDomainMessenger\"},{\"name\":\"ethLockboxProxy\",\"type\":\"address\",\"internalType\":\"contractIETHLockbox\"},{\"name\":\"optimismPortalProxy\",\"type\":\"address\",\"internalType\":\"contractIOptimismPortal2\"},{\"name\":\"disputeGameFactoryProxy\",\"type\":\"address\",\"internalType\":\"contractIDisputeGameFactory\"},{\"name\":\"anchorStateRegistryProxy\",\"type\":\"address\",\"internalType\":\"contractIAnchorStateRegistry\"},{\"name\":\"faultDisputeGame\",\"type\":\"address\",\"internalType\":\"contractIFaultDisputeGame\"},{\"name\":\"permissionedDisputeGame\",\"type\":\"address\",\"internalType\":\"contractIPermissionedDisputeGame\"},{\"name\":\"delayedWETHPermissionedGameProxy\",\"type\":\"address\",\"internalType\":\"contractIDelayedWETH\"},{\"name\":\"delayedWETHPermissionlessGameProxy\",\"type\":\"address\",\"internalType\":\"contractIDelayedWETH\"}]}],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"implementations\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"tuple\",\"internalType\":\"structOPContractsManager.Implementations\",\"components\":[{\"name\":\"superchainConfigImpl\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"protocolVersionsImpl\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"l1ERC721BridgeImpl\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"optimismPortalImpl\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"ethLockboxImpl\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"systemConfigImpl\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"optimismMintableERC20FactoryImpl\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"l1CrossDomainMessengerImpl\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"l1StandardBridgeImpl\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"disputeGameFactoryImpl\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"anchorStateRegistryImpl\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"delayedWETHImpl\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"mipsImpl\",\"type\":\"address\",\"internalType\":\"address\"}]}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"migrate\",\"inputs\":[{\"name\":\"_input\",\"type\":\"tuple\",\"internalType\":\"structOPContractsManagerInteropMigrator.MigrateInput\",\"components\":[{\"name\":\"usePermissionlessGame\",\"type\":\"bool\",\"internalType\":\"bool\"},{\"name\":\"startingAnchorRoot\",\"type\":\"tuple\",\"internalType\":\"structProposal\",\"components\":[{\"name\":\"root\",\"type\":\"bytes32\",\"internalType\":\"Hash\"},{\"name\":\"l2SequenceNumber\",\"type\":\"uint256\",\"internalType\":\"uint256\"}]},{\"name\":\"gameParameters\",\"type\":\"tuple\",\"internalType\":\"structOPContractsManagerInteropMigrator.GameParameters\",\"components\":[{\"name\":\"proposer\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"challenger\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"maxGameDepth\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"splitDepth\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"initBond\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"clockExtension\",\"type\":\"uint64\",\"internalType\":\"Duration\"},{\"name\":\"maxClockDuration\",\"type\":\"uint64\",\"internalType\":\"Duration\"}]},{\"name\":\"opChainConfigs\",\"type\":\"tuple[]\",\"internalType\":\"structOPContractsManager.OpChainConfig[]\",\"components\":[{\"name\":\"systemConfigProxy\",\"type\":\"address\",\"internalType\":\"contractISystemConfig\"},{\"name\":\"proxyAdmin\",\"type\":\"address\",\"internalType\":\"contractIProxyAdmin\"},{\"name\":\"absolutePrestate\",\"type\":\"bytes32\",\"internalType\":\"Claim\"}]}]}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"opcmDeployer\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"contractOPContractsManagerDeployer\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"opcmGameTypeAdder\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"contractOPContractsManagerGameTypeAdder\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"opcmInteropMigrator\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"contractOPContractsManagerInteropMigrator\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"opcmStandardValidator\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"contractOPContractsManagerStandardValidator\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"opcmUpgrader\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"contractOPContractsManagerUpgrader\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"protocolVersions\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"contractIProtocolVersions\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"superchainConfig\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"contractISuperchainConfig\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"superchainProxyAdmin\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"contractIProxyAdmin\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"updatePrestate\",\"inputs\":[{\"name\":\"_prestateUpdateInputs\",\"type\":\"tuple[]\",\"internalType\":\"structOPContractsManager.OpChainConfig[]\",\"components\":[{\"name\":\"systemConfigProxy\",\"type\":\"address\",\"internalType\":\"contractISystemConfig\"},{\"name\":\"proxyAdmin\",\"type\":\"address\",\"internalType\":\"contractIProxyAdmin\"},{\"name\":\"absolutePrestate\",\"type\":\"bytes32\",\"internalType\":\"Claim\"}]}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"upgrade\",\"inputs\":[{\"name\":\"_opChainConfigs\",\"type\":\"tuple[]\",\"internalType\":\"structOPContractsManager.OpChainConfig[]\",\"components\":[{\"name\":\"systemConfigProxy\",\"type\":\"address\",\"internalType\":\"contractISystemConfig\"},{\"name\":\"proxyAdmin\",\"type\":\"address\",\"internalType\":\"contractIProxyAdmin\"},{\"name\":\"absolutePrestate\",\"type\":\"bytes32\",\"internalType\":\"Claim\"}]}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"upgradeController\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"validate\",\"inputs\":[{\"name\":\"_input\",\"type\":\"tuple\",\"internalType\":\"structOPContractsManagerStandardValidator.ValidationInput\",\"components\":[{\"name\":\"proxyAdmin\",\"type\":\"address\",\"internalType\":\"contractIProxyAdmin\"},{\"name\":\"sysCfg\",\"type\":\"address\",\"internalType\":\"contractISystemConfig\"},{\"name\":\"absolutePrestate\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"l2ChainID\",\"type\":\"uint256\",\"internalType\":\"uint256\"}]},{\"name\":\"_allowFailure\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"outputs\":[{\"name\":\"\",\"type\":\"string\",\"internalType\":\"string\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"validateWithOverrides\",\"inputs\":[{\"name\":\"_input\",\"type\":\"tuple\",\"internalType\":\"structOPContractsManagerStandardValidator.ValidationInput\",\"components\":[{\"name\":\"proxyAdmin\",\"type\":\"address\",\"internalType\":\"contractIProxyAdmin\"},{\"name\":\"sysCfg\",\"type\":\"address\",\"internalType\":\"contractISystemConfig\"},{\"name\":\"absolutePrestate\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"l2ChainID\",\"type\":\"uint256\",\"internalType\":\"uint256\"}]},{\"name\":\"_allowFailure\",\"type\":\"bool\",\"internalType\":\"bool\"},{\"name\":\"_overrides\",\"type\":\"tuple\",\"internalType\":\"structOPContractsManagerStandardValidator.ValidationOverrides\",\"components\":[{\"name\":\"l1PAOMultisig\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"challenger\",\"type\":\"address\",\"internalType\":\"address\"}]}],\"outputs\":[{\"name\":\"\",\"type\":\"string\",\"internalType\":\"string\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"version\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"string\",\"internalType\":\"string\"}],\"stateMutability\":\"pure\"},{\"type\":\"error\",\"name\":\"AddressHasNoCode\",\"inputs\":[{\"name\":\"who\",\"type\":\"address\",\"internalType\":\"address\"}]},{\"type\":\"error\",\"name\":\"AddressNotFound\",\"inputs\":[{\"name\":\"who\",\"type\":\"address\",\"internalType\":\"address\"}]},{\"type\":\"error\",\"name\":\"AlreadyReleased\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"InvalidChainId\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"InvalidGameConfigs\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"InvalidRoleAddress\",\"inputs\":[{\"name\":\"role\",\"type\":\"string\",\"internalType\":\"string\"}]},{\"type\":\"error\",\"name\":\"InvalidStartingAnchorRoot\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"LatestReleaseNotSet\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"OnlyDelegatecall\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"PrestateNotSet\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"PrestateRequired\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"SuperchainConfigMismatch\",\"inputs\":[{\"name\":\"systemConfig\",\"type\":\"address\",\"internalType\":\"contractISystemConfig\"}]},{\"type\":\"error\",\"name\":\"SuperchainProxyAdminMismatch\",\"inputs\":[]}]",
	Bin: "0x6101c06040523480156200001257600080fd5b506040516200271b3803806200271b833981016040819052620000359162000313565b60405163b6a4cd2160e01b81526001600160a01b03858116600483015289169063b6a4cd219060240160006040518083038186803b1580156200007757600080fd5b505afa1580156200008c573d6000803e3d6000fd5b505060405163b6a4cd2160e01b81526001600160a01b0386811660048301528b16925063b6a4cd21915060240160006040518083038186803b158015620000d257600080fd5b505afa158015620000e7573d6000803e3d6000fd5b505060405163b6a4cd2160e01b81526001600160a01b038c811660048301528b16925063b6a4cd21915060240160006040518083038186803b1580156200012d57600080fd5b505afa15801562000142573d6000803e3d6000fd5b505060405163b6a4cd2160e01b81526001600160a01b038b1660048201819052925063b6a4cd21915060240160006040518083038186803b1580156200018757600080fd5b505afa1580156200019c573d6000803e3d6000fd5b505060405163b6a4cd2160e01b81526001600160a01b038a811660048301528b16925063b6a4cd21915060240160006040518083038186803b158015620001e257600080fd5b505afa158015620001f7573d6000803e3d6000fd5b505060405163b6a4cd2160e01b81526001600160a01b0389811660048301528b16925063b6a4cd21915060240160006040518083038186803b1580156200023d57600080fd5b505afa15801562000252573d6000803e3d6000fd5b505060405163b6a4cd2160e01b81526001600160a01b0388811660048301528b16925063b6a4cd21915060240160006040518083038186803b1580156200029857600080fd5b505afa158015620002ad573d6000803e3d6000fd5b5050506001600160a01b03998a166080525096881660a05294871660c05292861660e052908516610100528416610120528316610140528216610160523061018052166101a052620003ea565b6001600160a01b03811681146200031057600080fd5b50565b60008060008060008060008060006101208a8c0312156200033357600080fd5b89516200034081620002fa565b60208b01519099506200035381620002fa565b60408b01519098506200036681620002fa565b60608b01519097506200037981620002fa565b60808b01519096506200038c81620002fa565b60a08b01519095506200039f81620002fa565b60c08b0151909450620003b281620002fa565b60e08b0151909350620003c581620002fa565b6101008b0151909250620003d981620002fa565b809150509295985092959850929598565b60805160a05160c05160e05161010051610120516101405161016051610180516101a051612245620004d660003960006103660152600081816104100152818161077701528181610a680152610c290152600081816102010152610ca60152600061032c0152600081816102700152818161093f0152610c850152600081816103a80152818161055201526109e00152600081816101ba01526108410152600081816101760152610d3701526000818161030501528181610642015281816106fa015281816109080152610bc20152600081816103cf015281816104dc0152610b3201526122456000f3fe608060405234801561001057600080fd5b506004361061016c5760003560e01c8063613e827b116100cd5780639a72745b11610081578063ba7903db11610066578063ba7903db146103a3578063becbdf4a146103ca578063ff2dd5a1146103f157600080fd5b80639a72745b14610388578063b51f9c2b1461039b57600080fd5b80636624856a116100b25780636624856a1461032757806367cda69c1461034e57806387543ef61461036157600080fd5b8063613e827b146102e0578063622d56f11461030057600080fd5b806330e9012c1161012457806335e80ab31161010957806335e80ab31461026b5780633fe13f3f1461029257806354fd4d50146102a757600080fd5b806330e9012c14610243578063318b1b801461025857600080fd5b80631661a2e9116101555780631661a2e9146101dc5780632b96b839146101fc57806330d148881461022357600080fd5b806303dbe68c146101715780631481a724146101b5575b600080fd5b6101987f000000000000000000000000000000000000000000000000000000000000000081565b6040516001600160a01b0390911681526020015b60405180910390f35b6101987f000000000000000000000000000000000000000000000000000000000000000081565b6101ef6101ea366004610fca565b610404565b6040516101ac9190611172565b6101987f000000000000000000000000000000000000000000000000000000000000000081565b610236610231366004611246565b61051f565b6040516101ac91906112d6565b61024b6105d7565b6040516101ac9190611407565b610198610266366004611416565b6106c8565b6101987f000000000000000000000000000000000000000000000000000000000000000081565b6102a56102a036600461142f565b61076d565b005b60408051808201909152600581527f322e362e300000000000000000000000000000000000000000000000000000006020820152610236565b6102f36102ee366004611472565b61086b565b6040516101ac91906114ae565b6101987f000000000000000000000000000000000000000000000000000000000000000081565b6101987f000000000000000000000000000000000000000000000000000000000000000081565b61023661035c3660046115ea565b6109ad565b6101987f000000000000000000000000000000000000000000000000000000000000000081565b6102a5610396366004611682565b610a5e565b61024b610b57565b6101987f000000000000000000000000000000000000000000000000000000000000000081565b6101987f000000000000000000000000000000000000000000000000000000000000000081565b6102a56103ff366004611682565b610c1f565b60606001600160a01b037f0000000000000000000000000000000000000000000000000000000000000000163003610468576040517f0a57d61d00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b60008260405160240161047b9190611752565b60408051601f198184030181529190526020810180517bffffffffffffffffffffffffffffffffffffffffffffffffffffffff167f1661a2e900000000000000000000000000000000000000000000000000000000179052905060006105017f000000000000000000000000000000000000000000000000000000000000000083610d58565b905080806020019051810190610517919061189f565b949350505050565b6040517f30d148880000000000000000000000000000000000000000000000000000000081526060906001600160a01b037f000000000000000000000000000000000000000000000000000000000000000016906330d14888906105899086908690600401611968565b600060405180830381865afa1580156105a6573d6000803e3d6000fd5b505050506040513d6000823e601f3d908101601f191682016040526105ce91908101906119b3565b90505b92915050565b604080516101a081018252600080825260208201819052918101829052606081018290526080810182905260a0810182905260c0810182905260e0810182905261010081018290526101208101829052610140810182905261016081018290526101808101919091527f00000000000000000000000000000000000000000000000000000000000000006001600160a01b03166330e9012c6040518163ffffffff1660e01b81526004016101a060405180830381865afa15801561069f573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906106c39190611b29565b905090565b6040517f318b1b80000000000000000000000000000000000000000000000000000000008152600481018290526000907f00000000000000000000000000000000000000000000000000000000000000006001600160a01b03169063318b1b8090602401602060405180830381865afa158015610749573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906105d19190611b46565b6001600160a01b037f00000000000000000000000000000000000000000000000000000000000000001630036107cf576040517f0a57d61d00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b6000816040516024016107e29190611c3b565b60408051601f198184030181529190526020810180517bffffffffffffffffffffffffffffffffffffffffffffffffffffffff167f3fe13f3f0000000000000000000000000000000000000000000000000000000017905290506108667f000000000000000000000000000000000000000000000000000000000000000082610d58565b505050565b604080516101e081018252600080825260208201819052818301819052606082018190526080820181905260a0820181905260c0820181905260e08201819052610100820181905261012082018190526101408201819052610160820181905261018082018190526101a082018190526101c082015290517fb2e48a3f0000000000000000000000000000000000000000000000000000000081527f00000000000000000000000000000000000000000000000000000000000000006001600160a01b03169063b2e48a3f906109699085907f0000000000000000000000000000000000000000000000000000000000000000903390600401611e4a565b6101e0604051808303816000875af1158015610989573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906105d19190611fff565b6040517f67cda69c0000000000000000000000000000000000000000000000000000000081526060906001600160a01b037f000000000000000000000000000000000000000000000000000000000000000016906367cda69c90610a1990879087908790600401612121565b600060405180830381865afa158015610a36573d6000803e3d6000fd5b505050506040513d6000823e601f3d908101601f1916820160405261051791908101906119b3565b6001600160a01b037f0000000000000000000000000000000000000000000000000000000000000000163003610ac0576040517f0a57d61d00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b600081604051602401610ad391906121dd565b60408051601f198184030181529190526020810180517bffffffffffffffffffffffffffffffffffffffffffffffffffffffff167f9a72745b0000000000000000000000000000000000000000000000000000000017905290506108667f000000000000000000000000000000000000000000000000000000000000000082610d58565b604080516101a081018252600080825260208201819052918101829052606081018290526080810182905260a0810182905260c0810182905260e0810182905261010081018290526101208101829052610140810182905261016081018290526101808101919091527f00000000000000000000000000000000000000000000000000000000000000006001600160a01b031663b51f9c2b6040518163ffffffff1660e01b81526004016101a060405180830381865afa15801561069f573d6000803e3d6000fd5b6001600160a01b037f0000000000000000000000000000000000000000000000000000000000000000163003610c81576040517f0a57d61d00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b60007f00000000000000000000000000000000000000000000000000000000000000007f000000000000000000000000000000000000000000000000000000000000000083604051602401610cd8939291906121f0565b60408051601f198184030181529190526020810180517bffffffffffffffffffffffffffffffffffffffffffffffffffffffff167f8a196cd40000000000000000000000000000000000000000000000000000000017905290506108667f0000000000000000000000000000000000000000000000000000000000000000825b6060600080846001600160a01b031684604051610d75919061221c565b600060405180830381855af49150503d8060008114610db0576040519150601f19603f3d011682016040523d82523d6000602084013e610db5565b606091505b50915091508161051757805160208201fd5b7f4e487b7100000000000000000000000000000000000000000000000000000000600052604160045260246000fd5b6040516101a0810167ffffffffffffffff81118282101715610e1a57610e1a610dc7565b60405290565b6040805190810167ffffffffffffffff81118282101715610e1a57610e1a610dc7565b6040516060810167ffffffffffffffff81118282101715610e1a57610e1a610dc7565b6040516101e0810167ffffffffffffffff81118282101715610e1a57610e1a610dc7565b604051601f8201601f1916810167ffffffffffffffff81118282101715610eb357610eb3610dc7565b604052919050565b600067ffffffffffffffff821115610ed557610ed5610dc7565b5060051b60200190565b600067ffffffffffffffff821115610ef957610ef9610dc7565b50601f01601f191660200190565b600082601f830112610f1857600080fd5b8135610f2b610f2682610edf565b610e8a565b818152846020838601011115610f4057600080fd5b816020850160208301376000918101602001919091529392505050565b6001600160a01b0381168114610f7257600080fd5b50565b8035610f8081610f5d565b919050565b803563ffffffff81168114610f8057600080fd5b67ffffffffffffffff81168114610f7257600080fd5b8035610f8081610f99565b80358015158114610f8057600080fd5b60006020808385031215610fdd57600080fd5b823567ffffffffffffffff80821115610ff557600080fd5b818501915085601f83011261100957600080fd5b8135611017610f2682610ebb565b81815260059190911b8301840190848101908883111561103657600080fd5b8585015b83811015611165578035858111156110525760008081fd5b86016101a0818c03601f190181131561106b5760008081fd5b611073610df6565b89830135888111156110855760008081fd5b6110938e8c83870101610f07565b82525060406110a3818501610f75565b8b83015260606110b4818601610f75565b82840152608091506110c7828601610f75565b9083015260a06110d8858201610f85565b8284015260c0915081850135818401525060e08085013582840152610100915081850135818401525061012061110f818601610faf565b828401526101409150611123828601610faf565b818401525061016080850135828401526101809150611143828601610f75565b90830152611152848401610fba565b908201528552505091860191860161103a565b5098975050505050505050565b602080825282518282018190526000919060409081850190868401855b828110156111c157815180516001600160a01b039081168652908701511686850152928401929085019060010161118f565b5091979650505050505050565b6000608082840312156111e057600080fd5b6040516080810181811067ffffffffffffffff8211171561120357611203610dc7565b604052905080823561121481610f5d565b8152602083013561122481610f5d565b8060208301525060408301356040820152606083013560608201525092915050565b60008060a0838503121561125957600080fd5b61126384846111ce565b915061127160808401610fba565b90509250929050565b60005b8381101561129557818101518382015260200161127d565b838111156112a4576000848401525b50505050565b600081518084526112c281602086016020860161127a565b601f01601f19169290920160200192915050565b6020815260006105ce60208301846112aa565b80516001600160a01b03168252602081015161131060208401826001600160a01b03169052565b50604081015161132b60408401826001600160a01b03169052565b50606081015161134660608401826001600160a01b03169052565b50608081015161136160808401826001600160a01b03169052565b5060a081015161137c60a08401826001600160a01b03169052565b5060c081015161139760c08401826001600160a01b03169052565b5060e08101516113b260e08401826001600160a01b03169052565b50610100818101516001600160a01b03908116918401919091526101208083015182169084015261014080830151821690840152610160808301518216908401526101808083015191821681850152906112a4565b6101a081016105d182846112e9565b60006020828403121561142857600080fd5b5035919050565b60006020828403121561144157600080fd5b813567ffffffffffffffff81111561145857600080fd5b8201610160818503121561146b57600080fd5b9392505050565b60006020828403121561148457600080fd5b813567ffffffffffffffff81111561149b57600080fd5b8201610240818503121561146b57600080fd5b81516001600160a01b031681526101e0810160208301516114da60208401826001600160a01b03169052565b5060408301516114f560408401826001600160a01b03169052565b50606083015161151060608401826001600160a01b03169052565b50608083015161152b60808401826001600160a01b03169052565b5060a083015161154660a08401826001600160a01b03169052565b5060c083015161156160c08401826001600160a01b03169052565b5060e083015161157c60e08401826001600160a01b03169052565b50610100838101516001600160a01b0390811691840191909152610120808501518216908401526101408085015182169084015261016080850151821690840152610180808501518216908401526101a0808501518216908401526101c09384015116929091019190915290565b600080600083850360e081121561160057600080fd5b61160a86866111ce565b935061161860808601610fba565b925060407fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff608201121561164a57600080fd5b50611653610e20565b60a085013561166181610f5d565b815260c085013561167181610f5d565b602082015292959194509192509050565b6000602080838503121561169557600080fd5b823567ffffffffffffffff8111156116ac57600080fd5b8301601f810185136116bd57600080fd5b80356116cb610f2682610ebb565b818152606091820283018401918482019190888411156116ea57600080fd5b938501935b838510156117465780858a0312156117075760008081fd5b61170f610e43565b853561171a81610f5d565b81528587013561172981610f5d565b8188015260408681013590820152835293840193918501916116ef565b50979650505050505050565b60006020808301818452808551808352604092508286019150828160051b87010184880160005b83811015611886577fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffc089840301855281516101a081518186526117be828701826112aa565b915050888201516117d98a8701826001600160a01b03169052565b50878201516001600160a01b03908116868a015260608084015182169087015260808084015163ffffffff169087015260a0808401519087015260c0808401519087015260e080840151908701526101008084015167ffffffffffffffff9081169188019190915261012080850151909116908701526101408084015190870152610160808401519091169086015261018091820151151591909401529386019390860190600101611779565b509098975050505050505050565b8051610f8081610f5d565b600060208083850312156118b257600080fd5b825167ffffffffffffffff8111156118c957600080fd5b8301601f810185136118da57600080fd5b80516118e8610f2682610ebb565b81815260069190911b8201830190838101908783111561190757600080fd5b928401925b8284101561195d57604084890312156119255760008081fd5b61192d610e20565b845161193881610f5d565b81528486015161194781610f5d565b818701528252604093909301929084019061190c565b979650505050505050565b60a081016119a482856001600160a01b038082511683528060208301511660208401525060408101516040830152606081015160608301525050565b82151560808301529392505050565b6000602082840312156119c557600080fd5b815167ffffffffffffffff8111156119dc57600080fd5b8201601f810184136119ed57600080fd5b80516119fb610f2682610edf565b818152856020838501011115611a1057600080fd5b611a2182602083016020860161127a565b95945050505050565b60006101a08284031215611a3d57600080fd5b611a45610df6565b9050611a5082611894565b8152611a5e60208301611894565b6020820152611a6f60408301611894565b6040820152611a8060608301611894565b6060820152611a9160808301611894565b6080820152611aa260a08301611894565b60a0820152611ab360c08301611894565b60c0820152611ac460e08301611894565b60e0820152610100611ad7818401611894565b90820152610120611ae9838201611894565b90820152610140611afb838201611894565b90820152610160611b0d838201611894565b90820152610180611b1f838201611894565b9082015292915050565b60006101a08284031215611b3c57600080fd5b6105ce8383611a2a565b600060208284031215611b5857600080fd5b815161146b81610f5d565b60008083357fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe1843603018112611b9857600080fd5b830160208101925035905067ffffffffffffffff811115611bb857600080fd5b606081023603821315611bca57600080fd5b9250929050565b8183526000602080850194508260005b85811015611c30578135611bf481610f5d565b6001600160a01b0390811688528284013590611c0f82610f5d565b16878401526040828101359088015260609687019690910190600101611be1565b509495945050505050565b60208152611c4882610fba565b15156020820152602082013560408201526040820135606082015260006060830135611c7381610f5d565b6001600160a01b03808216608085015260808501359150611c9382610f5d565b80821660a0850152505060a083013560c083015260c083013560e083015261010060e084013581840152808401359050611ccc81610f99565b61012067ffffffffffffffff821681850152611ce9818601610faf565b915050610140611d048185018367ffffffffffffffff169052565b611d1081860186611b63565b6101608681015292509050611a2161018085018383611bd1565b8035611d3581610f5d565b6001600160a01b039081168352602082013590611d5182610f5d565b9081166020840152604082013590611d6882610f5d565b9081166040840152606082013590611d7f82610f5d565b9081166060840152608082013590611d9682610f5d565b908116608084015260a082013590611dad82610f5d565b80821660a085015250505050565b60008083357fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe1843603018112611df057600080fd5b830160208101925035905067ffffffffffffffff811115611e1057600080fd5b803603821315611bca57600080fd5b818352818160208501375060006020828401015260006020601f19601f840116840101905092915050565b60608152611e5b6060820185611d2a565b6000611e6960c08601610f85565b610120611e7d8185018363ffffffff169052565b611e8960e08801610f85565b9150610140611e9f8186018463ffffffff169052565b610160925061010088013583860152611eba82890189611dbb565b92506102406101808181890152611ed66102a089018685611e1f565b9450611ee4848c018c611dbb565b945092506101a07fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffa089870301818a0152611f1f868686611e1f565b9550611f2c878d01610faf565b96506101c09450611f48858a018867ffffffffffffffff169052565b611f53828d01610f85565b96506101e09350611f6b848a018863ffffffff169052565b6102009650808c0135878a01525050610220838b013581890152828b013582890152611f98868c01610faf565b67ffffffffffffffff81166102608a01529550611fb6818c01610faf565b955050505050611fd361028085018367ffffffffffffffff169052565b6001600160a01b03861660208501529150611feb9050565b6001600160a01b0383166040830152610517565b60006101e0828403121561201257600080fd5b61201a610e66565b61202383611894565b815261203160208401611894565b602082015261204260408401611894565b604082015261205360608401611894565b606082015261206460808401611894565b608082015261207560a08401611894565b60a082015261208660c08401611894565b60c082015261209760e08401611894565b60e08201526101006120aa818501611894565b908201526101206120bc848201611894565b908201526101406120ce848201611894565b908201526101606120e0848201611894565b908201526101806120f2848201611894565b908201526101a0612104848201611894565b908201526101c0612116848201611894565b908201529392505050565b60e0810161215d82866001600160a01b038082511683528060208301511660208401525060408101516040830152606081015160608301525050565b83151560808301526001600160a01b038084511660a08401528060208501511660c084015250949350505050565b600081518084526020808501945080840160005b83811015611c3057815180516001600160a01b039081168952848201511684890152604090810151908801526060909601959082019060010161219f565b6020815260006105ce602083018461218b565b60006001600160a01b03808616835280851660208401525060606040830152611a21606083018461218b565b6000825161222e81846020870161127a565b919091019291505056fea164736f6c634300080f000a",
}

// OPContractsManagerABI is the input ABI used to generate the binding from.
// Deprecated: Use OPContractsManagerMetaData.ABI instead.
var OPContractsManagerABI = OPContractsManagerMetaData.ABI

// OPContractsManagerBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use OPContractsManagerMetaData.Bin instead.
var OPContractsManagerBin = OPContractsManagerMetaData.Bin

// DeployOPContractsManager deploys a new Ethereum contract, binding an instance of OPContractsManager to it.
func DeployOPContractsManager(auth *bind.TransactOpts, backend bind.ContractBackend, _opcmGameTypeAdder common.Address, _opcmDeployer common.Address, _opcmUpgrader common.Address, _opcmInteropMigrator common.Address, _opcmStandardValidator common.Address, _superchainConfig common.Address, _protocolVersions common.Address, _superchainProxyAdmin common.Address, _upgradeController common.Address) (common.Address, *types.Transaction, *OPContractsManager, error) {
	parsed, err := OPContractsManagerMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(OPContractsManagerBin), backend, _opcmGameTypeAdder, _opcmDeployer, _opcmUpgrader, _opcmInteropMigrator, _opcmStandardValidator, _superchainConfig, _protocolVersions, _superchainProxyAdmin, _upgradeController)
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

// Implementations is a free data retrieval call binding the contract method 0x30e9012c.
//
// Solidity: function implementations() view returns((address,address,address,address,address,address,address,address,address,address,address,address,address))
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
// Solidity: function implementations() view returns((address,address,address,address,address,address,address,address,address,address,address,address,address))
func (_OPContractsManager *OPContractsManagerSession) Implementations() (OPContractsManagerImplementations, error) {
	return _OPContractsManager.Contract.Implementations(&_OPContractsManager.CallOpts)
}

// Implementations is a free data retrieval call binding the contract method 0x30e9012c.
//
// Solidity: function implementations() view returns((address,address,address,address,address,address,address,address,address,address,address,address,address))
func (_OPContractsManager *OPContractsManagerCallerSession) Implementations() (OPContractsManagerImplementations, error) {
	return _OPContractsManager.Contract.Implementations(&_OPContractsManager.CallOpts)
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

// UpgradeController is a free data retrieval call binding the contract method 0x87543ef6.
//
// Solidity: function upgradeController() view returns(address)
func (_OPContractsManager *OPContractsManagerCaller) UpgradeController(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _OPContractsManager.contract.Call(opts, &out, "upgradeController")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// UpgradeController is a free data retrieval call binding the contract method 0x87543ef6.
//
// Solidity: function upgradeController() view returns(address)
func (_OPContractsManager *OPContractsManagerSession) UpgradeController() (common.Address, error) {
	return _OPContractsManager.Contract.UpgradeController(&_OPContractsManager.CallOpts)
}

// UpgradeController is a free data retrieval call binding the contract method 0x87543ef6.
//
// Solidity: function upgradeController() view returns(address)
func (_OPContractsManager *OPContractsManagerCallerSession) UpgradeController() (common.Address, error) {
	return _OPContractsManager.Contract.UpgradeController(&_OPContractsManager.CallOpts)
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
