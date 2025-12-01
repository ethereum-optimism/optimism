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
	FaultDisputeGameV2Impl           common.Address
	PermissionedDisputeGameV2Impl    common.Address
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
	ABI: "[{\"type\":\"constructor\",\"inputs\":[{\"name\":\"_opcmGameTypeAdder\",\"type\":\"address\",\"internalType\":\"contractOPContractsManagerGameTypeAdder\"},{\"name\":\"_opcmDeployer\",\"type\":\"address\",\"internalType\":\"contractOPContractsManagerDeployer\"},{\"name\":\"_opcmUpgrader\",\"type\":\"address\",\"internalType\":\"contractOPContractsManagerUpgrader\"},{\"name\":\"_opcmInteropMigrator\",\"type\":\"address\",\"internalType\":\"contractOPContractsManagerInteropMigrator\"},{\"name\":\"_opcmStandardValidator\",\"type\":\"address\",\"internalType\":\"contractOPContractsManagerStandardValidator\"},{\"name\":\"_superchainConfig\",\"type\":\"address\",\"internalType\":\"contractISuperchainConfig\"},{\"name\":\"_protocolVersions\",\"type\":\"address\",\"internalType\":\"contractIProtocolVersions\"}],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"addGameType\",\"inputs\":[{\"name\":\"_gameConfigs\",\"type\":\"tuple[]\",\"internalType\":\"structOPContractsManager.AddGameInput[]\",\"components\":[{\"name\":\"saltMixer\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"systemConfig\",\"type\":\"address\",\"internalType\":\"contractISystemConfig\"},{\"name\":\"delayedWETH\",\"type\":\"address\",\"internalType\":\"contractIDelayedWETH\"},{\"name\":\"disputeGameType\",\"type\":\"uint32\",\"internalType\":\"GameType\"},{\"name\":\"disputeAbsolutePrestate\",\"type\":\"bytes32\",\"internalType\":\"Claim\"},{\"name\":\"disputeMaxGameDepth\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"disputeSplitDepth\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"disputeClockExtension\",\"type\":\"uint64\",\"internalType\":\"Duration\"},{\"name\":\"disputeMaxClockDuration\",\"type\":\"uint64\",\"internalType\":\"Duration\"},{\"name\":\"initialBond\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"vm\",\"type\":\"address\",\"internalType\":\"contractIBigStepper\"},{\"name\":\"permissioned\",\"type\":\"bool\",\"internalType\":\"bool\"}]}],\"outputs\":[{\"name\":\"\",\"type\":\"tuple[]\",\"internalType\":\"structOPContractsManager.AddGameOutput[]\",\"components\":[{\"name\":\"delayedWETH\",\"type\":\"address\",\"internalType\":\"contractIDelayedWETH\"},{\"name\":\"faultDisputeGame\",\"type\":\"address\",\"internalType\":\"contractIFaultDisputeGame\"}]}],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"blueprints\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"tuple\",\"internalType\":\"structOPContractsManager.Blueprints\",\"components\":[{\"name\":\"addressManager\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"proxy\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"proxyAdmin\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"l1ChugSplashProxy\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"resolvedDelegateProxy\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"permissionedDisputeGame1\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"permissionedDisputeGame2\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"permissionlessDisputeGame1\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"permissionlessDisputeGame2\",\"type\":\"address\",\"internalType\":\"address\"}]}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"chainIdToBatchInboxAddress\",\"inputs\":[{\"name\":\"_l2ChainId\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"deploy\",\"inputs\":[{\"name\":\"_input\",\"type\":\"tuple\",\"internalType\":\"structOPContractsManager.DeployInput\",\"components\":[{\"name\":\"roles\",\"type\":\"tuple\",\"internalType\":\"structOPContractsManager.Roles\",\"components\":[{\"name\":\"opChainProxyAdminOwner\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"systemConfigOwner\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"batcher\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"unsafeBlockSigner\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"proposer\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"challenger\",\"type\":\"address\",\"internalType\":\"address\"}]},{\"name\":\"basefeeScalar\",\"type\":\"uint32\",\"internalType\":\"uint32\"},{\"name\":\"blobBasefeeScalar\",\"type\":\"uint32\",\"internalType\":\"uint32\"},{\"name\":\"l2ChainId\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"startingAnchorRoot\",\"type\":\"bytes\",\"internalType\":\"bytes\"},{\"name\":\"saltMixer\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"gasLimit\",\"type\":\"uint64\",\"internalType\":\"uint64\"},{\"name\":\"disputeGameType\",\"type\":\"uint32\",\"internalType\":\"GameType\"},{\"name\":\"disputeAbsolutePrestate\",\"type\":\"bytes32\",\"internalType\":\"Claim\"},{\"name\":\"disputeMaxGameDepth\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"disputeSplitDepth\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"disputeClockExtension\",\"type\":\"uint64\",\"internalType\":\"Duration\"},{\"name\":\"disputeMaxClockDuration\",\"type\":\"uint64\",\"internalType\":\"Duration\"}]}],\"outputs\":[{\"name\":\"\",\"type\":\"tuple\",\"internalType\":\"structOPContractsManager.DeployOutput\",\"components\":[{\"name\":\"opChainProxyAdmin\",\"type\":\"address\",\"internalType\":\"contractIProxyAdmin\"},{\"name\":\"addressManager\",\"type\":\"address\",\"internalType\":\"contractIAddressManager\"},{\"name\":\"l1ERC721BridgeProxy\",\"type\":\"address\",\"internalType\":\"contractIL1ERC721Bridge\"},{\"name\":\"systemConfigProxy\",\"type\":\"address\",\"internalType\":\"contractISystemConfig\"},{\"name\":\"optimismMintableERC20FactoryProxy\",\"type\":\"address\",\"internalType\":\"contractIOptimismMintableERC20Factory\"},{\"name\":\"l1StandardBridgeProxy\",\"type\":\"address\",\"internalType\":\"contractIL1StandardBridge\"},{\"name\":\"l1CrossDomainMessengerProxy\",\"type\":\"address\",\"internalType\":\"contractIL1CrossDomainMessenger\"},{\"name\":\"ethLockboxProxy\",\"type\":\"address\",\"internalType\":\"contractIETHLockbox\"},{\"name\":\"optimismPortalProxy\",\"type\":\"address\",\"internalType\":\"contractIOptimismPortal2\"},{\"name\":\"disputeGameFactoryProxy\",\"type\":\"address\",\"internalType\":\"contractIDisputeGameFactory\"},{\"name\":\"anchorStateRegistryProxy\",\"type\":\"address\",\"internalType\":\"contractIAnchorStateRegistry\"},{\"name\":\"faultDisputeGame\",\"type\":\"address\",\"internalType\":\"contractIFaultDisputeGame\"},{\"name\":\"permissionedDisputeGame\",\"type\":\"address\",\"internalType\":\"contractIPermissionedDisputeGame\"},{\"name\":\"delayedWETHPermissionedGameProxy\",\"type\":\"address\",\"internalType\":\"contractIDelayedWETH\"},{\"name\":\"delayedWETHPermissionlessGameProxy\",\"type\":\"address\",\"internalType\":\"contractIDelayedWETH\"}]}],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"devFeatureBitmap\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"implementations\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"tuple\",\"internalType\":\"structOPContractsManager.Implementations\",\"components\":[{\"name\":\"superchainConfigImpl\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"protocolVersionsImpl\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"l1ERC721BridgeImpl\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"optimismPortalImpl\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"optimismPortalInteropImpl\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"ethLockboxImpl\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"systemConfigImpl\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"optimismMintableERC20FactoryImpl\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"l1CrossDomainMessengerImpl\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"l1StandardBridgeImpl\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"disputeGameFactoryImpl\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"anchorStateRegistryImpl\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"delayedWETHImpl\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"mipsImpl\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"faultDisputeGameV2Impl\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"permissionedDisputeGameV2Impl\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"superFaultDisputeGameImpl\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"superPermissionedDisputeGameImpl\",\"type\":\"address\",\"internalType\":\"address\"}]}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"isDevFeatureEnabled\",\"inputs\":[{\"name\":\"_feature\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"outputs\":[{\"name\":\"\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"migrate\",\"inputs\":[{\"name\":\"_input\",\"type\":\"tuple\",\"internalType\":\"structOPContractsManagerInteropMigrator.MigrateInput\",\"components\":[{\"name\":\"usePermissionlessGame\",\"type\":\"bool\",\"internalType\":\"bool\"},{\"name\":\"startingAnchorRoot\",\"type\":\"tuple\",\"internalType\":\"structProposal\",\"components\":[{\"name\":\"root\",\"type\":\"bytes32\",\"internalType\":\"Hash\"},{\"name\":\"l2SequenceNumber\",\"type\":\"uint256\",\"internalType\":\"uint256\"}]},{\"name\":\"gameParameters\",\"type\":\"tuple\",\"internalType\":\"structOPContractsManagerInteropMigrator.GameParameters\",\"components\":[{\"name\":\"proposer\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"challenger\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"maxGameDepth\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"splitDepth\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"initBond\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"clockExtension\",\"type\":\"uint64\",\"internalType\":\"Duration\"},{\"name\":\"maxClockDuration\",\"type\":\"uint64\",\"internalType\":\"Duration\"}]},{\"name\":\"opChainConfigs\",\"type\":\"tuple[]\",\"internalType\":\"structOPContractsManager.OpChainConfig[]\",\"components\":[{\"name\":\"systemConfigProxy\",\"type\":\"address\",\"internalType\":\"contractISystemConfig\"},{\"name\":\"cannonPrestate\",\"type\":\"bytes32\",\"internalType\":\"Claim\"},{\"name\":\"cannonKonaPrestate\",\"type\":\"bytes32\",\"internalType\":\"Claim\"}]}]}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"opcmDeployer\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"contractOPContractsManagerDeployer\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"opcmGameTypeAdder\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"contractOPContractsManagerGameTypeAdder\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"opcmInteropMigrator\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"contractOPContractsManagerInteropMigrator\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"opcmStandardValidator\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"contractOPContractsManagerStandardValidator\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"opcmUpgrader\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"contractOPContractsManagerUpgrader\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"protocolVersions\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"contractIProtocolVersions\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"superchainConfig\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"contractISuperchainConfig\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"updatePrestate\",\"inputs\":[{\"name\":\"_prestateUpdateInputs\",\"type\":\"tuple[]\",\"internalType\":\"structOPContractsManager.UpdatePrestateInput[]\",\"components\":[{\"name\":\"systemConfigProxy\",\"type\":\"address\",\"internalType\":\"contractISystemConfig\"},{\"name\":\"cannonPrestate\",\"type\":\"bytes32\",\"internalType\":\"Claim\"},{\"name\":\"cannonKonaPrestate\",\"type\":\"bytes32\",\"internalType\":\"Claim\"}]}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"upgrade\",\"inputs\":[{\"name\":\"_opChainConfigs\",\"type\":\"tuple[]\",\"internalType\":\"structOPContractsManager.OpChainConfig[]\",\"components\":[{\"name\":\"systemConfigProxy\",\"type\":\"address\",\"internalType\":\"contractISystemConfig\"},{\"name\":\"cannonPrestate\",\"type\":\"bytes32\",\"internalType\":\"Claim\"},{\"name\":\"cannonKonaPrestate\",\"type\":\"bytes32\",\"internalType\":\"Claim\"}]}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"upgradeSuperchainConfig\",\"inputs\":[{\"name\":\"_superchainConfig\",\"type\":\"address\",\"internalType\":\"contractISuperchainConfig\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"validate\",\"inputs\":[{\"name\":\"_input\",\"type\":\"tuple\",\"internalType\":\"structOPContractsManagerStandardValidator.ValidationInput\",\"components\":[{\"name\":\"sysCfg\",\"type\":\"address\",\"internalType\":\"contractISystemConfig\"},{\"name\":\"absolutePrestate\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"l2ChainID\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"proposer\",\"type\":\"address\",\"internalType\":\"address\"}]},{\"name\":\"_allowFailure\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"outputs\":[{\"name\":\"\",\"type\":\"string\",\"internalType\":\"string\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"validate\",\"inputs\":[{\"name\":\"_input\",\"type\":\"tuple\",\"internalType\":\"structOPContractsManagerStandardValidator.ValidationInputDev\",\"components\":[{\"name\":\"sysCfg\",\"type\":\"address\",\"internalType\":\"contractISystemConfig\"},{\"name\":\"cannonPrestate\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"cannonKonaPrestate\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"l2ChainID\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"proposer\",\"type\":\"address\",\"internalType\":\"address\"}]},{\"name\":\"_allowFailure\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"outputs\":[{\"name\":\"\",\"type\":\"string\",\"internalType\":\"string\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"validateWithOverrides\",\"inputs\":[{\"name\":\"_input\",\"type\":\"tuple\",\"internalType\":\"structOPContractsManagerStandardValidator.ValidationInput\",\"components\":[{\"name\":\"sysCfg\",\"type\":\"address\",\"internalType\":\"contractISystemConfig\"},{\"name\":\"absolutePrestate\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"l2ChainID\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"proposer\",\"type\":\"address\",\"internalType\":\"address\"}]},{\"name\":\"_allowFailure\",\"type\":\"bool\",\"internalType\":\"bool\"},{\"name\":\"_overrides\",\"type\":\"tuple\",\"internalType\":\"structOPContractsManagerStandardValidator.ValidationOverrides\",\"components\":[{\"name\":\"l1PAOMultisig\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"challenger\",\"type\":\"address\",\"internalType\":\"address\"}]}],\"outputs\":[{\"name\":\"\",\"type\":\"string\",\"internalType\":\"string\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"validateWithOverrides\",\"inputs\":[{\"name\":\"_input\",\"type\":\"tuple\",\"internalType\":\"structOPContractsManagerStandardValidator.ValidationInputDev\",\"components\":[{\"name\":\"sysCfg\",\"type\":\"address\",\"internalType\":\"contractISystemConfig\"},{\"name\":\"cannonPrestate\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"cannonKonaPrestate\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"l2ChainID\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"proposer\",\"type\":\"address\",\"internalType\":\"address\"}]},{\"name\":\"_allowFailure\",\"type\":\"bool\",\"internalType\":\"bool\"},{\"name\":\"_overrides\",\"type\":\"tuple\",\"internalType\":\"structOPContractsManagerStandardValidator.ValidationOverrides\",\"components\":[{\"name\":\"l1PAOMultisig\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"challenger\",\"type\":\"address\",\"internalType\":\"address\"}]}],\"outputs\":[{\"name\":\"\",\"type\":\"string\",\"internalType\":\"string\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"version\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"string\",\"internalType\":\"string\"}],\"stateMutability\":\"pure\"},{\"type\":\"error\",\"name\":\"AddressHasNoCode\",\"inputs\":[{\"name\":\"who\",\"type\":\"address\",\"internalType\":\"address\"}]},{\"type\":\"error\",\"name\":\"AddressNotFound\",\"inputs\":[{\"name\":\"who\",\"type\":\"address\",\"internalType\":\"address\"}]},{\"type\":\"error\",\"name\":\"AlreadyReleased\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"InvalidChainId\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"InvalidDevFeatureAccess\",\"inputs\":[{\"name\":\"devFeature\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}]},{\"type\":\"error\",\"name\":\"InvalidGameConfigs\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"InvalidRoleAddress\",\"inputs\":[{\"name\":\"role\",\"type\":\"string\",\"internalType\":\"string\"}]},{\"type\":\"error\",\"name\":\"InvalidStartingAnchorRoot\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"LatestReleaseNotSet\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"OnlyDelegatecall\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"PrestateNotSet\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"PrestateRequired\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"SuperchainConfigMismatch\",\"inputs\":[{\"name\":\"systemConfig\",\"type\":\"address\",\"internalType\":\"contractISystemConfig\"}]},{\"type\":\"error\",\"name\":\"SuperchainProxyAdminMismatch\",\"inputs\":[]}]",
	Bin: "0x6101806040523480156200001257600080fd5b5060405162002e0238038062002e02833981016040819052620000359162000306565b60405163b6a4cd2160e01b81526001600160a01b03838116600483015287169063b6a4cd219060240160006040518083038186803b1580156200007757600080fd5b505afa1580156200008c573d6000803e3d6000fd5b505060405163b6a4cd2160e01b81526001600160a01b0384811660048301528916925063b6a4cd21915060240160006040518083038186803b158015620000d257600080fd5b505afa158015620000e7573d6000803e3d6000fd5b505060405163b6a4cd2160e01b81526001600160a01b038a811660048301528916925063b6a4cd21915060240160006040518083038186803b1580156200012d57600080fd5b505afa15801562000142573d6000803e3d6000fd5b505060405163b6a4cd2160e01b81526001600160a01b03891660048201819052925063b6a4cd21915060240160006040518083038186803b1580156200018757600080fd5b505afa1580156200019c573d6000803e3d6000fd5b505060405163b6a4cd2160e01b81526001600160a01b0388811660048301528916925063b6a4cd21915060240160006040518083038186803b158015620001e257600080fd5b505afa158015620001f7573d6000803e3d6000fd5b505060405163b6a4cd2160e01b81526001600160a01b0387811660048301528916925063b6a4cd21915060240160006040518083038186803b1580156200023d57600080fd5b505afa15801562000252573d6000803e3d6000fd5b505060405163b6a4cd2160e01b81526001600160a01b0386811660048301528916925063b6a4cd21915060240160006040518083038186803b1580156200029857600080fd5b505afa158015620002ad573d6000803e3d6000fd5b5050506001600160a01b039788166080525094861660a05292851660c05290841660e05283166101005282166101205216610140523061016052620003b1565b6001600160a01b03811681146200030357600080fd5b50565b600080600080600080600060e0888a0312156200032257600080fd5b87516200032f81620002ed565b60208901519097506200034281620002ed565b60408901519096506200035581620002ed565b60608901519095506200036881620002ed565b60808901519094506200037b81620002ed565b60a08901519093506200038e81620002ed565b60c0890151909250620003a181620002ed565b8091505092959891949750929550565b60805160a05160c05160e0516101005161012051610140516101605161295f620004a3600039600081816108170152818161091701528181610c7b01528181610e400152610f3e0152600061034f0152600081816102600152610af20152600081816103d4015281816104890152818161078a01528181610c3801526110600152600081816101fb01526108e101526000818161019701528181610f0f0152611008015260008181610328015281816105130152818161062b015281816106de01528181610abb01528181610b920152610db50152600081816103fb015281816109e30152610d45015261295f6000f3fe608060405234801561001057600080fd5b506004361061018d5760003560e01c8063613e827b116100e3578063b51f9c2b1161008c578063c993f27c11610066578063c993f27c1461041d578063cbeda5a714610430578063f3edcbe11461044357600080fd5b8063b51f9c2b146103ba578063ba7903db146103cf578063becbdf4a146103f657600080fd5b806378ecabce116100bd57806378ecabce146103715780638970ac4414610394578063b23cc044146103a757600080fd5b8063613e827b14610303578063622d56f1146103235780636624856a1461034a57600080fd5b8063318b1b801161014557806354fd4d501161011f57806354fd4d501461029557806358084273146102ce578063604aa628146102e357600080fd5b8063318b1b801461024857806335e80ab31461025b57806341fe53851461028257600080fd5b80631481a724116101765780631481a724146101f65780631d8a4e921461021d57806330e9012c1461023357600080fd5b806303dbe68c146101925780630e9d5cb9146101d6575b600080fd5b6101b97f000000000000000000000000000000000000000000000000000000000000000081565b6040516001600160a01b0390911681526020015b60405180910390f35b6101e96101e436600461131b565b610456565b6040516101cd91906113be565b6101b97f000000000000000000000000000000000000000000000000000000000000000081565b61022561050f565b6040519081526020016101cd565b61023b610598565b6040516101cd91906113d1565b6101b9610256366004611539565b6106ac565b6101b97f000000000000000000000000000000000000000000000000000000000000000081565b6101e9610290366004611552565b610757565b60408051808201909152600581527f352e342e3000000000000000000000000000000000000000000000000000000060208201526101e9565b6102e16102dc36600461158a565b61080d565b005b6102f66102f136600461169d565b61090b565b6040516101cd9190611832565b61031661031136600461188e565b610a1e565b6040516101cd91906118ca565b6101b97f000000000000000000000000000000000000000000000000000000000000000081565b6101b97f000000000000000000000000000000000000000000000000000000000000000081565b61038461037f366004611539565b610b60565b60405190151581526020016101cd565b6101e96103a2366004611a8a565b610c05565b6102e16103b5366004611b6e565b610c71565b6103c2610d6a565b6040516101cd9190611bb7565b6101b97f000000000000000000000000000000000000000000000000000000000000000081565b6101b97f000000000000000000000000000000000000000000000000000000000000000081565b6102e161042b366004611c33565b610e36565b6102e161043e366004611b6e565b610f34565b6101e9610451366004611c50565b61102d565b6040517f0e9d5cb90000000000000000000000000000000000000000000000000000000081526060906001600160a01b037f00000000000000000000000000000000000000000000000000000000000000001690630e9d5cb9906104c290879087908790600401611c7d565b600060405180830381865afa1580156104df573d6000803e3d6000fd5b505050506040513d6000823e601f3d908101601f191682016040526105079190810190611cdf565b949350505050565b60007f00000000000000000000000000000000000000000000000000000000000000006001600160a01b0316631d8a4e926040518163ffffffff1660e01b8152600401602060405180830381865afa15801561056f573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906105939190611d56565b905090565b6040805161024081018252600080825260208201819052918101829052606081018290526080810182905260a0810182905260c0810182905260e08101829052610100810182905261012081018290526101408101829052610160810182905261018081018290526101a081018290526101c081018290526101e0810182905261020081018290526102208101919091527f00000000000000000000000000000000000000000000000000000000000000006001600160a01b03166330e9012c6040518163ffffffff1660e01b815260040161024060405180830381865afa158015610688573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906105939190611d7a565b6040517f318b1b80000000000000000000000000000000000000000000000000000000008152600481018290526000907f00000000000000000000000000000000000000000000000000000000000000006001600160a01b03169063318b1b8090602401602060405180830381865afa15801561072d573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906107519190611ed2565b92915050565b6040517f41fe53850000000000000000000000000000000000000000000000000000000081526060906001600160a01b037f000000000000000000000000000000000000000000000000000000000000000016906341fe5385906107c19086908690600401611eef565b600060405180830381865afa1580156107de573d6000803e3d6000fd5b505050506040513d6000823e601f3d908101601f191682016040526108069190810190611cdf565b9392505050565b6001600160a01b037f000000000000000000000000000000000000000000000000000000000000000016300361086f576040517f0a57d61d00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b6000816040516024016108829190612001565b60408051601f198184030181529190526020810180517bffffffffffffffffffffffffffffffffffffffffffffffffffffffff167f580842730000000000000000000000000000000000000000000000000000000017905290506109067f000000000000000000000000000000000000000000000000000000000000000082611097565b505050565b60606001600160a01b037f000000000000000000000000000000000000000000000000000000000000000016300361096f576040517f0a57d61d00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b60008260405160240161098291906120f4565b60408051601f198184030181529190526020810180517bffffffffffffffffffffffffffffffffffffffffffffffffffffffff167f604aa6280000000000000000000000000000000000000000000000000000000017905290506000610a087f000000000000000000000000000000000000000000000000000000000000000083611097565b9050808060200190518101906105079190612229565b604080516101e081018252600080825260208201819052818301819052606082018190526080820181905260a0820181905260c0820181905260e08201819052610100820181905261012082018190526101408201819052610160820181905261018082018190526101a082018190526101c082015290517fb2e48a3f0000000000000000000000000000000000000000000000000000000081527f00000000000000000000000000000000000000000000000000000000000000006001600160a01b03169063b2e48a3f90610b1c9085907f0000000000000000000000000000000000000000000000000000000000000000903390600401612412565b6101e0604051808303816000875af1158015610b3c573d6000803e3d6000fd5b505050506040513d601f19601f8201168201806040525081019061075191906125c7565b6040517f78ecabce000000000000000000000000000000000000000000000000000000008152600481018290526000907f00000000000000000000000000000000000000000000000000000000000000006001600160a01b0316906378ecabce90602401602060405180830381865afa158015610be1573d6000803e3d6000fd5b505050506040513d601f19601f8201168201806040525081019061075191906126de565b6040517f8970ac440000000000000000000000000000000000000000000000000000000081526060906001600160a01b037f00000000000000000000000000000000000000000000000000000000000000001690638970ac44906104c2908790879087906004016126fb565b6001600160a01b037f0000000000000000000000000000000000000000000000000000000000000000163003610cd3576040517f0a57d61d00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b600081604051602401610ce6919061276c565b60408051601f198184030181529190526020810180517bffffffffffffffffffffffffffffffffffffffffffffffffffffffff167fb23cc0440000000000000000000000000000000000000000000000000000000017905290506109067f000000000000000000000000000000000000000000000000000000000000000082611097565b6040805161012081018252600080825260208201819052918101829052606081018290526080810182905260a0810182905260c0810182905260e081018290526101008101919091527f00000000000000000000000000000000000000000000000000000000000000006001600160a01b031663b51f9c2b6040518163ffffffff1660e01b815260040161012060405180830381865afa158015610e12573d6000803e3d6000fd5b505050506040513d601f19601f8201168201806040525081019061059391906127d7565b6001600160a01b037f0000000000000000000000000000000000000000000000000000000000000000163003610e98576040517f0a57d61d00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b6040516001600160a01b038216602482015260009060440160408051601f198184030181529190526020810180517bffffffffffffffffffffffffffffffffffffffffffffffffffffffff167fc993f27c0000000000000000000000000000000000000000000000000000000017905290506109067f000000000000000000000000000000000000000000000000000000000000000082611097565b6001600160a01b037f0000000000000000000000000000000000000000000000000000000000000000163003610f96576040517f0a57d61d00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b600081604051602401610fa99190612882565b60408051601f198184030181529190526020810180517bffffffffffffffffffffffffffffffffffffffffffffffffffffffff167fcbeda5a70000000000000000000000000000000000000000000000000000000017905290506109067f000000000000000000000000000000000000000000000000000000000000000082611097565b6040517ff3edcbe10000000000000000000000000000000000000000000000000000000081526060906001600160a01b037f0000000000000000000000000000000000000000000000000000000000000000169063f3edcbe1906107c190869086906004016128e1565b6060600080846001600160a01b0316846040516110b49190612936565b600060405180830381855af49150503d80600081146110ef576040519150601f19603f3d011682016040523d82523d6000602084013e6110f4565b606091505b50915091508161050757805160208201fd5b7f4e487b7100000000000000000000000000000000000000000000000000000000600052604160045260246000fd5b6040805190810167ffffffffffffffff8111828210171561115857611158611106565b60405290565b604051610180810167ffffffffffffffff8111828210171561115857611158611106565b604051610240810167ffffffffffffffff8111828210171561115857611158611106565b6040516101e0810167ffffffffffffffff8111828210171561115857611158611106565b604051610120810167ffffffffffffffff8111828210171561115857611158611106565b604051601f8201601f1916810167ffffffffffffffff8111828210171561121757611217611106565b604052919050565b6001600160a01b038116811461123457600080fd5b50565b80356112428161121f565b919050565b60006080828403121561125957600080fd5b6040516080810181811067ffffffffffffffff8211171561127c5761127c611106565b604052905080823561128d8161121f565b80825250602083013560208201526040830135604082015260608301356112b38161121f565b6060919091015292915050565b801515811461123457600080fd5b8035611242816112c0565b6000604082840312156112eb57600080fd5b6112f3611135565b905081356113008161121f565b815260208201356113108161121f565b602082015292915050565b600080600060e0848603121561133057600080fd5b61133a8585611247565b9250608084013561134a816112c0565b91506113598560a086016112d9565b90509250925092565b60005b8381101561137d578181015183820152602001611365565b8381111561138c576000848401525b50505050565b600081518084526113aa816020860160208601611362565b601f01601f19169290920160200192915050565b6020815260006108066020830184611392565b81516001600160a01b03168152610240810160208301516113fd60208401826001600160a01b03169052565b50604083015161141860408401826001600160a01b03169052565b50606083015161143360608401826001600160a01b03169052565b50608083015161144e60808401826001600160a01b03169052565b5060a083015161146960a08401826001600160a01b03169052565b5060c083015161148460c08401826001600160a01b03169052565b5060e083015161149f60e08401826001600160a01b03169052565b50610100838101516001600160a01b0390811691840191909152610120808501518216908401526101408085015182169084015261016080850151821690840152610180808501518216908401526101a0808501518216908401526101c0808501518216908401526101e080850151821690840152610200808501518216908401526102208085015191821681850152905b505092915050565b60006020828403121561154b57600080fd5b5035919050565b60008060a0838503121561156557600080fd5b61156f8484611247565b9150608083013561157f816112c0565b809150509250929050565b60006020828403121561159c57600080fd5b813567ffffffffffffffff8111156115b357600080fd5b8201610160818503121561080657600080fd5b600067ffffffffffffffff8211156115e0576115e0611106565b5060051b60200190565b600067ffffffffffffffff82111561160457611604611106565b50601f01601f191660200190565b600082601f83011261162357600080fd5b8135611636611631826115ea565b6111ee565b81815284602083860101111561164b57600080fd5b816020850160208301376000918101602001919091529392505050565b803563ffffffff8116811461124257600080fd5b67ffffffffffffffff8116811461123457600080fd5b80356112428161167c565b600060208083850312156116b057600080fd5b823567ffffffffffffffff808211156116c857600080fd5b818501915085601f8301126116dc57600080fd5b81356116ea611631826115c6565b81815260059190911b8301840190848101908883111561170957600080fd5b8585015b83811015611825578035858111156117255760008081fd5b8601610180818c03601f190181131561173e5760008081fd5b61174661115e565b89830135888111156117585760008081fd5b6117668e8c83870101611612565b8252506040611776818501611237565b8b8301526060611787818601611237565b828401526080915061179a828601611668565b818401525060a0808501358284015260c0915081850135818401525060e0808501358284015261010091506117d0828601611692565b908301526101206117e2858201611692565b828401526101409150818501358184015250610160611802818601611237565b828401526118118486016112ce565b90830152508552505091860191860161170d565b5098975050505050505050565b602080825282518282018190526000919060409081850190868401855b8281101561188157815180516001600160a01b039081168652908701511686850152928401929085019060010161184f565b5091979650505050505050565b6000602082840312156118a057600080fd5b813567ffffffffffffffff8111156118b757600080fd5b8201610240818503121561080657600080fd5b81516001600160a01b031681526101e0810160208301516118f660208401826001600160a01b03169052565b50604083015161191160408401826001600160a01b03169052565b50606083015161192c60608401826001600160a01b03169052565b50608083015161194760808401826001600160a01b03169052565b5060a083015161196260a08401826001600160a01b03169052565b5060c083015161197d60c08401826001600160a01b03169052565b5060e083015161199860e08401826001600160a01b03169052565b50610100838101516001600160a01b0390811691840191909152610120808501518216908401526101408085015182169084015261016080850151821690840152610180808501518216908401526101a0808501518216908401526101c0808501519182168185015290611531565b600060a08284031215611a1957600080fd5b60405160a0810181811067ffffffffffffffff82111715611a3c57611a3c611106565b6040529050808235611a4d8161121f565b808252506020830135602082015260408301356040820152606083013560608201526080830135611a7d8161121f565b6080919091015292915050565b60008060006101008486031215611aa057600080fd5b611aaa8585611a07565b925060a0840135611aba816112c0565b91506113598560c086016112d9565b6000611ad7611631846115c6565b83815290506020808201906060808602850187811115611af657600080fd5b855b81811015611b625782818a031215611b105760008081fd5b6040805184810181811067ffffffffffffffff82111715611b3357611b33611106565b82528235611b408161121f565b8152828601358682015281830135918101919091528552938301938201611af8565b50505050509392505050565b600060208284031215611b8057600080fd5b813567ffffffffffffffff811115611b9757600080fd5b8201601f81018413611ba857600080fd5b61050784823560208401611ac9565b81516001600160a01b03908116825260208084015182169083015260408084015182169083015260608084015182169083015260808084015182169083015260a08084015182169083015260c08084015182169083015260e0808401518216908301526101008084015191821681840152610120830191611531565b600060208284031215611c4557600080fd5b81356108068161121f565b60008060c08385031215611c6357600080fd5b611c6d8484611a07565b915060a083013561157f816112c0565b60e08101611cb5828680516001600160a01b039081168352602080830151908401526040808301519084015260609182015116910152565b831515608083015282516001600160a01b0390811660a084015260208401511660c0830152610507565b600060208284031215611cf157600080fd5b815167ffffffffffffffff811115611d0857600080fd5b8201601f81018413611d1957600080fd5b8051611d27611631826115ea565b818152856020838501011115611d3c57600080fd5b611d4d826020830160208601611362565b95945050505050565b600060208284031215611d6857600080fd5b5051919050565b80516112428161121f565b60006102408284031215611d8d57600080fd5b611d95611182565b611d9e83611d6f565b8152611dac60208401611d6f565b6020820152611dbd60408401611d6f565b6040820152611dce60608401611d6f565b6060820152611ddf60808401611d6f565b6080820152611df060a08401611d6f565b60a0820152611e0160c08401611d6f565b60c0820152611e1260e08401611d6f565b60e0820152610100611e25818501611d6f565b90820152610120611e37848201611d6f565b90820152610140611e49848201611d6f565b90820152610160611e5b848201611d6f565b90820152610180611e6d848201611d6f565b908201526101a0611e7f848201611d6f565b908201526101c0611e91848201611d6f565b908201526101e0611ea3848201611d6f565b90820152610200611eb5848201611d6f565b90820152610220611ec7848201611d6f565b908201529392505050565b600060208284031215611ee457600080fd5b81516108068161121f565b60a08101611f27828580516001600160a01b039081168352602080830151908401526040808301519084015260609182015116910152565b82151560808301529392505050565b60008083357fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe1843603018112611f6b57600080fd5b830160208101925035905067ffffffffffffffff811115611f8b57600080fd5b606081023603821315611f9d57600080fd5b9250929050565b8183526000602080850194508260005b85811015611ff6578135611fc78161121f565b6001600160a01b0316875281830135838801526040808301359088015260609687019690910190600101611fb4565b509495945050505050565b6020815260008235612012816112c0565b8015156020840152506020830135604083015260408301356060830152606083013561203d8161121f565b6001600160a01b0380821660808501526080850135915061205d8261121f565b80821660a0850152505060a083013560c083015260c083013560e083015261010060e0840135818401528084013590506120968161167c565b61012067ffffffffffffffff8216818501526120b3818601611692565b9150506101406120ce8185018367ffffffffffffffff169052565b6120da81860186611f36565b6101608681015292509050611d4d61018085018383611fa4565b60006020808301818452808551808352604092508286019150828160051b87010184880160005b8381101561221b577fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffc08984030185528151610180815181865261216082870182611392565b9150508882015161217b8a8701826001600160a01b03169052565b50878201516001600160a01b03908116868a015260608084015163ffffffff16908701526080808401519087015260a0808401519087015260c0808401519087015260e08084015167ffffffffffffffff908116918801919091526101008085015190911690870152610120808401519087015261014080840151909116908601526101609182015115159190940152938601939086019060010161211b565b509098975050505050505050565b6000602080838503121561223c57600080fd5b825167ffffffffffffffff81111561225357600080fd5b8301601f8101851361226457600080fd5b8051612272611631826115c6565b81815260069190911b8201830190838101908783111561229157600080fd5b928401925b828410156122e757604084890312156122af5760008081fd5b6122b7611135565b84516122c28161121f565b8152848601516122d18161121f565b8187015282526040939093019290840190612296565b979650505050505050565b80356122fd8161121f565b6001600160a01b0390811683526020820135906123198261121f565b90811660208401526040820135906123308261121f565b90811660408401526060820135906123478261121f565b908116606084015260808201359061235e8261121f565b908116608084015260a0820135906123758261121f565b80821660a085015250505050565b60008083357fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe18436030181126123b857600080fd5b830160208101925035905067ffffffffffffffff8111156123d857600080fd5b803603821315611f9d57600080fd5b818352818160208501375060006020828401015260006020601f19601f840116840101905092915050565b6060815261242360608201856122f2565b600061243160c08601611668565b6101206124458185018363ffffffff169052565b61245160e08801611668565b91506101406124678186018463ffffffff169052565b61016092506101008801358386015261248282890189612383565b9250610240610180818189015261249e6102a0890186856123e7565b94506124ac848c018c612383565b945092506101a07fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffa089870301818a01526124e78686866123e7565b95506124f4878d01611692565b96506101c09450612510858a018867ffffffffffffffff169052565b61251b828d01611668565b96506101e09350612533848a018863ffffffff169052565b6102009650808c0135878a01525050610220838b013581890152828b013582890152612560868c01611692565b67ffffffffffffffff81166102608a0152955061257e818c01611692565b95505050505061259b61028085018367ffffffffffffffff169052565b6001600160a01b038616602085015291506125b39050565b6001600160a01b0383166040830152610507565b60006101e082840312156125da57600080fd5b6125e26111a6565b6125eb83611d6f565b81526125f960208401611d6f565b602082015261260a60408401611d6f565b604082015261261b60608401611d6f565b606082015261262c60808401611d6f565b608082015261263d60a08401611d6f565b60a082015261264e60c08401611d6f565b60c082015261265f60e08401611d6f565b60e0820152610100612672818501611d6f565b90820152610120612684848201611d6f565b90820152610140612696848201611d6f565b908201526101606126a8848201611d6f565b908201526101806126ba848201611d6f565b908201526101a06126cc848201611d6f565b908201526101c0611ec7848201611d6f565b6000602082840312156126f057600080fd5b8151610806816112c0565b610100810161274282866001600160a01b03808251168352602082015160208401526040820151604084015260608201516060840152806080830151166080840152505050565b83151560a083015282516001600160a01b0390811660c084015260208401511660e0830152610507565b6020808252825182820181905260009190848201906040850190845b818110156127cb576127b883855180516001600160a01b0316825260208082015190830152604090810151910152565b9284019260609290920191600101612788565b50909695505050505050565b600061012082840312156127ea57600080fd5b6127f26111ca565b6127fb83611d6f565b815261280960208401611d6f565b602082015261281a60408401611d6f565b604082015261282b60608401611d6f565b606082015261283c60808401611d6f565b608082015261284d60a08401611d6f565b60a082015261285e60c08401611d6f565b60c082015261286f60e08401611d6f565b60e0820152610100611ec7818501611d6f565b6020808252825182820181905260009190848201906040850190845b818110156127cb576128ce83855180516001600160a01b0316825260208082015190830152604090810151910152565b928401926060929092019160010161289e565b60c0810161292782856001600160a01b03808251168352602082015160208401526040820151604084015260608201516060840152806080830151166080840152505050565b82151560a08301529392505050565b60008251612948818460208701611362565b919091019291505056fea164736f6c634300080f000a",
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
// Solidity: function blueprints() view returns((address,address,address,address,address,address,address,address,address))
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
// Solidity: function blueprints() view returns((address,address,address,address,address,address,address,address,address))
func (_OPContractsManager *OPContractsManagerSession) Blueprints() (OPContractsManagerBlueprints, error) {
	return _OPContractsManager.Contract.Blueprints(&_OPContractsManager.CallOpts)
}

// Blueprints is a free data retrieval call binding the contract method 0xb51f9c2b.
//
// Solidity: function blueprints() view returns((address,address,address,address,address,address,address,address,address))
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
