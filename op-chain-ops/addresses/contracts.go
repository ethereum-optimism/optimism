package addresses

import "github.com/ethereum/go-ethereum/common"

type L1Contracts struct {
	Superchain      *SuperchainContracts
	Implementations *ImplementationsContracts
	OpChain         *OpChainContracts
}

// DeploySuperchain.s.sol output
type SuperchainContracts struct {
	SuperchainProxyAdminImpl common.Address
	SuperchainConfigProxy    common.Address
	SuperchainConfigImpl     common.Address
	ProtocolVersionsProxy    common.Address
	ProtocolVersionsImpl     common.Address
}

// DeployImplementations.s.sol output
type ImplementationsContracts struct {
	OpcmImpl                         common.Address
	OpcmContractsContainerImpl       common.Address
	OpcmGameTypeAdderImpl            common.Address
	OpcmDeployerImpl                 common.Address
	OpcmUpgraderImpl                 common.Address
	OpcmInteropMigratorImpl          common.Address
	DelayedWethImpl                  common.Address
	OptimismPortalImpl               common.Address
	EthLockboxImpl                   common.Address
	PreimageOracleImpl               common.Address
	MipsImpl                         common.Address
	SystemConfigImpl                 common.Address
	L1CrossDomainMessengerImpl       common.Address
	L1Erc721BridgeImpl               common.Address
	L1StandardBridgeImpl             common.Address
	OptimismMintableErc20FactoryImpl common.Address
	DisputeGameFactoryImpl           common.Address
	AnchorStateRegistryImpl          common.Address
}

type OpChainContracts struct {
	Core        *OpChainCoreContracts
	FaultProofs *OpChainFaultProofsContracts
	AltDA       *OpChainAltDAContracts
	Legacy      *OpChainLegacyContracts
}

// DeployOPChain.s.sol output
type OpChainCoreContracts struct {
	OpChainProxyAdminImpl             common.Address
	AddressManagerImpl                common.Address
	L1Erc721BridgeProxy               common.Address
	SystemConfigProxy                 common.Address
	OptimismMintableErc20FactoryProxy common.Address
	L1StandardBridgeProxy             common.Address
	L1CrossDomainMessengerProxy       common.Address
}

// DeployOPChain.s.sol output
type OpChainFaultProofsContracts struct {
	OptimismPortalProxy                common.Address
	EthLockboxProxy                    common.Address `evm:"ethLockboxProxy"`
	DisputeGameFactoryProxy            common.Address
	AnchorStateRegistryProxy           common.Address
	FaultDisputeGameImpl               common.Address
	PermissionedDisputeGameImpl        common.Address
	DelayedWethPermissionedGameProxy   common.Address
	DelayedWethPermissionlessGameProxy common.Address
}

// DeployAltDA.s.sol output
type OpChainAltDAContracts struct {
	AltDAChallengeProxy common.Address
	AltDAChallengeImpl  common.Address
}

type OpChainLegacyContracts struct {
	L2OutputOracleProxy common.Address
}
