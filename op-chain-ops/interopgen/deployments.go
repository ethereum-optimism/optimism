package interopgen

import (
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/opcm"
	"github.com/ethereum/go-ethereum/common"
)

type L1Deployment struct {
	// No global deployed contracts that aren't part of the superchain, yet.
}

type Implementations struct {
	OpcmStandardValidator            common.Address `json:"OPCMStandardValidator"`
	OpcmUtils                        common.Address `json:"OPCMUtils"`
	OpcmMigrator                     common.Address `json:"OPCMMigrator"`
	OpcmV2                           common.Address `json:"OPCMV2"`
	OpcmContainer                    common.Address `json:"OPCMContainer"`
	DelayedWETHImpl                  common.Address `json:"DelayedWETHImpl"`
	OptimismPortalImpl               common.Address `json:"OptimismPortalImpl"`
	EthLockboxImpl                   common.Address `json:"ETHLockboxImpl"`
	PreimageOracleSingleton          common.Address `json:"PreimageOracleSingleton"`
	MipsSingleton                    common.Address `json:"MipsSingleton"`
	SystemConfigImpl                 common.Address `json:"SystemConfigImpl"`
	L1CrossDomainMessengerImpl       common.Address `json:"L1CrossDomainMessengerImpl"`
	L1ERC721BridgeImpl               common.Address `json:"L1ERC721BridgeImpl"`
	L1StandardBridgeImpl             common.Address `json:"L1StandardBridgeImpl"`
	OptimismMintableERC20FactoryImpl common.Address `json:"OptimismMintableERC20FactoryImpl"`
	DisputeGameFactoryImpl           common.Address `json:"DisputeGameFactoryImpl"`
	AnchorStateRegistryImpl          common.Address `json:"AnchorStateRegistryImpl"`
	SuperchainConfigImpl             common.Address `json:"SuperchainConfigImpl"`
	FaultDisputeGameImpl             common.Address `json:"FaultDisputeGameImpl"`
	PermissionedDisputeGameImpl      common.Address `json:"PermissionedDisputeGameImpl"`
	SuperFaultDisputeGameImpl        common.Address `json:"SuperFaultDisputeGameImpl"`
	SuperPermissionedDisputeGameImpl common.Address `json:"SuperPermissionedDisputeGameImpl"`
	ZkDisputeGameImpl                common.Address `json:"ZkDisputeGameImpl"`
	StorageSetterImpl                common.Address `json:"StorageSetterImpl"`
}

func NewImplementationsFromDeployImplementationsOutput(output opcm.DeployImplementationsOutput) Implementations {
	return Implementations{
		OpcmStandardValidator:            output.Address("opcmStandardValidator"),
		OpcmUtils:                        output.Address("opcmUtils"),
		OpcmMigrator:                     output.Address("opcmMigrator"),
		OpcmV2:                           output.Address("opcmV2"),
		OpcmContainer:                    output.Address("opcmContainer"),
		DelayedWETHImpl:                  output.Address("delayedWETHImpl"),
		OptimismPortalImpl:               output.Address("optimismPortalImpl"),
		EthLockboxImpl:                   output.Address("ethLockboxImpl"),
		PreimageOracleSingleton:          output.Address("preimageOracleSingleton"),
		MipsSingleton:                    output.Address("mipsSingleton"),
		SystemConfigImpl:                 output.Address("systemConfigImpl"),
		L1CrossDomainMessengerImpl:       output.Address("l1CrossDomainMessengerImpl"),
		L1ERC721BridgeImpl:               output.Address("l1ERC721BridgeImpl"),
		L1StandardBridgeImpl:             output.Address("l1StandardBridgeImpl"),
		OptimismMintableERC20FactoryImpl: output.Address("optimismMintableERC20FactoryImpl"),
		DisputeGameFactoryImpl:           output.Address("disputeGameFactoryImpl"),
		AnchorStateRegistryImpl:          output.Address("anchorStateRegistryImpl"),
		SuperchainConfigImpl:             output.Address("superchainConfigImpl"),
		FaultDisputeGameImpl:             output.Address("faultDisputeGameImpl"),
		PermissionedDisputeGameImpl:      output.Address("permissionedDisputeGameImpl"),
		SuperFaultDisputeGameImpl:        output.Address("superFaultDisputeGameImpl"),
		SuperPermissionedDisputeGameImpl: output.Address("superPermissionedDisputeGameImpl"),
		ZkDisputeGameImpl:                output.Address("zkDisputeGameImpl"),
		StorageSetterImpl:                output.Address("storageSetterImpl"),
	}
}

type SuperchainDeployment struct {
	Implementations

	ProxyAdmin common.Address `json:"ProxyAdmin"`

	SuperchainConfig      common.Address `json:"SuperchainConfig"`
	SuperchainConfigProxy common.Address `json:"SuperchainConfigProxy"`
}

type L2OpchainDeployment struct {
	OpChainProxyAdmin                 common.Address `json:"OpChainProxyAdmin"`
	AddressManager                    common.Address `json:"AddressManager"`
	L1ERC721BridgeProxy               common.Address `json:"L1ERC721BridgeProxy"`
	SystemConfigProxy                 common.Address `json:"SystemConfigProxy"`
	OptimismMintableERC20FactoryProxy common.Address `json:"OptimismMintableERC20FactoryProxy"`
	L1StandardBridgeProxy             common.Address `json:"L1StandardBridgeProxy"`
	L1CrossDomainMessengerProxy       common.Address `json:"L1CrossDomainMessengerProxy"`
	// Fault proof contracts below.
	OptimismPortalProxy                common.Address `json:"OptimismPortalProxy"`
	ETHLockboxProxy                    common.Address `json:"ETHLockboxProxy"`
	DisputeGameFactoryProxy            common.Address `json:"DisputeGameFactoryProxy"`
	AnchorStateRegistryProxy           common.Address `json:"AnchorStateRegistryProxy"`
	FaultDisputeGame                   common.Address `json:"FaultDisputeGame"`
	PermissionedDisputeGame            common.Address `json:"PermissionedDisputeGame"`
	DelayedWETHPermissionedGameProxy   common.Address `json:"DelayedWETHPermissionedGameProxy"`
	DelayedWETHPermissionlessGameProxy common.Address `json:"DelayedWETHPermissionlessGameProxy"`
}

func NewL2OPChainDeploymentFromDeployOPChainOutput(output opcm.DeployOPChainOutput) L2OpchainDeployment {
	return L2OpchainDeployment{
		OpChainProxyAdmin:                 output.Address("opChainProxyAdmin"),
		AddressManager:                    output.Address("addressManager"),
		L1ERC721BridgeProxy:               output.Address("l1ERC721BridgeProxy"),
		SystemConfigProxy:                 output.Address("systemConfigProxy"),
		OptimismMintableERC20FactoryProxy: output.Address("optimismMintableERC20FactoryProxy"),
		L1StandardBridgeProxy:             output.Address("l1StandardBridgeProxy"),
		L1CrossDomainMessengerProxy:       output.Address("l1CrossDomainMessengerProxy"),
		// Fault proof contracts below.
		OptimismPortalProxy:                output.Address("optimismPortalProxy"),
		ETHLockboxProxy:                    output.Address("ethLockboxProxy"),
		DisputeGameFactoryProxy:            output.Address("disputeGameFactoryProxy"),
		AnchorStateRegistryProxy:           output.Address("anchorStateRegistryProxy"),
		FaultDisputeGame:                   output.Address("faultDisputeGame"),
		PermissionedDisputeGame:            output.Address("permissionedDisputeGame"),
		DelayedWETHPermissionedGameProxy:   output.Address("delayedWETHPermissionedGameProxy"),
		DelayedWETHPermissionlessGameProxy: output.Address("delayedWETHPermissionlessGameProxy"),
	}
}

type L2Deployment struct {
	L2OpchainDeployment

	// In the future this may contain optional extras,
	// e.g. a Safe that will own the L2 chain contracts
}

type InteropDeployment struct {
	DisputeGameFactory common.Address `json:"DisputeGameFactory"`
}

type WorldDeployment struct {
	L1         *L1Deployment            `json:"L1"`
	Superchain *SuperchainDeployment    `json:"Superchain"`
	L2s        map[string]*L2Deployment `json:"L2s"`
	Interop    *InteropDeployment       `json:"Interop"`
}
