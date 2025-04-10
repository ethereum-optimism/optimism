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
	AddressManager             common.Address
	Proxy                      common.Address
	ProxyAdmin                 common.Address
	L1ChugSplashProxy          common.Address
	ResolvedDelegateProxy      common.Address
	PermissionedDisputeGame1   common.Address
	PermissionedDisputeGame2   common.Address
	PermissionlessDisputeGame1 common.Address
	PermissionlessDisputeGame2 common.Address
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
	SystemConfigImpl                 common.Address
	OptimismMintableERC20FactoryImpl common.Address
	L1CrossDomainMessengerImpl       common.Address
	L1StandardBridgeImpl             common.Address
	DisputeGameFactoryImpl           common.Address
	AnchorStateRegistryImpl          common.Address
	DelayedWETHImpl                  common.Address
	MipsImpl                         common.Address
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

// Opcm200MetaData contains all meta data concerning the Opcm200 contract.
var Opcm200MetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"contractISuperchainConfig\",\"name\":\"_superchainConfig\",\"type\":\"address\"},{\"internalType\":\"contractIProtocolVersions\",\"name\":\"_protocolVersions\",\"type\":\"address\"},{\"internalType\":\"contractIProxyAdmin\",\"name\":\"_superchainProxyAdmin\",\"type\":\"address\"},{\"internalType\":\"string\",\"name\":\"_l1ContractsRelease\",\"type\":\"string\"},{\"components\":[{\"internalType\":\"address\",\"name\":\"addressManager\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"proxy\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"proxyAdmin\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"l1ChugSplashProxy\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"resolvedDelegateProxy\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"permissionedDisputeGame1\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"permissionedDisputeGame2\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"permissionlessDisputeGame1\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"permissionlessDisputeGame2\",\"type\":\"address\"}],\"internalType\":\"structOPContractsManager.Blueprints\",\"name\":\"_blueprints\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"address\",\"name\":\"superchainConfigImpl\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"protocolVersionsImpl\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"l1ERC721BridgeImpl\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"optimismPortalImpl\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"systemConfigImpl\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"optimismMintableERC20FactoryImpl\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"l1CrossDomainMessengerImpl\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"l1StandardBridgeImpl\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"disputeGameFactoryImpl\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"anchorStateRegistryImpl\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"delayedWETHImpl\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"mipsImpl\",\"type\":\"address\"}],\"internalType\":\"structOPContractsManager.Implementations\",\"name\":\"_implementations\",\"type\":\"tuple\"},{\"internalType\":\"address\",\"name\":\"_upgradeController\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"inputs\":[{\"components\":[{\"internalType\":\"string\",\"name\":\"saltMixer\",\"type\":\"string\"},{\"internalType\":\"contractISystemConfig\",\"name\":\"systemConfig\",\"type\":\"address\"},{\"internalType\":\"contractIProxyAdmin\",\"name\":\"proxyAdmin\",\"type\":\"address\"},{\"internalType\":\"contractIDelayedWETH\",\"name\":\"delayedWETH\",\"type\":\"address\"},{\"internalType\":\"GameType\",\"name\":\"disputeGameType\",\"type\":\"uint32\"},{\"internalType\":\"Claim\",\"name\":\"disputeAbsolutePrestate\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"disputeMaxGameDepth\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"disputeSplitDepth\",\"type\":\"uint256\"},{\"internalType\":\"Duration\",\"name\":\"disputeClockExtension\",\"type\":\"uint64\"},{\"internalType\":\"Duration\",\"name\":\"disputeMaxClockDuration\",\"type\":\"uint64\"},{\"internalType\":\"uint256\",\"name\":\"initialBond\",\"type\":\"uint256\"},{\"internalType\":\"contractIBigStepper\",\"name\":\"vm\",\"type\":\"address\"},{\"internalType\":\"bool\",\"name\":\"permissioned\",\"type\":\"bool\"}],\"internalType\":\"structOPContractsManager.AddGameInput[]\",\"name\":\"_gameConfigs\",\"type\":\"tuple[]\"}],\"name\":\"addGameType\",\"outputs\":[{\"components\":[{\"internalType\":\"contractIDelayedWETH\",\"name\":\"delayedWETH\",\"type\":\"address\"},{\"internalType\":\"contractIFaultDisputeGame\",\"name\":\"faultDisputeGame\",\"type\":\"address\"}],\"internalType\":\"structOPContractsManager.AddGameOutput[]\",\"name\":\"\",\"type\":\"tuple[]\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"blueprints\",\"outputs\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"addressManager\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"proxy\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"proxyAdmin\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"l1ChugSplashProxy\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"resolvedDelegateProxy\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"permissionedDisputeGame1\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"permissionedDisputeGame2\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"permissionlessDisputeGame1\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"permissionlessDisputeGame2\",\"type\":\"address\"}],\"internalType\":\"structOPContractsManager.Blueprints\",\"name\":\"\",\"type\":\"tuple\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_l2ChainId\",\"type\":\"uint256\"}],\"name\":\"chainIdToBatchInboxAddress\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"opChainProxyAdminOwner\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"systemConfigOwner\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"batcher\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"unsafeBlockSigner\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"proposer\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"challenger\",\"type\":\"address\"}],\"internalType\":\"structOPContractsManager.Roles\",\"name\":\"roles\",\"type\":\"tuple\"},{\"internalType\":\"uint32\",\"name\":\"basefeeScalar\",\"type\":\"uint32\"},{\"internalType\":\"uint32\",\"name\":\"blobBasefeeScalar\",\"type\":\"uint32\"},{\"internalType\":\"uint256\",\"name\":\"l2ChainId\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"startingAnchorRoot\",\"type\":\"bytes\"},{\"internalType\":\"string\",\"name\":\"saltMixer\",\"type\":\"string\"},{\"internalType\":\"uint64\",\"name\":\"gasLimit\",\"type\":\"uint64\"},{\"internalType\":\"GameType\",\"name\":\"disputeGameType\",\"type\":\"uint32\"},{\"internalType\":\"Claim\",\"name\":\"disputeAbsolutePrestate\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"disputeMaxGameDepth\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"disputeSplitDepth\",\"type\":\"uint256\"},{\"internalType\":\"Duration\",\"name\":\"disputeClockExtension\",\"type\":\"uint64\"},{\"internalType\":\"Duration\",\"name\":\"disputeMaxClockDuration\",\"type\":\"uint64\"}],\"internalType\":\"structOPContractsManager.DeployInput\",\"name\":\"_input\",\"type\":\"tuple\"}],\"name\":\"deploy\",\"outputs\":[{\"components\":[{\"internalType\":\"contractIProxyAdmin\",\"name\":\"opChainProxyAdmin\",\"type\":\"address\"},{\"internalType\":\"contractIAddressManager\",\"name\":\"addressManager\",\"type\":\"address\"},{\"internalType\":\"contractIL1ERC721Bridge\",\"name\":\"l1ERC721BridgeProxy\",\"type\":\"address\"},{\"internalType\":\"contractISystemConfig\",\"name\":\"systemConfigProxy\",\"type\":\"address\"},{\"internalType\":\"contractIOptimismMintableERC20Factory\",\"name\":\"optimismMintableERC20FactoryProxy\",\"type\":\"address\"},{\"internalType\":\"contractIL1StandardBridge\",\"name\":\"l1StandardBridgeProxy\",\"type\":\"address\"},{\"internalType\":\"contractIL1CrossDomainMessenger\",\"name\":\"l1CrossDomainMessengerProxy\",\"type\":\"address\"},{\"internalType\":\"contractIOptimismPortal2\",\"name\":\"optimismPortalProxy\",\"type\":\"address\"},{\"internalType\":\"contractIDisputeGameFactory\",\"name\":\"disputeGameFactoryProxy\",\"type\":\"address\"},{\"internalType\":\"contractIAnchorStateRegistry\",\"name\":\"anchorStateRegistryProxy\",\"type\":\"address\"},{\"internalType\":\"contractIFaultDisputeGame\",\"name\":\"faultDisputeGame\",\"type\":\"address\"},{\"internalType\":\"contractIPermissionedDisputeGame\",\"name\":\"permissionedDisputeGame\",\"type\":\"address\"},{\"internalType\":\"contractIDelayedWETH\",\"name\":\"delayedWETHPermissionedGameProxy\",\"type\":\"address\"},{\"internalType\":\"contractIDelayedWETH\",\"name\":\"delayedWETHPermissionlessGameProxy\",\"type\":\"address\"}],\"internalType\":\"structOPContractsManager.DeployOutput\",\"name\":\"\",\"type\":\"tuple\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"implementations\",\"outputs\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"superchainConfigImpl\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"protocolVersionsImpl\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"l1ERC721BridgeImpl\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"optimismPortalImpl\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"systemConfigImpl\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"optimismMintableERC20FactoryImpl\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"l1CrossDomainMessengerImpl\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"l1StandardBridgeImpl\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"disputeGameFactoryImpl\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"anchorStateRegistryImpl\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"delayedWETHImpl\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"mipsImpl\",\"type\":\"address\"}],\"internalType\":\"structOPContractsManager.Implementations\",\"name\":\"\",\"type\":\"tuple\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"isRC\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"l1ContractsRelease\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"protocolVersions\",\"outputs\":[{\"internalType\":\"contractIProtocolVersions\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bool\",\"name\":\"_isRC\",\"type\":\"bool\"}],\"name\":\"setRC\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"superchainConfig\",\"outputs\":[{\"internalType\":\"contractISuperchainConfig\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"superchainProxyAdmin\",\"outputs\":[{\"internalType\":\"contractIProxyAdmin\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"contractISystemConfig\",\"name\":\"systemConfigProxy\",\"type\":\"address\"},{\"internalType\":\"contractIProxyAdmin\",\"name\":\"proxyAdmin\",\"type\":\"address\"},{\"internalType\":\"Claim\",\"name\":\"absolutePrestate\",\"type\":\"bytes32\"}],\"internalType\":\"structOPContractsManager.OpChainConfig[]\",\"name\":\"_opChainConfigs\",\"type\":\"tuple[]\"}],\"name\":\"upgrade\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"upgradeController\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"version\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"l2ChainId\",\"type\":\"uint256\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"deployer\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"deployOutput\",\"type\":\"bytes\"}],\"name\":\"Deployed\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"l2ChainId\",\"type\":\"uint256\"},{\"indexed\":true,\"internalType\":\"GameType\",\"name\":\"gameType\",\"type\":\"uint32\"},{\"indexed\":false,\"internalType\":\"contractIDisputeGame\",\"name\":\"newDisputeGame\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"contractIDisputeGame\",\"name\":\"oldDisputeGame\",\"type\":\"address\"}],\"name\":\"GameTypeAdded\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"l2ChainId\",\"type\":\"uint256\"},{\"indexed\":true,\"internalType\":\"contractISystemConfig\",\"name\":\"systemConfig\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"upgrader\",\"type\":\"address\"}],\"name\":\"Upgraded\",\"type\":\"event\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"who\",\"type\":\"address\"}],\"name\":\"AddressHasNoCode\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"who\",\"type\":\"address\"}],\"name\":\"AddressNotFound\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"AlreadyReleased\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"BytesArrayTooLong\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"DeploymentFailed\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"EmptyInitcode\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"IdentityPrecompileCallFailed\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"InvalidChainId\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"InvalidGameConfigs\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"string\",\"name\":\"role\",\"type\":\"string\"}],\"name\":\"InvalidRoleAddress\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"InvalidStartingAnchorRoot\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"LatestReleaseNotSet\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"NotABlueprint\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"OnlyDelegatecall\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"OnlyUpgradeController\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"PrestateNotSet\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"ReservedBitsSet\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"contractISystemConfig\",\"name\":\"systemConfig\",\"type\":\"address\"}],\"name\":\"SuperchainConfigMismatch\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"SuperchainProxyAdminMismatch\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"UnexpectedPreambleData\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"uint8\",\"name\":\"version\",\"type\":\"uint8\"}],\"name\":\"UnsupportedERCVersion\",\"type\":\"error\"}]",
}

// Opcm200ABI is the input ABI used to generate the binding from.
// Deprecated: Use Opcm200MetaData.ABI instead.
var Opcm200ABI = Opcm200MetaData.ABI

// Opcm200 is an auto generated Go binding around an Ethereum contract.
type Opcm200 struct {
	Opcm200Caller     // Read-only binding to the contract
	Opcm200Transactor // Write-only binding to the contract
	Opcm200Filterer   // Log filterer for contract events
}

// Opcm200Caller is an auto generated read-only Go binding around an Ethereum contract.
type Opcm200Caller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// Opcm200Transactor is an auto generated write-only Go binding around an Ethereum contract.
type Opcm200Transactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// Opcm200Filterer is an auto generated log filtering Go binding around an Ethereum contract events.
type Opcm200Filterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// Opcm200Session is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type Opcm200Session struct {
	Contract     *Opcm200          // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// Opcm200CallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type Opcm200CallerSession struct {
	Contract *Opcm200Caller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts  // Call options to use throughout this session
}

// Opcm200TransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type Opcm200TransactorSession struct {
	Contract     *Opcm200Transactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts  // Transaction auth options to use throughout this session
}

// Opcm200Raw is an auto generated low-level Go binding around an Ethereum contract.
type Opcm200Raw struct {
	Contract *Opcm200 // Generic contract binding to access the raw methods on
}

// Opcm200CallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type Opcm200CallerRaw struct {
	Contract *Opcm200Caller // Generic read-only contract binding to access the raw methods on
}

// Opcm200TransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type Opcm200TransactorRaw struct {
	Contract *Opcm200Transactor // Generic write-only contract binding to access the raw methods on
}

// NewOpcm200 creates a new instance of Opcm200, bound to a specific deployed contract.
func NewOpcm200(address common.Address, backend bind.ContractBackend) (*Opcm200, error) {
	contract, err := bindOpcm200(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Opcm200{Opcm200Caller: Opcm200Caller{contract: contract}, Opcm200Transactor: Opcm200Transactor{contract: contract}, Opcm200Filterer: Opcm200Filterer{contract: contract}}, nil
}

// NewOpcm200Caller creates a new read-only instance of Opcm200, bound to a specific deployed contract.
func NewOpcm200Caller(address common.Address, caller bind.ContractCaller) (*Opcm200Caller, error) {
	contract, err := bindOpcm200(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &Opcm200Caller{contract: contract}, nil
}

// NewOpcm200Transactor creates a new write-only instance of Opcm200, bound to a specific deployed contract.
func NewOpcm200Transactor(address common.Address, transactor bind.ContractTransactor) (*Opcm200Transactor, error) {
	contract, err := bindOpcm200(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &Opcm200Transactor{contract: contract}, nil
}

// NewOpcm200Filterer creates a new log filterer instance of Opcm200, bound to a specific deployed contract.
func NewOpcm200Filterer(address common.Address, filterer bind.ContractFilterer) (*Opcm200Filterer, error) {
	contract, err := bindOpcm200(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &Opcm200Filterer{contract: contract}, nil
}

// bindOpcm200 binds a generic wrapper to an already deployed contract.
func bindOpcm200(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(Opcm200ABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Opcm200 *Opcm200Raw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Opcm200.Contract.Opcm200Caller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Opcm200 *Opcm200Raw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Opcm200.Contract.Opcm200Transactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Opcm200 *Opcm200Raw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Opcm200.Contract.Opcm200Transactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Opcm200 *Opcm200CallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Opcm200.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Opcm200 *Opcm200TransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Opcm200.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Opcm200 *Opcm200TransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Opcm200.Contract.contract.Transact(opts, method, params...)
}

// Blueprints is a free data retrieval call binding the contract method 0xb51f9c2b.
//
// Solidity: function blueprints() view returns((address,address,address,address,address,address,address,address,address))
func (_Opcm200 *Opcm200Caller) Blueprints(opts *bind.CallOpts) (OPContractsManagerBlueprints, error) {
	var out []interface{}
	err := _Opcm200.contract.Call(opts, &out, "blueprints")

	if err != nil {
		return *new(OPContractsManagerBlueprints), err
	}

	out0 := *abi.ConvertType(out[0], new(OPContractsManagerBlueprints)).(*OPContractsManagerBlueprints)

	return out0, err

}

// Blueprints is a free data retrieval call binding the contract method 0xb51f9c2b.
//
// Solidity: function blueprints() view returns((address,address,address,address,address,address,address,address,address))
func (_Opcm200 *Opcm200Session) Blueprints() (OPContractsManagerBlueprints, error) {
	return _Opcm200.Contract.Blueprints(&_Opcm200.CallOpts)
}

// Blueprints is a free data retrieval call binding the contract method 0xb51f9c2b.
//
// Solidity: function blueprints() view returns((address,address,address,address,address,address,address,address,address))
func (_Opcm200 *Opcm200CallerSession) Blueprints() (OPContractsManagerBlueprints, error) {
	return _Opcm200.Contract.Blueprints(&_Opcm200.CallOpts)
}

// ChainIdToBatchInboxAddress is a free data retrieval call binding the contract method 0x318b1b80.
//
// Solidity: function chainIdToBatchInboxAddress(uint256 _l2ChainId) pure returns(address)
func (_Opcm200 *Opcm200Caller) ChainIdToBatchInboxAddress(opts *bind.CallOpts, _l2ChainId *big.Int) (common.Address, error) {
	var out []interface{}
	err := _Opcm200.contract.Call(opts, &out, "chainIdToBatchInboxAddress", _l2ChainId)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// ChainIdToBatchInboxAddress is a free data retrieval call binding the contract method 0x318b1b80.
//
// Solidity: function chainIdToBatchInboxAddress(uint256 _l2ChainId) pure returns(address)
func (_Opcm200 *Opcm200Session) ChainIdToBatchInboxAddress(_l2ChainId *big.Int) (common.Address, error) {
	return _Opcm200.Contract.ChainIdToBatchInboxAddress(&_Opcm200.CallOpts, _l2ChainId)
}

// ChainIdToBatchInboxAddress is a free data retrieval call binding the contract method 0x318b1b80.
//
// Solidity: function chainIdToBatchInboxAddress(uint256 _l2ChainId) pure returns(address)
func (_Opcm200 *Opcm200CallerSession) ChainIdToBatchInboxAddress(_l2ChainId *big.Int) (common.Address, error) {
	return _Opcm200.Contract.ChainIdToBatchInboxAddress(&_Opcm200.CallOpts, _l2ChainId)
}

// Implementations is a free data retrieval call binding the contract method 0x30e9012c.
//
// Solidity: function implementations() view returns((address,address,address,address,address,address,address,address,address,address,address,address))
func (_Opcm200 *Opcm200Caller) Implementations(opts *bind.CallOpts) (OPContractsManagerImplementations, error) {
	var out []interface{}
	err := _Opcm200.contract.Call(opts, &out, "implementations")

	if err != nil {
		return *new(OPContractsManagerImplementations), err
	}

	out0 := *abi.ConvertType(out[0], new(OPContractsManagerImplementations)).(*OPContractsManagerImplementations)

	return out0, err

}

// Implementations is a free data retrieval call binding the contract method 0x30e9012c.
//
// Solidity: function implementations() view returns((address,address,address,address,address,address,address,address,address,address,address,address))
func (_Opcm200 *Opcm200Session) Implementations() (OPContractsManagerImplementations, error) {
	return _Opcm200.Contract.Implementations(&_Opcm200.CallOpts)
}

// Implementations is a free data retrieval call binding the contract method 0x30e9012c.
//
// Solidity: function implementations() view returns((address,address,address,address,address,address,address,address,address,address,address,address))
func (_Opcm200 *Opcm200CallerSession) Implementations() (OPContractsManagerImplementations, error) {
	return _Opcm200.Contract.Implementations(&_Opcm200.CallOpts)
}

// IsRC is a free data retrieval call binding the contract method 0xf179c48d.
//
// Solidity: function isRC() view returns(bool)
func (_Opcm200 *Opcm200Caller) IsRC(opts *bind.CallOpts) (bool, error) {
	var out []interface{}
	err := _Opcm200.contract.Call(opts, &out, "isRC")

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsRC is a free data retrieval call binding the contract method 0xf179c48d.
//
// Solidity: function isRC() view returns(bool)
func (_Opcm200 *Opcm200Session) IsRC() (bool, error) {
	return _Opcm200.Contract.IsRC(&_Opcm200.CallOpts)
}

// IsRC is a free data retrieval call binding the contract method 0xf179c48d.
//
// Solidity: function isRC() view returns(bool)
func (_Opcm200 *Opcm200CallerSession) IsRC() (bool, error) {
	return _Opcm200.Contract.IsRC(&_Opcm200.CallOpts)
}

// L1ContractsRelease is a free data retrieval call binding the contract method 0x35cb2e9b.
//
// Solidity: function l1ContractsRelease() view returns(string)
func (_Opcm200 *Opcm200Caller) L1ContractsRelease(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _Opcm200.contract.Call(opts, &out, "l1ContractsRelease")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// L1ContractsRelease is a free data retrieval call binding the contract method 0x35cb2e9b.
//
// Solidity: function l1ContractsRelease() view returns(string)
func (_Opcm200 *Opcm200Session) L1ContractsRelease() (string, error) {
	return _Opcm200.Contract.L1ContractsRelease(&_Opcm200.CallOpts)
}

// L1ContractsRelease is a free data retrieval call binding the contract method 0x35cb2e9b.
//
// Solidity: function l1ContractsRelease() view returns(string)
func (_Opcm200 *Opcm200CallerSession) L1ContractsRelease() (string, error) {
	return _Opcm200.Contract.L1ContractsRelease(&_Opcm200.CallOpts)
}

// ProtocolVersions is a free data retrieval call binding the contract method 0x6624856a.
//
// Solidity: function protocolVersions() view returns(address)
func (_Opcm200 *Opcm200Caller) ProtocolVersions(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _Opcm200.contract.Call(opts, &out, "protocolVersions")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// ProtocolVersions is a free data retrieval call binding the contract method 0x6624856a.
//
// Solidity: function protocolVersions() view returns(address)
func (_Opcm200 *Opcm200Session) ProtocolVersions() (common.Address, error) {
	return _Opcm200.Contract.ProtocolVersions(&_Opcm200.CallOpts)
}

// ProtocolVersions is a free data retrieval call binding the contract method 0x6624856a.
//
// Solidity: function protocolVersions() view returns(address)
func (_Opcm200 *Opcm200CallerSession) ProtocolVersions() (common.Address, error) {
	return _Opcm200.Contract.ProtocolVersions(&_Opcm200.CallOpts)
}

// SuperchainConfig is a free data retrieval call binding the contract method 0x35e80ab3.
//
// Solidity: function superchainConfig() view returns(address)
func (_Opcm200 *Opcm200Caller) SuperchainConfig(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _Opcm200.contract.Call(opts, &out, "superchainConfig")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// SuperchainConfig is a free data retrieval call binding the contract method 0x35e80ab3.
//
// Solidity: function superchainConfig() view returns(address)
func (_Opcm200 *Opcm200Session) SuperchainConfig() (common.Address, error) {
	return _Opcm200.Contract.SuperchainConfig(&_Opcm200.CallOpts)
}

// SuperchainConfig is a free data retrieval call binding the contract method 0x35e80ab3.
//
// Solidity: function superchainConfig() view returns(address)
func (_Opcm200 *Opcm200CallerSession) SuperchainConfig() (common.Address, error) {
	return _Opcm200.Contract.SuperchainConfig(&_Opcm200.CallOpts)
}

// SuperchainProxyAdmin is a free data retrieval call binding the contract method 0x2b96b839.
//
// Solidity: function superchainProxyAdmin() view returns(address)
func (_Opcm200 *Opcm200Caller) SuperchainProxyAdmin(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _Opcm200.contract.Call(opts, &out, "superchainProxyAdmin")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// SuperchainProxyAdmin is a free data retrieval call binding the contract method 0x2b96b839.
//
// Solidity: function superchainProxyAdmin() view returns(address)
func (_Opcm200 *Opcm200Session) SuperchainProxyAdmin() (common.Address, error) {
	return _Opcm200.Contract.SuperchainProxyAdmin(&_Opcm200.CallOpts)
}

// SuperchainProxyAdmin is a free data retrieval call binding the contract method 0x2b96b839.
//
// Solidity: function superchainProxyAdmin() view returns(address)
func (_Opcm200 *Opcm200CallerSession) SuperchainProxyAdmin() (common.Address, error) {
	return _Opcm200.Contract.SuperchainProxyAdmin(&_Opcm200.CallOpts)
}

// UpgradeController is a free data retrieval call binding the contract method 0x87543ef6.
//
// Solidity: function upgradeController() view returns(address)
func (_Opcm200 *Opcm200Caller) UpgradeController(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _Opcm200.contract.Call(opts, &out, "upgradeController")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// UpgradeController is a free data retrieval call binding the contract method 0x87543ef6.
//
// Solidity: function upgradeController() view returns(address)
func (_Opcm200 *Opcm200Session) UpgradeController() (common.Address, error) {
	return _Opcm200.Contract.UpgradeController(&_Opcm200.CallOpts)
}

// UpgradeController is a free data retrieval call binding the contract method 0x87543ef6.
//
// Solidity: function upgradeController() view returns(address)
func (_Opcm200 *Opcm200CallerSession) UpgradeController() (common.Address, error) {
	return _Opcm200.Contract.UpgradeController(&_Opcm200.CallOpts)
}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() pure returns(string)
func (_Opcm200 *Opcm200Caller) Version(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _Opcm200.contract.Call(opts, &out, "version")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() pure returns(string)
func (_Opcm200 *Opcm200Session) Version() (string, error) {
	return _Opcm200.Contract.Version(&_Opcm200.CallOpts)
}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() pure returns(string)
func (_Opcm200 *Opcm200CallerSession) Version() (string, error) {
	return _Opcm200.Contract.Version(&_Opcm200.CallOpts)
}

// AddGameType is a paid mutator transaction binding the contract method 0x1661a2e9.
//
// Solidity: function addGameType((string,address,address,address,uint32,bytes32,uint256,uint256,uint64,uint64,uint256,address,bool)[] _gameConfigs) returns((address,address)[])
func (_Opcm200 *Opcm200Transactor) AddGameType(opts *bind.TransactOpts, _gameConfigs []OPContractsManagerAddGameInput) (*types.Transaction, error) {
	return _Opcm200.contract.Transact(opts, "addGameType", _gameConfigs)
}

// AddGameType is a paid mutator transaction binding the contract method 0x1661a2e9.
//
// Solidity: function addGameType((string,address,address,address,uint32,bytes32,uint256,uint256,uint64,uint64,uint256,address,bool)[] _gameConfigs) returns((address,address)[])
func (_Opcm200 *Opcm200Session) AddGameType(_gameConfigs []OPContractsManagerAddGameInput) (*types.Transaction, error) {
	return _Opcm200.Contract.AddGameType(&_Opcm200.TransactOpts, _gameConfigs)
}

// AddGameType is a paid mutator transaction binding the contract method 0x1661a2e9.
//
// Solidity: function addGameType((string,address,address,address,uint32,bytes32,uint256,uint256,uint64,uint64,uint256,address,bool)[] _gameConfigs) returns((address,address)[])
func (_Opcm200 *Opcm200TransactorSession) AddGameType(_gameConfigs []OPContractsManagerAddGameInput) (*types.Transaction, error) {
	return _Opcm200.Contract.AddGameType(&_Opcm200.TransactOpts, _gameConfigs)
}

// Deploy is a paid mutator transaction binding the contract method 0x613e827b.
//
// Solidity: function deploy(((address,address,address,address,address,address),uint32,uint32,uint256,bytes,string,uint64,uint32,bytes32,uint256,uint256,uint64,uint64) _input) returns((address,address,address,address,address,address,address,address,address,address,address,address,address,address))
func (_Opcm200 *Opcm200Transactor) Deploy(opts *bind.TransactOpts, _input OPContractsManagerDeployInput) (*types.Transaction, error) {
	return _Opcm200.contract.Transact(opts, "deploy", _input)
}

// Deploy is a paid mutator transaction binding the contract method 0x613e827b.
//
// Solidity: function deploy(((address,address,address,address,address,address),uint32,uint32,uint256,bytes,string,uint64,uint32,bytes32,uint256,uint256,uint64,uint64) _input) returns((address,address,address,address,address,address,address,address,address,address,address,address,address,address))
func (_Opcm200 *Opcm200Session) Deploy(_input OPContractsManagerDeployInput) (*types.Transaction, error) {
	return _Opcm200.Contract.Deploy(&_Opcm200.TransactOpts, _input)
}

// Deploy is a paid mutator transaction binding the contract method 0x613e827b.
//
// Solidity: function deploy(((address,address,address,address,address,address),uint32,uint32,uint256,bytes,string,uint64,uint32,bytes32,uint256,uint256,uint64,uint64) _input) returns((address,address,address,address,address,address,address,address,address,address,address,address,address,address))
func (_Opcm200 *Opcm200TransactorSession) Deploy(_input OPContractsManagerDeployInput) (*types.Transaction, error) {
	return _Opcm200.Contract.Deploy(&_Opcm200.TransactOpts, _input)
}

// SetRC is a paid mutator transaction binding the contract method 0x6ccdfe11.
//
// Solidity: function setRC(bool _isRC) returns()
func (_Opcm200 *Opcm200Transactor) SetRC(opts *bind.TransactOpts, _isRC bool) (*types.Transaction, error) {
	return _Opcm200.contract.Transact(opts, "setRC", _isRC)
}

// SetRC is a paid mutator transaction binding the contract method 0x6ccdfe11.
//
// Solidity: function setRC(bool _isRC) returns()
func (_Opcm200 *Opcm200Session) SetRC(_isRC bool) (*types.Transaction, error) {
	return _Opcm200.Contract.SetRC(&_Opcm200.TransactOpts, _isRC)
}

// SetRC is a paid mutator transaction binding the contract method 0x6ccdfe11.
//
// Solidity: function setRC(bool _isRC) returns()
func (_Opcm200 *Opcm200TransactorSession) SetRC(_isRC bool) (*types.Transaction, error) {
	return _Opcm200.Contract.SetRC(&_Opcm200.TransactOpts, _isRC)
}

// Upgrade is a paid mutator transaction binding the contract method 0xff2dd5a1.
//
// Solidity: function upgrade((address,address,bytes32)[] _opChainConfigs) returns()
func (_Opcm200 *Opcm200Transactor) Upgrade(opts *bind.TransactOpts, _opChainConfigs []OPContractsManagerOpChainConfig) (*types.Transaction, error) {
	return _Opcm200.contract.Transact(opts, "upgrade", _opChainConfigs)
}

// Upgrade is a paid mutator transaction binding the contract method 0xff2dd5a1.
//
// Solidity: function upgrade((address,address,bytes32)[] _opChainConfigs) returns()
func (_Opcm200 *Opcm200Session) Upgrade(_opChainConfigs []OPContractsManagerOpChainConfig) (*types.Transaction, error) {
	return _Opcm200.Contract.Upgrade(&_Opcm200.TransactOpts, _opChainConfigs)
}

// Upgrade is a paid mutator transaction binding the contract method 0xff2dd5a1.
//
// Solidity: function upgrade((address,address,bytes32)[] _opChainConfigs) returns()
func (_Opcm200 *Opcm200TransactorSession) Upgrade(_opChainConfigs []OPContractsManagerOpChainConfig) (*types.Transaction, error) {
	return _Opcm200.Contract.Upgrade(&_Opcm200.TransactOpts, _opChainConfigs)
}

// Opcm200DeployedIterator is returned from FilterDeployed and is used to iterate over the raw logs and unpacked data for Deployed events raised by the Opcm200 contract.
type Opcm200DeployedIterator struct {
	Event *Opcm200Deployed // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *Opcm200DeployedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(Opcm200Deployed)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(Opcm200Deployed)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *Opcm200DeployedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *Opcm200DeployedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// Opcm200Deployed represents a Deployed event raised by the Opcm200 contract.
type Opcm200Deployed struct {
	L2ChainId    *big.Int
	Deployer     common.Address
	DeployOutput []byte
	Raw          types.Log // Blockchain specific contextual infos
}

// FilterDeployed is a free log retrieval operation binding the contract event 0xb40fb1137b92aa97efb20f29c17d36c5947aac681c3315ba854b0232f8349542.
//
// Solidity: event Deployed(uint256 indexed l2ChainId, address indexed deployer, bytes deployOutput)
func (_Opcm200 *Opcm200Filterer) FilterDeployed(opts *bind.FilterOpts, l2ChainId []*big.Int, deployer []common.Address) (*Opcm200DeployedIterator, error) {

	var l2ChainIdRule []interface{}
	for _, l2ChainIdItem := range l2ChainId {
		l2ChainIdRule = append(l2ChainIdRule, l2ChainIdItem)
	}
	var deployerRule []interface{}
	for _, deployerItem := range deployer {
		deployerRule = append(deployerRule, deployerItem)
	}

	logs, sub, err := _Opcm200.contract.FilterLogs(opts, "Deployed", l2ChainIdRule, deployerRule)
	if err != nil {
		return nil, err
	}
	return &Opcm200DeployedIterator{contract: _Opcm200.contract, event: "Deployed", logs: logs, sub: sub}, nil
}

// WatchDeployed is a free log subscription operation binding the contract event 0xb40fb1137b92aa97efb20f29c17d36c5947aac681c3315ba854b0232f8349542.
//
// Solidity: event Deployed(uint256 indexed l2ChainId, address indexed deployer, bytes deployOutput)
func (_Opcm200 *Opcm200Filterer) WatchDeployed(opts *bind.WatchOpts, sink chan<- *Opcm200Deployed, l2ChainId []*big.Int, deployer []common.Address) (event.Subscription, error) {

	var l2ChainIdRule []interface{}
	for _, l2ChainIdItem := range l2ChainId {
		l2ChainIdRule = append(l2ChainIdRule, l2ChainIdItem)
	}
	var deployerRule []interface{}
	for _, deployerItem := range deployer {
		deployerRule = append(deployerRule, deployerItem)
	}

	logs, sub, err := _Opcm200.contract.WatchLogs(opts, "Deployed", l2ChainIdRule, deployerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(Opcm200Deployed)
				if err := _Opcm200.contract.UnpackLog(event, "Deployed", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseDeployed is a log parse operation binding the contract event 0xb40fb1137b92aa97efb20f29c17d36c5947aac681c3315ba854b0232f8349542.
//
// Solidity: event Deployed(uint256 indexed l2ChainId, address indexed deployer, bytes deployOutput)
func (_Opcm200 *Opcm200Filterer) ParseDeployed(log types.Log) (*Opcm200Deployed, error) {
	event := new(Opcm200Deployed)
	if err := _Opcm200.contract.UnpackLog(event, "Deployed", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// Opcm200GameTypeAddedIterator is returned from FilterGameTypeAdded and is used to iterate over the raw logs and unpacked data for GameTypeAdded events raised by the Opcm200 contract.
type Opcm200GameTypeAddedIterator struct {
	Event *Opcm200GameTypeAdded // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *Opcm200GameTypeAddedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(Opcm200GameTypeAdded)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(Opcm200GameTypeAdded)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *Opcm200GameTypeAddedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *Opcm200GameTypeAddedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// Opcm200GameTypeAdded represents a GameTypeAdded event raised by the Opcm200 contract.
type Opcm200GameTypeAdded struct {
	L2ChainId      *big.Int
	GameType       uint32
	NewDisputeGame common.Address
	OldDisputeGame common.Address
	Raw            types.Log // Blockchain specific contextual infos
}

// FilterGameTypeAdded is a free log retrieval operation binding the contract event 0x4b8d2d3f00ea4ebab553d99606c8aea67fd4deb9ef0abee0e7c4b246c59a0e0f.
//
// Solidity: event GameTypeAdded(uint256 indexed l2ChainId, uint32 indexed gameType, address newDisputeGame, address oldDisputeGame)
func (_Opcm200 *Opcm200Filterer) FilterGameTypeAdded(opts *bind.FilterOpts, l2ChainId []*big.Int, gameType []uint32) (*Opcm200GameTypeAddedIterator, error) {

	var l2ChainIdRule []interface{}
	for _, l2ChainIdItem := range l2ChainId {
		l2ChainIdRule = append(l2ChainIdRule, l2ChainIdItem)
	}
	var gameTypeRule []interface{}
	for _, gameTypeItem := range gameType {
		gameTypeRule = append(gameTypeRule, gameTypeItem)
	}

	logs, sub, err := _Opcm200.contract.FilterLogs(opts, "GameTypeAdded", l2ChainIdRule, gameTypeRule)
	if err != nil {
		return nil, err
	}
	return &Opcm200GameTypeAddedIterator{contract: _Opcm200.contract, event: "GameTypeAdded", logs: logs, sub: sub}, nil
}

// WatchGameTypeAdded is a free log subscription operation binding the contract event 0x4b8d2d3f00ea4ebab553d99606c8aea67fd4deb9ef0abee0e7c4b246c59a0e0f.
//
// Solidity: event GameTypeAdded(uint256 indexed l2ChainId, uint32 indexed gameType, address newDisputeGame, address oldDisputeGame)
func (_Opcm200 *Opcm200Filterer) WatchGameTypeAdded(opts *bind.WatchOpts, sink chan<- *Opcm200GameTypeAdded, l2ChainId []*big.Int, gameType []uint32) (event.Subscription, error) {

	var l2ChainIdRule []interface{}
	for _, l2ChainIdItem := range l2ChainId {
		l2ChainIdRule = append(l2ChainIdRule, l2ChainIdItem)
	}
	var gameTypeRule []interface{}
	for _, gameTypeItem := range gameType {
		gameTypeRule = append(gameTypeRule, gameTypeItem)
	}

	logs, sub, err := _Opcm200.contract.WatchLogs(opts, "GameTypeAdded", l2ChainIdRule, gameTypeRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(Opcm200GameTypeAdded)
				if err := _Opcm200.contract.UnpackLog(event, "GameTypeAdded", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseGameTypeAdded is a log parse operation binding the contract event 0x4b8d2d3f00ea4ebab553d99606c8aea67fd4deb9ef0abee0e7c4b246c59a0e0f.
//
// Solidity: event GameTypeAdded(uint256 indexed l2ChainId, uint32 indexed gameType, address newDisputeGame, address oldDisputeGame)
func (_Opcm200 *Opcm200Filterer) ParseGameTypeAdded(log types.Log) (*Opcm200GameTypeAdded, error) {
	event := new(Opcm200GameTypeAdded)
	if err := _Opcm200.contract.UnpackLog(event, "GameTypeAdded", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// Opcm200UpgradedIterator is returned from FilterUpgraded and is used to iterate over the raw logs and unpacked data for Upgraded events raised by the Opcm200 contract.
type Opcm200UpgradedIterator struct {
	Event *Opcm200Upgraded // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *Opcm200UpgradedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(Opcm200Upgraded)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(Opcm200Upgraded)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *Opcm200UpgradedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *Opcm200UpgradedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// Opcm200Upgraded represents a Upgraded event raised by the Opcm200 contract.
type Opcm200Upgraded struct {
	L2ChainId    *big.Int
	SystemConfig common.Address
	Upgrader     common.Address
	Raw          types.Log // Blockchain specific contextual infos
}

// FilterUpgraded is a free log retrieval operation binding the contract event 0x78bc67b9bf548ef6410becd31a3e10b9ea6c255974ef6b4530728b431df30030.
//
// Solidity: event Upgraded(uint256 indexed l2ChainId, address indexed systemConfig, address indexed upgrader)
func (_Opcm200 *Opcm200Filterer) FilterUpgraded(opts *bind.FilterOpts, l2ChainId []*big.Int, systemConfig []common.Address, upgrader []common.Address) (*Opcm200UpgradedIterator, error) {

	var l2ChainIdRule []interface{}
	for _, l2ChainIdItem := range l2ChainId {
		l2ChainIdRule = append(l2ChainIdRule, l2ChainIdItem)
	}
	var systemConfigRule []interface{}
	for _, systemConfigItem := range systemConfig {
		systemConfigRule = append(systemConfigRule, systemConfigItem)
	}
	var upgraderRule []interface{}
	for _, upgraderItem := range upgrader {
		upgraderRule = append(upgraderRule, upgraderItem)
	}

	logs, sub, err := _Opcm200.contract.FilterLogs(opts, "Upgraded", l2ChainIdRule, systemConfigRule, upgraderRule)
	if err != nil {
		return nil, err
	}
	return &Opcm200UpgradedIterator{contract: _Opcm200.contract, event: "Upgraded", logs: logs, sub: sub}, nil
}

// WatchUpgraded is a free log subscription operation binding the contract event 0x78bc67b9bf548ef6410becd31a3e10b9ea6c255974ef6b4530728b431df30030.
//
// Solidity: event Upgraded(uint256 indexed l2ChainId, address indexed systemConfig, address indexed upgrader)
func (_Opcm200 *Opcm200Filterer) WatchUpgraded(opts *bind.WatchOpts, sink chan<- *Opcm200Upgraded, l2ChainId []*big.Int, systemConfig []common.Address, upgrader []common.Address) (event.Subscription, error) {

	var l2ChainIdRule []interface{}
	for _, l2ChainIdItem := range l2ChainId {
		l2ChainIdRule = append(l2ChainIdRule, l2ChainIdItem)
	}
	var systemConfigRule []interface{}
	for _, systemConfigItem := range systemConfig {
		systemConfigRule = append(systemConfigRule, systemConfigItem)
	}
	var upgraderRule []interface{}
	for _, upgraderItem := range upgrader {
		upgraderRule = append(upgraderRule, upgraderItem)
	}

	logs, sub, err := _Opcm200.contract.WatchLogs(opts, "Upgraded", l2ChainIdRule, systemConfigRule, upgraderRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(Opcm200Upgraded)
				if err := _Opcm200.contract.UnpackLog(event, "Upgraded", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseUpgraded is a log parse operation binding the contract event 0x78bc67b9bf548ef6410becd31a3e10b9ea6c255974ef6b4530728b431df30030.
//
// Solidity: event Upgraded(uint256 indexed l2ChainId, address indexed systemConfig, address indexed upgrader)
func (_Opcm200 *Opcm200Filterer) ParseUpgraded(log types.Log) (*Opcm200Upgraded, error) {
	event := new(Opcm200Upgraded)
	if err := _Opcm200.contract.UnpackLog(event, "Upgraded", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
