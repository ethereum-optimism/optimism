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
	SysCfg           common.Address
	AbsolutePrestate [32]byte
	L2ChainID        *big.Int
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
	ABI: "[{\"type\":\"constructor\",\"inputs\":[{\"name\":\"_opcmGameTypeAdder\",\"type\":\"address\",\"internalType\":\"contractOPContractsManagerGameTypeAdder\"},{\"name\":\"_opcmDeployer\",\"type\":\"address\",\"internalType\":\"contractOPContractsManagerDeployer\"},{\"name\":\"_opcmUpgrader\",\"type\":\"address\",\"internalType\":\"contractOPContractsManagerUpgrader\"},{\"name\":\"_opcmInteropMigrator\",\"type\":\"address\",\"internalType\":\"contractOPContractsManagerInteropMigrator\"},{\"name\":\"_opcmStandardValidator\",\"type\":\"address\",\"internalType\":\"contractOPContractsManagerStandardValidator\"},{\"name\":\"_superchainConfig\",\"type\":\"address\",\"internalType\":\"contractISuperchainConfig\"},{\"name\":\"_protocolVersions\",\"type\":\"address\",\"internalType\":\"contractIProtocolVersions\"},{\"name\":\"_superchainProxyAdmin\",\"type\":\"address\",\"internalType\":\"contractIProxyAdmin\"}],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"addGameType\",\"inputs\":[{\"name\":\"_gameConfigs\",\"type\":\"tuple[]\",\"internalType\":\"structOPContractsManager.AddGameInput[]\",\"components\":[{\"name\":\"saltMixer\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"systemConfig\",\"type\":\"address\",\"internalType\":\"contractISystemConfig\"},{\"name\":\"delayedWETH\",\"type\":\"address\",\"internalType\":\"contractIDelayedWETH\"},{\"name\":\"disputeGameType\",\"type\":\"uint32\",\"internalType\":\"GameType\"},{\"name\":\"disputeAbsolutePrestate\",\"type\":\"bytes32\",\"internalType\":\"Claim\"},{\"name\":\"disputeMaxGameDepth\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"disputeSplitDepth\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"disputeClockExtension\",\"type\":\"uint64\",\"internalType\":\"Duration\"},{\"name\":\"disputeMaxClockDuration\",\"type\":\"uint64\",\"internalType\":\"Duration\"},{\"name\":\"initialBond\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"vm\",\"type\":\"address\",\"internalType\":\"contractIBigStepper\"},{\"name\":\"permissioned\",\"type\":\"bool\",\"internalType\":\"bool\"}]}],\"outputs\":[{\"name\":\"\",\"type\":\"tuple[]\",\"internalType\":\"structOPContractsManager.AddGameOutput[]\",\"components\":[{\"name\":\"delayedWETH\",\"type\":\"address\",\"internalType\":\"contractIDelayedWETH\"},{\"name\":\"faultDisputeGame\",\"type\":\"address\",\"internalType\":\"contractIFaultDisputeGame\"}]}],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"blueprints\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"tuple\",\"internalType\":\"structOPContractsManager.Blueprints\",\"components\":[{\"name\":\"addressManager\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"proxy\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"proxyAdmin\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"l1ChugSplashProxy\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"resolvedDelegateProxy\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"permissionedDisputeGame1\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"permissionedDisputeGame2\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"permissionlessDisputeGame1\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"permissionlessDisputeGame2\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"superPermissionedDisputeGame1\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"superPermissionedDisputeGame2\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"superPermissionlessDisputeGame1\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"superPermissionlessDisputeGame2\",\"type\":\"address\",\"internalType\":\"address\"}]}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"chainIdToBatchInboxAddress\",\"inputs\":[{\"name\":\"_l2ChainId\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"deploy\",\"inputs\":[{\"name\":\"_input\",\"type\":\"tuple\",\"internalType\":\"structOPContractsManager.DeployInput\",\"components\":[{\"name\":\"roles\",\"type\":\"tuple\",\"internalType\":\"structOPContractsManager.Roles\",\"components\":[{\"name\":\"opChainProxyAdminOwner\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"systemConfigOwner\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"batcher\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"unsafeBlockSigner\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"proposer\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"challenger\",\"type\":\"address\",\"internalType\":\"address\"}]},{\"name\":\"basefeeScalar\",\"type\":\"uint32\",\"internalType\":\"uint32\"},{\"name\":\"blobBasefeeScalar\",\"type\":\"uint32\",\"internalType\":\"uint32\"},{\"name\":\"l2ChainId\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"startingAnchorRoot\",\"type\":\"bytes\",\"internalType\":\"bytes\"},{\"name\":\"saltMixer\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"gasLimit\",\"type\":\"uint64\",\"internalType\":\"uint64\"},{\"name\":\"disputeGameType\",\"type\":\"uint32\",\"internalType\":\"GameType\"},{\"name\":\"disputeAbsolutePrestate\",\"type\":\"bytes32\",\"internalType\":\"Claim\"},{\"name\":\"disputeMaxGameDepth\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"disputeSplitDepth\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"disputeClockExtension\",\"type\":\"uint64\",\"internalType\":\"Duration\"},{\"name\":\"disputeMaxClockDuration\",\"type\":\"uint64\",\"internalType\":\"Duration\"}]}],\"outputs\":[{\"name\":\"\",\"type\":\"tuple\",\"internalType\":\"structOPContractsManager.DeployOutput\",\"components\":[{\"name\":\"opChainProxyAdmin\",\"type\":\"address\",\"internalType\":\"contractIProxyAdmin\"},{\"name\":\"addressManager\",\"type\":\"address\",\"internalType\":\"contractIAddressManager\"},{\"name\":\"l1ERC721BridgeProxy\",\"type\":\"address\",\"internalType\":\"contractIL1ERC721Bridge\"},{\"name\":\"systemConfigProxy\",\"type\":\"address\",\"internalType\":\"contractISystemConfig\"},{\"name\":\"optimismMintableERC20FactoryProxy\",\"type\":\"address\",\"internalType\":\"contractIOptimismMintableERC20Factory\"},{\"name\":\"l1StandardBridgeProxy\",\"type\":\"address\",\"internalType\":\"contractIL1StandardBridge\"},{\"name\":\"l1CrossDomainMessengerProxy\",\"type\":\"address\",\"internalType\":\"contractIL1CrossDomainMessenger\"},{\"name\":\"ethLockboxProxy\",\"type\":\"address\",\"internalType\":\"contractIETHLockbox\"},{\"name\":\"optimismPortalProxy\",\"type\":\"address\",\"internalType\":\"contractIOptimismPortal2\"},{\"name\":\"disputeGameFactoryProxy\",\"type\":\"address\",\"internalType\":\"contractIDisputeGameFactory\"},{\"name\":\"anchorStateRegistryProxy\",\"type\":\"address\",\"internalType\":\"contractIAnchorStateRegistry\"},{\"name\":\"faultDisputeGame\",\"type\":\"address\",\"internalType\":\"contractIFaultDisputeGame\"},{\"name\":\"permissionedDisputeGame\",\"type\":\"address\",\"internalType\":\"contractIPermissionedDisputeGame\"},{\"name\":\"delayedWETHPermissionedGameProxy\",\"type\":\"address\",\"internalType\":\"contractIDelayedWETH\"},{\"name\":\"delayedWETHPermissionlessGameProxy\",\"type\":\"address\",\"internalType\":\"contractIDelayedWETH\"}]}],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"devFeatureBitmap\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"implementations\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"tuple\",\"internalType\":\"structOPContractsManager.Implementations\",\"components\":[{\"name\":\"superchainConfigImpl\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"protocolVersionsImpl\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"l1ERC721BridgeImpl\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"optimismPortalImpl\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"optimismPortalInteropImpl\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"ethLockboxImpl\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"systemConfigImpl\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"optimismMintableERC20FactoryImpl\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"l1CrossDomainMessengerImpl\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"l1StandardBridgeImpl\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"disputeGameFactoryImpl\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"anchorStateRegistryImpl\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"delayedWETHImpl\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"mipsImpl\",\"type\":\"address\",\"internalType\":\"address\"}]}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"isDevFeatureEnabled\",\"inputs\":[{\"name\":\"_feature\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"outputs\":[{\"name\":\"\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"migrate\",\"inputs\":[{\"name\":\"_input\",\"type\":\"tuple\",\"internalType\":\"structOPContractsManagerInteropMigrator.MigrateInput\",\"components\":[{\"name\":\"usePermissionlessGame\",\"type\":\"bool\",\"internalType\":\"bool\"},{\"name\":\"startingAnchorRoot\",\"type\":\"tuple\",\"internalType\":\"structProposal\",\"components\":[{\"name\":\"root\",\"type\":\"bytes32\",\"internalType\":\"Hash\"},{\"name\":\"l2SequenceNumber\",\"type\":\"uint256\",\"internalType\":\"uint256\"}]},{\"name\":\"gameParameters\",\"type\":\"tuple\",\"internalType\":\"structOPContractsManagerInteropMigrator.GameParameters\",\"components\":[{\"name\":\"proposer\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"challenger\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"maxGameDepth\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"splitDepth\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"initBond\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"clockExtension\",\"type\":\"uint64\",\"internalType\":\"Duration\"},{\"name\":\"maxClockDuration\",\"type\":\"uint64\",\"internalType\":\"Duration\"}]},{\"name\":\"opChainConfigs\",\"type\":\"tuple[]\",\"internalType\":\"structOPContractsManager.OpChainConfig[]\",\"components\":[{\"name\":\"systemConfigProxy\",\"type\":\"address\",\"internalType\":\"contractISystemConfig\"},{\"name\":\"absolutePrestate\",\"type\":\"bytes32\",\"internalType\":\"Claim\"}]}]}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"opcmDeployer\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"contractOPContractsManagerDeployer\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"opcmGameTypeAdder\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"contractOPContractsManagerGameTypeAdder\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"opcmInteropMigrator\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"contractOPContractsManagerInteropMigrator\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"opcmStandardValidator\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"contractOPContractsManagerStandardValidator\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"opcmUpgrader\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"contractOPContractsManagerUpgrader\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"protocolVersions\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"contractIProtocolVersions\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"superchainConfig\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"contractISuperchainConfig\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"superchainProxyAdmin\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"contractIProxyAdmin\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"updatePrestate\",\"inputs\":[{\"name\":\"_prestateUpdateInputs\",\"type\":\"tuple[]\",\"internalType\":\"structOPContractsManager.UpdatePrestateInput[]\",\"components\":[{\"name\":\"systemConfigProxy\",\"type\":\"address\",\"internalType\":\"contractISystemConfig\"},{\"name\":\"cannonPrestate\",\"type\":\"bytes32\",\"internalType\":\"Claim\"},{\"name\":\"cannonKonaPrestate\",\"type\":\"bytes32\",\"internalType\":\"Claim\"}]}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"upgrade\",\"inputs\":[{\"name\":\"_opChainConfigs\",\"type\":\"tuple[]\",\"internalType\":\"structOPContractsManager.OpChainConfig[]\",\"components\":[{\"name\":\"systemConfigProxy\",\"type\":\"address\",\"internalType\":\"contractISystemConfig\"},{\"name\":\"absolutePrestate\",\"type\":\"bytes32\",\"internalType\":\"Claim\"}]}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"upgradeSuperchainConfig\",\"inputs\":[{\"name\":\"_superchainConfig\",\"type\":\"address\",\"internalType\":\"contractISuperchainConfig\"},{\"name\":\"_superchainProxyAdmin\",\"type\":\"address\",\"internalType\":\"contractIProxyAdmin\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"validate\",\"inputs\":[{\"name\":\"_input\",\"type\":\"tuple\",\"internalType\":\"structOPContractsManagerStandardValidator.ValidationInput\",\"components\":[{\"name\":\"sysCfg\",\"type\":\"address\",\"internalType\":\"contractISystemConfig\"},{\"name\":\"absolutePrestate\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"l2ChainID\",\"type\":\"uint256\",\"internalType\":\"uint256\"}]},{\"name\":\"_allowFailure\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"outputs\":[{\"name\":\"\",\"type\":\"string\",\"internalType\":\"string\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"validateWithOverrides\",\"inputs\":[{\"name\":\"_input\",\"type\":\"tuple\",\"internalType\":\"structOPContractsManagerStandardValidator.ValidationInput\",\"components\":[{\"name\":\"sysCfg\",\"type\":\"address\",\"internalType\":\"contractISystemConfig\"},{\"name\":\"absolutePrestate\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"l2ChainID\",\"type\":\"uint256\",\"internalType\":\"uint256\"}]},{\"name\":\"_allowFailure\",\"type\":\"bool\",\"internalType\":\"bool\"},{\"name\":\"_overrides\",\"type\":\"tuple\",\"internalType\":\"structOPContractsManagerStandardValidator.ValidationOverrides\",\"components\":[{\"name\":\"l1PAOMultisig\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"challenger\",\"type\":\"address\",\"internalType\":\"address\"}]}],\"outputs\":[{\"name\":\"\",\"type\":\"string\",\"internalType\":\"string\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"version\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"string\",\"internalType\":\"string\"}],\"stateMutability\":\"pure\"},{\"type\":\"error\",\"name\":\"AddressHasNoCode\",\"inputs\":[{\"name\":\"who\",\"type\":\"address\",\"internalType\":\"address\"}]},{\"type\":\"error\",\"name\":\"AddressNotFound\",\"inputs\":[{\"name\":\"who\",\"type\":\"address\",\"internalType\":\"address\"}]},{\"type\":\"error\",\"name\":\"AlreadyReleased\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"InvalidChainId\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"InvalidGameConfigs\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"InvalidRoleAddress\",\"inputs\":[{\"name\":\"role\",\"type\":\"string\",\"internalType\":\"string\"}]},{\"type\":\"error\",\"name\":\"InvalidStartingAnchorRoot\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"LatestReleaseNotSet\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"OnlyDelegatecall\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"PrestateNotSet\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"PrestateRequired\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"SuperchainConfigMismatch\",\"inputs\":[{\"name\":\"systemConfig\",\"type\":\"address\",\"internalType\":\"contractISystemConfig\"}]},{\"type\":\"error\",\"name\":\"SuperchainProxyAdminMismatch\",\"inputs\":[]}]",
	Bin: "0x6101a06040523480156200001257600080fd5b5060405162002c0538038062002c0583398101604081905262000035916200030c565b60405163b6a4cd2160e01b81526001600160a01b03848116600483015288169063b6a4cd219060240160006040518083038186803b1580156200007757600080fd5b505afa1580156200008c573d6000803e3d6000fd5b505060405163b6a4cd2160e01b81526001600160a01b0385811660048301528a16925063b6a4cd21915060240160006040518083038186803b158015620000d257600080fd5b505afa158015620000e7573d6000803e3d6000fd5b505060405163b6a4cd2160e01b81526001600160a01b038b811660048301528a16925063b6a4cd21915060240160006040518083038186803b1580156200012d57600080fd5b505afa15801562000142573d6000803e3d6000fd5b505060405163b6a4cd2160e01b81526001600160a01b038a1660048201819052925063b6a4cd21915060240160006040518083038186803b1580156200018757600080fd5b505afa1580156200019c573d6000803e3d6000fd5b505060405163b6a4cd2160e01b81526001600160a01b0389811660048301528a16925063b6a4cd21915060240160006040518083038186803b158015620001e257600080fd5b505afa158015620001f7573d6000803e3d6000fd5b505060405163b6a4cd2160e01b81526001600160a01b0388811660048301528a16925063b6a4cd21915060240160006040518083038186803b1580156200023d57600080fd5b505afa15801562000252573d6000803e3d6000fd5b505060405163b6a4cd2160e01b81526001600160a01b0387811660048301528a16925063b6a4cd21915060240160006040518083038186803b1580156200029857600080fd5b505afa158015620002ad573d6000803e3d6000fd5b5050506001600160a01b039889166080525095871660a05293861660c05291851660e05284166101005283166101205282166101405216610160523061018052620003cd565b6001600160a01b03811681146200030957600080fd5b50565b600080600080600080600080610100898b0312156200032a57600080fd5b88516200033781620002f3565b60208a01519098506200034a81620002f3565b60408a01519097506200035d81620002f3565b60608a01519096506200037081620002f3565b60808a01519095506200038381620002f3565b60a08a01519094506200039681620002f3565b60c08a0151909350620003a981620002f3565b60e08a0151909250620003bc81620002f3565b809150509295985092959890939650565b60805160a05160c05160e0516101005161012051610140516101605161018051612749620004bc600039600081816104520152818161077a015281816109d501528181610c420152610d4e0152600061022201526000610341015260008181610271015261095d0152600081816103ff01528181610bc20152610f970152600081816101e50152610a9f01526000818161018c0152818161051c0152610d1901526000818161031a0152818161054a01528181610642015281816106f50152818161092601528181610af60152610ea8015260008181610426015281816108460152610e1801526127496000f3fe608060405234801561001057600080fd5b50600436106101825760003560e01c8063622d56f1116100d8578063b0b807eb1161008c578063b806c80511610066578063b806c805146103e7578063ba7903db146103fa578063becbdf4a1461042157600080fd5b8063b0b807eb146103ac578063b23cc044146103bf578063b51f9c2b146103d257600080fd5b80636d510c5e116100bd5780636d510c5e1461036357806378ecabce14610376578063a9008b691461039957600080fd5b8063622d56f1146103155780636624856a1461033c57600080fd5b806330e9012c1161013a57806354fd4d501161011457806354fd4d5014610293578063604aa628146102d5578063613e827b146102f557600080fd5b806330e9012c14610244578063318b1b801461025957806335e80ab31461026c57600080fd5b80631481a7241161016b5780631481a724146101e05780631d8a4e92146102075780632b96b8391461021d57600080fd5b806303dbe68c146101875780630b8bd7cb146101cb575b600080fd5b6101ae7f000000000000000000000000000000000000000000000000000000000000000081565b6040516001600160a01b0390911681526020015b60405180910390f35b6101de6101d93660046111fd565b610448565b005b6101ae7f000000000000000000000000000000000000000000000000000000000000000081565b61020f610546565b6040519081526020016101c2565b6101ae7f000000000000000000000000000000000000000000000000000000000000000081565b61024c6105cf565b6040516101c291906112c1565b6101ae6102673660046113f5565b6106c3565b6101ae7f000000000000000000000000000000000000000000000000000000000000000081565b60408051808201909152600581527f342e312e3000000000000000000000000000000000000000000000000000000060208201525b6040516101c29190611466565b6102e86102e3366004611540565b61076e565b6040516101c291906116d5565b610308610303366004611731565b610889565b6040516101c2919061176d565b6101ae7f000000000000000000000000000000000000000000000000000000000000000081565b6101ae7f000000000000000000000000000000000000000000000000000000000000000081565b6101de6103713660046118aa565b6109cb565b6103896103843660046113f5565b610ac4565b60405190151581526020016101c2565b6102c86103a736600461192b565b610b69565b6101de6103ba366004611963565b610c38565b6101de6103cd366004611991565b610d44565b6103da610e3d565b6040516101c29190611a58565b6102c86103f5366004611b7b565b610f29565b6101ae7f000000000000000000000000000000000000000000000000000000000000000081565b6101ae7f000000000000000000000000000000000000000000000000000000000000000081565b6001600160a01b037f00000000000000000000000000000000000000000000000000000000000000001630036104aa576040517f0a57d61d00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b6000816040516024016104bd9190611c15565b60408051601f198184030181529190526020810180517bffffffffffffffffffffffffffffffffffffffffffffffffffffffff167f0b8bd7cb0000000000000000000000000000000000000000000000000000000017905290506105417f000000000000000000000000000000000000000000000000000000000000000082611006565b505050565b60007f00000000000000000000000000000000000000000000000000000000000000006001600160a01b0316631d8a4e926040518163ffffffff1660e01b8152600401602060405180830381865afa1580156105a6573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906105ca9190611c60565b905090565b604080516101c081018252600080825260208201819052918101829052606081018290526080810182905260a0810182905260c0810182905260e08101829052610100810182905261012081018290526101408101829052610160810182905261018081018290526101a08101919091527f00000000000000000000000000000000000000000000000000000000000000006001600160a01b03166330e9012c6040518163ffffffff1660e01b81526004016101c060405180830381865afa15801561069f573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906105ca9190611c84565b6040517f318b1b80000000000000000000000000000000000000000000000000000000008152600481018290526000907f00000000000000000000000000000000000000000000000000000000000000006001600160a01b03169063318b1b8090602401602060405180830381865afa158015610744573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906107689190611d94565b92915050565b60606001600160a01b037f00000000000000000000000000000000000000000000000000000000000000001630036107d2576040517f0a57d61d00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b6000826040516024016107e59190611db1565b60408051601f198184030181529190526020810180517bffffffffffffffffffffffffffffffffffffffffffffffffffffffff167f604aa628000000000000000000000000000000000000000000000000000000001790529050600061086b7f000000000000000000000000000000000000000000000000000000000000000083611006565b9050808060200190518101906108819190611ee6565b949350505050565b604080516101e081018252600080825260208201819052818301819052606082018190526080820181905260a0820181905260c0820181905260e08201819052610100820181905261012082018190526101408201819052610160820181905261018082018190526101a082018190526101c082015290517fb2e48a3f0000000000000000000000000000000000000000000000000000000081527f00000000000000000000000000000000000000000000000000000000000000006001600160a01b03169063b2e48a3f906109879085907f00000000000000000000000000000000000000000000000000000000000000009033906004016120cb565b6101e0604051808303816000875af11580156109a7573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906107689190612280565b6001600160a01b037f0000000000000000000000000000000000000000000000000000000000000000163003610a2d576040517f0a57d61d00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b600081604051602401610a409190612451565b60408051601f198184030181529190526020810180517bffffffffffffffffffffffffffffffffffffffffffffffffffffffff167f6d510c5e0000000000000000000000000000000000000000000000000000000017905290506105417f000000000000000000000000000000000000000000000000000000000000000082611006565b6040517f78ecabce000000000000000000000000000000000000000000000000000000008152600481018290526000907f00000000000000000000000000000000000000000000000000000000000000006001600160a01b0316906378ecabce90602401602060405180830381865afa158015610b45573d6000803e3d6000fd5b505050506040513d601f19601f82011682018060405250810190610768919061254d565b604080517fa9008b6900000000000000000000000000000000000000000000000000000000815283516001600160a01b0390811660048301526020850151602483015291840151604482015282151560648201526060917f0000000000000000000000000000000000000000000000000000000000000000169063a9008b6990608401600060405180830381865afa158015610c09573d6000803e3d6000fd5b505050506040513d6000823e601f3d908101601f19168201604052610c31919081019061256a565b9392505050565b6001600160a01b037f0000000000000000000000000000000000000000000000000000000000000000163003610c9a576040517f0a57d61d00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b6040516001600160a01b0380841660248301528216604482015260009060640160408051601f198184030181529190526020810180517bffffffffffffffffffffffffffffffffffffffffffffffffffffffff167fb0b807eb000000000000000000000000000000000000000000000000000000001790529050610d3e7f000000000000000000000000000000000000000000000000000000000000000082611006565b50505050565b6001600160a01b037f0000000000000000000000000000000000000000000000000000000000000000163003610da6576040517f0a57d61d00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b600081604051602401610db991906125d8565b60408051601f198184030181529190526020810180517bffffffffffffffffffffffffffffffffffffffffffffffffffffffff167fb23cc0440000000000000000000000000000000000000000000000000000000017905290506105417f000000000000000000000000000000000000000000000000000000000000000082611006565b604080516101a081018252600080825260208201819052918101829052606081018290526080810182905260a0810182905260c0810182905260e0810182905261010081018290526101208101829052610140810182905261016081018290526101808101919091527f00000000000000000000000000000000000000000000000000000000000000006001600160a01b031663b51f9c2b6040518163ffffffff1660e01b81526004016101a060405180830381865afa158015610f05573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906105ca919061262d565b604080517fb806c80500000000000000000000000000000000000000000000000000000000815284516001600160a01b0390811660048301526020808701516024840152928601516044830152841515606483015283518116608483015291830151821660a48201526060917f0000000000000000000000000000000000000000000000000000000000000000169063b806c8059060c401600060405180830381865afa158015610fde573d6000803e3d6000fd5b505050506040513d6000823e601f3d908101601f19168201604052610881919081019061256a565b6060600080846001600160a01b0316846040516110239190612720565b600060405180830381855af49150503d806000811461105e576040519150601f19603f3d011682016040523d82523d6000602084013e611063565b606091505b50915091508161088157805160208201fd5b7f4e487b7100000000000000000000000000000000000000000000000000000000600052604160045260246000fd5b6040805190810167ffffffffffffffff811182821017156110c7576110c7611075565b60405290565b604051610180810167ffffffffffffffff811182821017156110c7576110c7611075565b6040516060810167ffffffffffffffff811182821017156110c7576110c7611075565b6040516101c0810167ffffffffffffffff811182821017156110c7576110c7611075565b6040516101e0810167ffffffffffffffff811182821017156110c7576110c7611075565b6040516101a0810167ffffffffffffffff811182821017156110c7576110c7611075565b604051601f8201601f1916810167ffffffffffffffff811182821017156111a9576111a9611075565b604052919050565b600067ffffffffffffffff8211156111cb576111cb611075565b5060051b60200190565b6001600160a01b03811681146111ea57600080fd5b50565b80356111f8816111d5565b919050565b6000602080838503121561121057600080fd5b823567ffffffffffffffff81111561122757600080fd5b8301601f8101851361123857600080fd5b803561124b611246826111b1565b611180565b81815260069190911b8201830190838101908783111561126a57600080fd5b928401925b828410156112b657604084890312156112885760008081fd5b6112906110a4565b843561129b816111d5565b8152848601358682015282526040909301929084019061126f565b979650505050505050565b81516001600160a01b031681526101c0810160208301516112ed60208401826001600160a01b03169052565b50604083015161130860408401826001600160a01b03169052565b50606083015161132360608401826001600160a01b03169052565b50608083015161133e60808401826001600160a01b03169052565b5060a083015161135960a08401826001600160a01b03169052565b5060c083015161137460c08401826001600160a01b03169052565b5060e083015161138f60e08401826001600160a01b03169052565b50610100838101516001600160a01b0390811691840191909152610120808501518216908401526101408085015182169084015261016080850151821690840152610180808501518216908401526101a08085015191821681850152905b505092915050565b60006020828403121561140757600080fd5b5035919050565b60005b83811015611429578181015183820152602001611411565b83811115610d3e5750506000910152565b6000815180845261145281602086016020860161140e565b601f01601f19169290920160200192915050565b602081526000610c31602083018461143a565b600067ffffffffffffffff82111561149357611493611075565b50601f01601f191660200190565b600082601f8301126114b257600080fd5b81356114c061124682611479565b8181528460208386010111156114d557600080fd5b816020850160208301376000918101602001919091529392505050565b803563ffffffff811681146111f857600080fd5b67ffffffffffffffff811681146111ea57600080fd5b80356111f881611506565b80151581146111ea57600080fd5b80356111f881611527565b6000602080838503121561155357600080fd5b823567ffffffffffffffff8082111561156b57600080fd5b818501915085601f83011261157f57600080fd5b813561158d611246826111b1565b81815260059190911b830184019084810190888311156115ac57600080fd5b8585015b838110156116c8578035858111156115c85760008081fd5b8601610180818c03601f19018113156115e15760008081fd5b6115e96110cd565b89830135888111156115fb5760008081fd5b6116098e8c838701016114a1565b82525060406116198185016111ed565b8b830152606061162a8186016111ed565b828401526080915061163d8286016114f2565b818401525060a0808501358284015260c0915081850135818401525060e08085013582840152610100915061167382860161151c565b9083015261012061168585820161151c565b8284015261014091508185013581840152506101606116a58186016111ed565b828401526116b4848601611535565b9083015250855250509186019186016115b0565b5098975050505050505050565b602080825282518282018190526000919060409081850190868401855b8281101561172457815180516001600160a01b03908116865290870151168685015292840192908501906001016116f2565b5091979650505050505050565b60006020828403121561174357600080fd5b813567ffffffffffffffff81111561175a57600080fd5b82016102408185031215610c3157600080fd5b81516001600160a01b031681526101e08101602083015161179960208401826001600160a01b03169052565b5060408301516117b460408401826001600160a01b03169052565b5060608301516117cf60608401826001600160a01b03169052565b5060808301516117ea60808401826001600160a01b03169052565b5060a083015161180560a08401826001600160a01b03169052565b5060c083015161182060c08401826001600160a01b03169052565b5060e083015161183b60e08401826001600160a01b03169052565b50610100838101516001600160a01b0390811691840191909152610120808501518216908401526101408085015182169084015261016080850151821690840152610180808501518216908401526101a0808501518216908401526101c08085015191821681850152906113ed565b6000602082840312156118bc57600080fd5b813567ffffffffffffffff8111156118d357600080fd5b82016101608185031215610c3157600080fd5b6000606082840312156118f857600080fd5b6119006110f1565b9050813561190d816111d5565b80825250602082013560208201526040820135604082015292915050565b6000806080838503121561193e57600080fd5b61194884846118e6565b9150606083013561195881611527565b809150509250929050565b6000806040838503121561197657600080fd5b8235611981816111d5565b91506020830135611958816111d5565b600060208083850312156119a457600080fd5b823567ffffffffffffffff8111156119bb57600080fd5b8301601f810185136119cc57600080fd5b80356119da611246826111b1565b818152606091820283018401918482019190888411156119f957600080fd5b938501935b83851015611a4c5780858a031215611a165760008081fd5b611a1e6110f1565b8535611a29816111d5565b8152858701358782015260408087013590820152835293840193918501916119fe565b50979650505050505050565b81516001600160a01b031681526101a081016020830151611a8460208401826001600160a01b03169052565b506040830151611a9f60408401826001600160a01b03169052565b506060830151611aba60608401826001600160a01b03169052565b506080830151611ad560808401826001600160a01b03169052565b5060a0830151611af060a08401826001600160a01b03169052565b5060c0830151611b0b60c08401826001600160a01b03169052565b5060e0830151611b2660e08401826001600160a01b03169052565b50610100838101516001600160a01b03908116918401919091526101208085015182169084015261014080850151821690840152610160808501518216908401526101808085015191821681850152906113ed565b600080600083850360c0811215611b9157600080fd5b611b9b86866118e6565b93506060850135611bab81611527565b925060407fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff8082011215611bdd57600080fd5b50611be66110a4565b6080850135611bf4816111d5565b815260a0850135611c04816111d5565b602082015292959194509192509050565b602080825282518282018190526000919060409081850190868401855b8281101561172457815180516001600160a01b03168552860151868501529284019290850190600101611c32565b600060208284031215611c7257600080fd5b5051919050565b80516111f8816111d5565b60006101c08284031215611c9757600080fd5b611c9f611114565b611ca883611c79565b8152611cb660208401611c79565b6020820152611cc760408401611c79565b6040820152611cd860608401611c79565b6060820152611ce960808401611c79565b6080820152611cfa60a08401611c79565b60a0820152611d0b60c08401611c79565b60c0820152611d1c60e08401611c79565b60e0820152610100611d2f818501611c79565b90820152610120611d41848201611c79565b90820152610140611d53848201611c79565b90820152610160611d65848201611c79565b90820152610180611d77848201611c79565b908201526101a0611d89848201611c79565b908201529392505050565b600060208284031215611da657600080fd5b8151610c31816111d5565b60006020808301818452808551808352604092508286019150828160051b87010184880160005b83811015611ed8577fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffc089840301855281516101808151818652611e1d8287018261143a565b91505088820151611e388a8701826001600160a01b03169052565b50878201516001600160a01b03908116868a015260608084015163ffffffff16908701526080808401519087015260a0808401519087015260c0808401519087015260e08084015167ffffffffffffffff9081169188019190915261010080850151909116908701526101208084015190870152610140808401519091169086015261016091820151151591909401529386019390860190600101611dd8565b509098975050505050505050565b60006020808385031215611ef957600080fd5b825167ffffffffffffffff811115611f1057600080fd5b8301601f81018513611f2157600080fd5b8051611f2f611246826111b1565b81815260069190911b82018301908381019087831115611f4e57600080fd5b928401925b828410156112b65760408489031215611f6c5760008081fd5b611f746110a4565b8451611f7f816111d5565b815284860151611f8e816111d5565b8187015282526040939093019290840190611f53565b8035611faf816111d5565b6001600160a01b039081168352602082013590611fcb826111d5565b9081166020840152604082013590611fe2826111d5565b9081166040840152606082013590611ff9826111d5565b9081166060840152608082013590612010826111d5565b908116608084015260a082013590612027826111d5565b80821660a085015250505050565b60008083357fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe184360301811261206a57600080fd5b830160208101925035905067ffffffffffffffff81111561208a57600080fd5b80360382131561209957600080fd5b9250929050565b818352818160208501375060006020828401015260006020601f19601f840116840101905092915050565b606081526120dc6060820185611fa4565b60006120ea60c086016114f2565b6101206120fe8185018363ffffffff169052565b61210a60e088016114f2565b91506101406121208186018463ffffffff169052565b61016092506101008801358386015261213b82890189612035565b925061024061018081818901526121576102a0890186856120a0565b9450612165848c018c612035565b945092506101a07fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffa089870301818a01526121a08686866120a0565b95506121ad878d0161151c565b96506101c094506121c9858a018867ffffffffffffffff169052565b6121d4828d016114f2565b96506101e093506121ec848a018863ffffffff169052565b6102009650808c0135878a01525050610220838b013581890152828b013582890152612219868c0161151c565b67ffffffffffffffff81166102608a01529550612237818c0161151c565b95505050505061225461028085018367ffffffffffffffff169052565b6001600160a01b0386166020850152915061226c9050565b6001600160a01b0383166040830152610881565b60006101e0828403121561229357600080fd5b61229b611138565b6122a483611c79565b81526122b260208401611c79565b60208201526122c360408401611c79565b60408201526122d460608401611c79565b60608201526122e560808401611c79565b60808201526122f660a08401611c79565b60a082015261230760c08401611c79565b60c082015261231860e08401611c79565b60e082015261010061232b818501611c79565b9082015261012061233d848201611c79565b9082015261014061234f848201611c79565b90820152610160612361848201611c79565b90820152610180612373848201611c79565b908201526101a0612385848201611c79565b908201526101c0611d89848201611c79565b60008083357fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe18436030181126123cc57600080fd5b830160208101925035905067ffffffffffffffff8111156123ec57600080fd5b8060061b360382131561209957600080fd5b8183526000602080850194508260005b85811015612446578135612421816111d5565b6001600160a01b0316875281830135838801526040968701969091019060010161240e565b509495945050505050565b602081526000823561246281611527565b8015156020840152506020830135604083015260408301356060830152606083013561248d816111d5565b6001600160a01b038082166080850152608085013591506124ad826111d5565b80821660a0850152505060a083013560c083015260c083013560e083015261010060e0840135818401528084013590506124e681611506565b61012067ffffffffffffffff82168185015261250381860161151c565b91505061014061251e8185018367ffffffffffffffff169052565b61252a81860186612397565b6101608681015292509050612544610180850183836123fe565b95945050505050565b60006020828403121561255f57600080fd5b8151610c3181611527565b60006020828403121561257c57600080fd5b815167ffffffffffffffff81111561259357600080fd5b8201601f810184136125a457600080fd5b80516125b261124682611479565b8181528560208385010111156125c757600080fd5b61254482602083016020860161140e565b602080825282518282018190526000919060409081850190868401855b8281101561172457815180516001600160a01b03168552868101518786015285015185850152606090930192908501906001016125f5565b60006101a0828403121561264057600080fd5b61264861115c565b61265183611c79565b815261265f60208401611c79565b602082015261267060408401611c79565b604082015261268160608401611c79565b606082015261269260808401611c79565b60808201526126a360a08401611c79565b60a08201526126b460c08401611c79565b60c08201526126c560e08401611c79565b60e08201526101006126d8818501611c79565b908201526101206126ea848201611c79565b908201526101406126fc848201611c79565b9082015261016061270e848201611c79565b90820152610180611d89848201611c79565b6000825161273281846020870161140e565b919091019291505056fea164736f6c634300080f000a",
}

// OPContractsManagerABI is the input ABI used to generate the binding from.
// Deprecated: Use OPContractsManagerMetaData.ABI instead.
var OPContractsManagerABI = OPContractsManagerMetaData.ABI

// OPContractsManagerBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use OPContractsManagerMetaData.Bin instead.
var OPContractsManagerBin = OPContractsManagerMetaData.Bin

// DeployOPContractsManager deploys a new Ethereum contract, binding an instance of OPContractsManager to it.
func DeployOPContractsManager(auth *bind.TransactOpts, backend bind.ContractBackend, _opcmGameTypeAdder common.Address, _opcmDeployer common.Address, _opcmUpgrader common.Address, _opcmInteropMigrator common.Address, _opcmStandardValidator common.Address, _superchainConfig common.Address, _protocolVersions common.Address, _superchainProxyAdmin common.Address) (common.Address, *types.Transaction, *OPContractsManager, error) {
	parsed, err := OPContractsManagerMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(OPContractsManagerBin), backend, _opcmGameTypeAdder, _opcmDeployer, _opcmUpgrader, _opcmInteropMigrator, _opcmStandardValidator, _superchainConfig, _protocolVersions, _superchainProxyAdmin)
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

// Validate is a free data retrieval call binding the contract method 0xa9008b69.
//
// Solidity: function validate((address,bytes32,uint256) _input, bool _allowFailure) view returns(string)
func (_OPContractsManager *OPContractsManagerCaller) Validate(opts *bind.CallOpts, _input OPContractsManagerStandardValidatorValidationInput, _allowFailure bool) (string, error) {
	var out []interface{}
	err := _OPContractsManager.contract.Call(opts, &out, "validate", _input, _allowFailure)

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Validate is a free data retrieval call binding the contract method 0xa9008b69.
//
// Solidity: function validate((address,bytes32,uint256) _input, bool _allowFailure) view returns(string)
func (_OPContractsManager *OPContractsManagerSession) Validate(_input OPContractsManagerStandardValidatorValidationInput, _allowFailure bool) (string, error) {
	return _OPContractsManager.Contract.Validate(&_OPContractsManager.CallOpts, _input, _allowFailure)
}

// Validate is a free data retrieval call binding the contract method 0xa9008b69.
//
// Solidity: function validate((address,bytes32,uint256) _input, bool _allowFailure) view returns(string)
func (_OPContractsManager *OPContractsManagerCallerSession) Validate(_input OPContractsManagerStandardValidatorValidationInput, _allowFailure bool) (string, error) {
	return _OPContractsManager.Contract.Validate(&_OPContractsManager.CallOpts, _input, _allowFailure)
}

// ValidateWithOverrides is a free data retrieval call binding the contract method 0xb806c805.
//
// Solidity: function validateWithOverrides((address,bytes32,uint256) _input, bool _allowFailure, (address,address) _overrides) view returns(string)
func (_OPContractsManager *OPContractsManagerCaller) ValidateWithOverrides(opts *bind.CallOpts, _input OPContractsManagerStandardValidatorValidationInput, _allowFailure bool, _overrides OPContractsManagerStandardValidatorValidationOverrides) (string, error) {
	var out []interface{}
	err := _OPContractsManager.contract.Call(opts, &out, "validateWithOverrides", _input, _allowFailure, _overrides)

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// ValidateWithOverrides is a free data retrieval call binding the contract method 0xb806c805.
//
// Solidity: function validateWithOverrides((address,bytes32,uint256) _input, bool _allowFailure, (address,address) _overrides) view returns(string)
func (_OPContractsManager *OPContractsManagerSession) ValidateWithOverrides(_input OPContractsManagerStandardValidatorValidationInput, _allowFailure bool, _overrides OPContractsManagerStandardValidatorValidationOverrides) (string, error) {
	return _OPContractsManager.Contract.ValidateWithOverrides(&_OPContractsManager.CallOpts, _input, _allowFailure, _overrides)
}

// ValidateWithOverrides is a free data retrieval call binding the contract method 0xb806c805.
//
// Solidity: function validateWithOverrides((address,bytes32,uint256) _input, bool _allowFailure, (address,address) _overrides) view returns(string)
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

// Migrate is a paid mutator transaction binding the contract method 0x6d510c5e.
//
// Solidity: function migrate((bool,(bytes32,uint256),(address,address,uint256,uint256,uint256,uint64,uint64),(address,bytes32)[]) _input) returns()
func (_OPContractsManager *OPContractsManagerTransactor) Migrate(opts *bind.TransactOpts, _input OPContractsManagerInteropMigratorMigrateInput) (*types.Transaction, error) {
	return _OPContractsManager.contract.Transact(opts, "migrate", _input)
}

// Migrate is a paid mutator transaction binding the contract method 0x6d510c5e.
//
// Solidity: function migrate((bool,(bytes32,uint256),(address,address,uint256,uint256,uint256,uint64,uint64),(address,bytes32)[]) _input) returns()
func (_OPContractsManager *OPContractsManagerSession) Migrate(_input OPContractsManagerInteropMigratorMigrateInput) (*types.Transaction, error) {
	return _OPContractsManager.Contract.Migrate(&_OPContractsManager.TransactOpts, _input)
}

// Migrate is a paid mutator transaction binding the contract method 0x6d510c5e.
//
// Solidity: function migrate((bool,(bytes32,uint256),(address,address,uint256,uint256,uint256,uint64,uint64),(address,bytes32)[]) _input) returns()
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

// Upgrade is a paid mutator transaction binding the contract method 0x0b8bd7cb.
//
// Solidity: function upgrade((address,bytes32)[] _opChainConfigs) returns()
func (_OPContractsManager *OPContractsManagerTransactor) Upgrade(opts *bind.TransactOpts, _opChainConfigs []OPContractsManagerOpChainConfig) (*types.Transaction, error) {
	return _OPContractsManager.contract.Transact(opts, "upgrade", _opChainConfigs)
}

// Upgrade is a paid mutator transaction binding the contract method 0x0b8bd7cb.
//
// Solidity: function upgrade((address,bytes32)[] _opChainConfigs) returns()
func (_OPContractsManager *OPContractsManagerSession) Upgrade(_opChainConfigs []OPContractsManagerOpChainConfig) (*types.Transaction, error) {
	return _OPContractsManager.Contract.Upgrade(&_OPContractsManager.TransactOpts, _opChainConfigs)
}

// Upgrade is a paid mutator transaction binding the contract method 0x0b8bd7cb.
//
// Solidity: function upgrade((address,bytes32)[] _opChainConfigs) returns()
func (_OPContractsManager *OPContractsManagerTransactorSession) Upgrade(_opChainConfigs []OPContractsManagerOpChainConfig) (*types.Transaction, error) {
	return _OPContractsManager.Contract.Upgrade(&_OPContractsManager.TransactOpts, _opChainConfigs)
}

// UpgradeSuperchainConfig is a paid mutator transaction binding the contract method 0xb0b807eb.
//
// Solidity: function upgradeSuperchainConfig(address _superchainConfig, address _superchainProxyAdmin) returns()
func (_OPContractsManager *OPContractsManagerTransactor) UpgradeSuperchainConfig(opts *bind.TransactOpts, _superchainConfig common.Address, _superchainProxyAdmin common.Address) (*types.Transaction, error) {
	return _OPContractsManager.contract.Transact(opts, "upgradeSuperchainConfig", _superchainConfig, _superchainProxyAdmin)
}

// UpgradeSuperchainConfig is a paid mutator transaction binding the contract method 0xb0b807eb.
//
// Solidity: function upgradeSuperchainConfig(address _superchainConfig, address _superchainProxyAdmin) returns()
func (_OPContractsManager *OPContractsManagerSession) UpgradeSuperchainConfig(_superchainConfig common.Address, _superchainProxyAdmin common.Address) (*types.Transaction, error) {
	return _OPContractsManager.Contract.UpgradeSuperchainConfig(&_OPContractsManager.TransactOpts, _superchainConfig, _superchainProxyAdmin)
}

// UpgradeSuperchainConfig is a paid mutator transaction binding the contract method 0xb0b807eb.
//
// Solidity: function upgradeSuperchainConfig(address _superchainConfig, address _superchainProxyAdmin) returns()
func (_OPContractsManager *OPContractsManagerTransactorSession) UpgradeSuperchainConfig(_superchainConfig common.Address, _superchainProxyAdmin common.Address) (*types.Transaction, error) {
	return _OPContractsManager.Contract.UpgradeSuperchainConfig(&_OPContractsManager.TransactOpts, _superchainConfig, _superchainProxyAdmin)
}
