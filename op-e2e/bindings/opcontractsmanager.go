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

// Proposal is an auto generated low-level Go binding around an user-defined struct.
type Proposal struct {
	Root             [32]byte
	L2SequenceNumber *big.Int
}

// OPContractsManagerMetaData contains all meta data concerning the OPContractsManager contract.
var OPContractsManagerMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"contractOPContractsManagerGameTypeAdder\",\"name\":\"_opcmGameTypeAdder\",\"type\":\"address\"},{\"internalType\":\"contractOPContractsManagerDeployer\",\"name\":\"_opcmDeployer\",\"type\":\"address\"},{\"internalType\":\"contractOPContractsManagerUpgrader\",\"name\":\"_opcmUpgrader\",\"type\":\"address\"},{\"internalType\":\"contractOPContractsManagerInteropMigrator\",\"name\":\"_opcmInteropMigrator\",\"type\":\"address\"},{\"internalType\":\"contractISuperchainConfig\",\"name\":\"_superchainConfig\",\"type\":\"address\"},{\"internalType\":\"contractIProtocolVersions\",\"name\":\"_protocolVersions\",\"type\":\"address\"},{\"internalType\":\"contractIProxyAdmin\",\"name\":\"_superchainProxyAdmin\",\"type\":\"address\"},{\"internalType\":\"string\",\"name\":\"_l1ContractsRelease\",\"type\":\"string\"},{\"internalType\":\"address\",\"name\":\"_upgradeController\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"inputs\":[{\"components\":[{\"internalType\":\"string\",\"name\":\"saltMixer\",\"type\":\"string\"},{\"internalType\":\"contractISystemConfig\",\"name\":\"systemConfig\",\"type\":\"address\"},{\"internalType\":\"contractIProxyAdmin\",\"name\":\"proxyAdmin\",\"type\":\"address\"},{\"internalType\":\"contractIDelayedWETH\",\"name\":\"delayedWETH\",\"type\":\"address\"},{\"internalType\":\"GameType\",\"name\":\"disputeGameType\",\"type\":\"uint32\"},{\"internalType\":\"Claim\",\"name\":\"disputeAbsolutePrestate\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"disputeMaxGameDepth\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"disputeSplitDepth\",\"type\":\"uint256\"},{\"internalType\":\"Duration\",\"name\":\"disputeClockExtension\",\"type\":\"uint64\"},{\"internalType\":\"Duration\",\"name\":\"disputeMaxClockDuration\",\"type\":\"uint64\"},{\"internalType\":\"uint256\",\"name\":\"initialBond\",\"type\":\"uint256\"},{\"internalType\":\"contractIBigStepper\",\"name\":\"vm\",\"type\":\"address\"},{\"internalType\":\"bool\",\"name\":\"permissioned\",\"type\":\"bool\"}],\"internalType\":\"structOPContractsManager.AddGameInput[]\",\"name\":\"_gameConfigs\",\"type\":\"tuple[]\"}],\"name\":\"addGameType\",\"outputs\":[{\"components\":[{\"internalType\":\"contractIDelayedWETH\",\"name\":\"delayedWETH\",\"type\":\"address\"},{\"internalType\":\"contractIFaultDisputeGame\",\"name\":\"faultDisputeGame\",\"type\":\"address\"}],\"internalType\":\"structOPContractsManager.AddGameOutput[]\",\"name\":\"\",\"type\":\"tuple[]\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"blueprints\",\"outputs\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"addressManager\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"proxy\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"proxyAdmin\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"l1ChugSplashProxy\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"resolvedDelegateProxy\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"permissionedDisputeGame1\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"permissionedDisputeGame2\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"permissionlessDisputeGame1\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"permissionlessDisputeGame2\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"superPermissionedDisputeGame1\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"superPermissionedDisputeGame2\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"superPermissionlessDisputeGame1\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"superPermissionlessDisputeGame2\",\"type\":\"address\"}],\"internalType\":\"structOPContractsManager.Blueprints\",\"name\":\"\",\"type\":\"tuple\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_l2ChainId\",\"type\":\"uint256\"}],\"name\":\"chainIdToBatchInboxAddress\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"opChainProxyAdminOwner\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"systemConfigOwner\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"batcher\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"unsafeBlockSigner\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"proposer\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"challenger\",\"type\":\"address\"}],\"internalType\":\"structOPContractsManager.Roles\",\"name\":\"roles\",\"type\":\"tuple\"},{\"internalType\":\"uint32\",\"name\":\"basefeeScalar\",\"type\":\"uint32\"},{\"internalType\":\"uint32\",\"name\":\"blobBasefeeScalar\",\"type\":\"uint32\"},{\"internalType\":\"uint256\",\"name\":\"l2ChainId\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"startingAnchorRoot\",\"type\":\"bytes\"},{\"internalType\":\"string\",\"name\":\"saltMixer\",\"type\":\"string\"},{\"internalType\":\"uint64\",\"name\":\"gasLimit\",\"type\":\"uint64\"},{\"internalType\":\"GameType\",\"name\":\"disputeGameType\",\"type\":\"uint32\"},{\"internalType\":\"Claim\",\"name\":\"disputeAbsolutePrestate\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"disputeMaxGameDepth\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"disputeSplitDepth\",\"type\":\"uint256\"},{\"internalType\":\"Duration\",\"name\":\"disputeClockExtension\",\"type\":\"uint64\"},{\"internalType\":\"Duration\",\"name\":\"disputeMaxClockDuration\",\"type\":\"uint64\"}],\"internalType\":\"structOPContractsManager.DeployInput\",\"name\":\"_input\",\"type\":\"tuple\"}],\"name\":\"deploy\",\"outputs\":[{\"components\":[{\"internalType\":\"contractIProxyAdmin\",\"name\":\"opChainProxyAdmin\",\"type\":\"address\"},{\"internalType\":\"contractIAddressManager\",\"name\":\"addressManager\",\"type\":\"address\"},{\"internalType\":\"contractIL1ERC721Bridge\",\"name\":\"l1ERC721BridgeProxy\",\"type\":\"address\"},{\"internalType\":\"contractISystemConfig\",\"name\":\"systemConfigProxy\",\"type\":\"address\"},{\"internalType\":\"contractIOptimismMintableERC20Factory\",\"name\":\"optimismMintableERC20FactoryProxy\",\"type\":\"address\"},{\"internalType\":\"contractIL1StandardBridge\",\"name\":\"l1StandardBridgeProxy\",\"type\":\"address\"},{\"internalType\":\"contractIL1CrossDomainMessenger\",\"name\":\"l1CrossDomainMessengerProxy\",\"type\":\"address\"},{\"internalType\":\"contractIETHLockbox\",\"name\":\"ethLockboxProxy\",\"type\":\"address\"},{\"internalType\":\"contractIOptimismPortal2\",\"name\":\"optimismPortalProxy\",\"type\":\"address\"},{\"internalType\":\"contractIDisputeGameFactory\",\"name\":\"disputeGameFactoryProxy\",\"type\":\"address\"},{\"internalType\":\"contractIAnchorStateRegistry\",\"name\":\"anchorStateRegistryProxy\",\"type\":\"address\"},{\"internalType\":\"contractIFaultDisputeGame\",\"name\":\"faultDisputeGame\",\"type\":\"address\"},{\"internalType\":\"contractIPermissionedDisputeGame\",\"name\":\"permissionedDisputeGame\",\"type\":\"address\"},{\"internalType\":\"contractIDelayedWETH\",\"name\":\"delayedWETHPermissionedGameProxy\",\"type\":\"address\"},{\"internalType\":\"contractIDelayedWETH\",\"name\":\"delayedWETHPermissionlessGameProxy\",\"type\":\"address\"}],\"internalType\":\"structOPContractsManager.DeployOutput\",\"name\":\"\",\"type\":\"tuple\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"implementations\",\"outputs\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"superchainConfigImpl\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"protocolVersionsImpl\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"l1ERC721BridgeImpl\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"optimismPortalImpl\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"ethLockboxImpl\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"systemConfigImpl\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"optimismMintableERC20FactoryImpl\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"l1CrossDomainMessengerImpl\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"l1StandardBridgeImpl\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"disputeGameFactoryImpl\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"anchorStateRegistryImpl\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"delayedWETHImpl\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"mipsImpl\",\"type\":\"address\"}],\"internalType\":\"structOPContractsManager.Implementations\",\"name\":\"\",\"type\":\"tuple\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"isRC\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"l1ContractsRelease\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"bool\",\"name\":\"usePermissionlessGame\",\"type\":\"bool\"},{\"components\":[{\"internalType\":\"Hash\",\"name\":\"root\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"l2SequenceNumber\",\"type\":\"uint256\"}],\"internalType\":\"structProposal\",\"name\":\"startingAnchorRoot\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"address\",\"name\":\"proposer\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"challenger\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"maxGameDepth\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"splitDepth\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"initBond\",\"type\":\"uint256\"},{\"internalType\":\"Duration\",\"name\":\"clockExtension\",\"type\":\"uint64\"},{\"internalType\":\"Duration\",\"name\":\"maxClockDuration\",\"type\":\"uint64\"}],\"internalType\":\"structOPContractsManagerInteropMigrator.GameParameters\",\"name\":\"gameParameters\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"contractISystemConfig\",\"name\":\"systemConfigProxy\",\"type\":\"address\"},{\"internalType\":\"contractIProxyAdmin\",\"name\":\"proxyAdmin\",\"type\":\"address\"},{\"internalType\":\"Claim\",\"name\":\"absolutePrestate\",\"type\":\"bytes32\"}],\"internalType\":\"structOPContractsManager.OpChainConfig[]\",\"name\":\"opChainConfigs\",\"type\":\"tuple[]\"}],\"internalType\":\"structOPContractsManagerInteropMigrator.MigrateInput\",\"name\":\"_input\",\"type\":\"tuple\"}],\"name\":\"migrate\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"opcmDeployer\",\"outputs\":[{\"internalType\":\"contractOPContractsManagerDeployer\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"opcmGameTypeAdder\",\"outputs\":[{\"internalType\":\"contractOPContractsManagerGameTypeAdder\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"opcmInteropMigrator\",\"outputs\":[{\"internalType\":\"contractOPContractsManagerInteropMigrator\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"opcmUpgrader\",\"outputs\":[{\"internalType\":\"contractOPContractsManagerUpgrader\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"protocolVersions\",\"outputs\":[{\"internalType\":\"contractIProtocolVersions\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bool\",\"name\":\"_isRC\",\"type\":\"bool\"}],\"name\":\"setRC\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"superchainConfig\",\"outputs\":[{\"internalType\":\"contractISuperchainConfig\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"superchainProxyAdmin\",\"outputs\":[{\"internalType\":\"contractIProxyAdmin\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"contractISystemConfig\",\"name\":\"systemConfigProxy\",\"type\":\"address\"},{\"internalType\":\"contractIProxyAdmin\",\"name\":\"proxyAdmin\",\"type\":\"address\"},{\"internalType\":\"Claim\",\"name\":\"absolutePrestate\",\"type\":\"bytes32\"}],\"internalType\":\"structOPContractsManager.OpChainConfig[]\",\"name\":\"_prestateUpdateInputs\",\"type\":\"tuple[]\"}],\"name\":\"updatePrestate\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"contractISystemConfig\",\"name\":\"systemConfigProxy\",\"type\":\"address\"},{\"internalType\":\"contractIProxyAdmin\",\"name\":\"proxyAdmin\",\"type\":\"address\"},{\"internalType\":\"Claim\",\"name\":\"absolutePrestate\",\"type\":\"bytes32\"}],\"internalType\":\"structOPContractsManager.OpChainConfig[]\",\"name\":\"_opChainConfigs\",\"type\":\"tuple[]\"}],\"name\":\"upgrade\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"upgradeController\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"version\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"who\",\"type\":\"address\"}],\"name\":\"AddressHasNoCode\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"who\",\"type\":\"address\"}],\"name\":\"AddressNotFound\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"AlreadyReleased\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"InvalidChainId\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"InvalidGameConfigs\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"string\",\"name\":\"role\",\"type\":\"string\"}],\"name\":\"InvalidRoleAddress\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"InvalidStartingAnchorRoot\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"LatestReleaseNotSet\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"OnlyDelegatecall\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"OnlyUpgradeController\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"PrestateNotSet\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"PrestateRequired\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"contractISystemConfig\",\"name\":\"systemConfig\",\"type\":\"address\"}],\"name\":\"SuperchainConfigMismatch\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"SuperchainProxyAdminMismatch\",\"type\":\"error\"}]",
}

// OPContractsManagerABI is the input ABI used to generate the binding from.
// Deprecated: Use OPContractsManagerMetaData.ABI instead.
var OPContractsManagerABI = OPContractsManagerMetaData.ABI

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

// IsRC is a free data retrieval call binding the contract method 0xf179c48d.
//
// Solidity: function isRC() view returns(bool)
func (_OPContractsManager *OPContractsManagerCaller) IsRC(opts *bind.CallOpts) (bool, error) {
	var out []interface{}
	err := _OPContractsManager.contract.Call(opts, &out, "isRC")

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsRC is a free data retrieval call binding the contract method 0xf179c48d.
//
// Solidity: function isRC() view returns(bool)
func (_OPContractsManager *OPContractsManagerSession) IsRC() (bool, error) {
	return _OPContractsManager.Contract.IsRC(&_OPContractsManager.CallOpts)
}

// IsRC is a free data retrieval call binding the contract method 0xf179c48d.
//
// Solidity: function isRC() view returns(bool)
func (_OPContractsManager *OPContractsManagerCallerSession) IsRC() (bool, error) {
	return _OPContractsManager.Contract.IsRC(&_OPContractsManager.CallOpts)
}

// L1ContractsRelease is a free data retrieval call binding the contract method 0x35cb2e9b.
//
// Solidity: function l1ContractsRelease() view returns(string)
func (_OPContractsManager *OPContractsManagerCaller) L1ContractsRelease(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _OPContractsManager.contract.Call(opts, &out, "l1ContractsRelease")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// L1ContractsRelease is a free data retrieval call binding the contract method 0x35cb2e9b.
//
// Solidity: function l1ContractsRelease() view returns(string)
func (_OPContractsManager *OPContractsManagerSession) L1ContractsRelease() (string, error) {
	return _OPContractsManager.Contract.L1ContractsRelease(&_OPContractsManager.CallOpts)
}

// L1ContractsRelease is a free data retrieval call binding the contract method 0x35cb2e9b.
//
// Solidity: function l1ContractsRelease() view returns(string)
func (_OPContractsManager *OPContractsManagerCallerSession) L1ContractsRelease() (string, error) {
	return _OPContractsManager.Contract.L1ContractsRelease(&_OPContractsManager.CallOpts)
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

// SetRC is a paid mutator transaction binding the contract method 0x6ccdfe11.
//
// Solidity: function setRC(bool _isRC) returns()
func (_OPContractsManager *OPContractsManagerTransactor) SetRC(opts *bind.TransactOpts, _isRC bool) (*types.Transaction, error) {
	return _OPContractsManager.contract.Transact(opts, "setRC", _isRC)
}

// SetRC is a paid mutator transaction binding the contract method 0x6ccdfe11.
//
// Solidity: function setRC(bool _isRC) returns()
func (_OPContractsManager *OPContractsManagerSession) SetRC(_isRC bool) (*types.Transaction, error) {
	return _OPContractsManager.Contract.SetRC(&_OPContractsManager.TransactOpts, _isRC)
}

// SetRC is a paid mutator transaction binding the contract method 0x6ccdfe11.
//
// Solidity: function setRC(bool _isRC) returns()
func (_OPContractsManager *OPContractsManagerTransactorSession) SetRC(_isRC bool) (*types.Transaction, error) {
	return _OPContractsManager.Contract.SetRC(&_OPContractsManager.TransactOpts, _isRC)
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
