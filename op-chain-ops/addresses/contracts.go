package addresses

import "github.com/ethereum/go-ethereum/common"

type L1Contracts struct {
	Superchain      *SuperchainContracts
	Implementations *ImplementationsContracts
	OpChain         *OpChainContracts
}

type SuperchainContracts struct {
	SuperchainProxyAdminImpl common.Address
	SuperchainConfigProxy    common.Address
	SuperchainConfigImpl     common.Address
	ProtocolVersionsProxy    common.Address
	ProtocolVersionsImpl     common.Address
}

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
	OpChainCoreContracts
	OpChainFaultProofsContracts
	OpChainAltDAContracts
	OpChainLegacyContracts
}

type OpChainCoreContracts struct {
	OpChainProxyAdminImpl             common.Address
	OptimismPortalProxy               common.Address
	AddressManagerImpl                common.Address
	L1Erc721BridgeProxy               common.Address
	SystemConfigProxy                 common.Address
	OptimismMintableErc20FactoryProxy common.Address
	L1StandardBridgeProxy             common.Address
	L1CrossDomainMessengerProxy       common.Address
}

type OpChainFaultProofsContracts struct {
	DisputeGameFactoryProxy            common.Address
	AnchorStateRegistryProxy           common.Address
	FaultDisputeGameImpl               common.Address
	PermissionedDisputeGameImpl        common.Address
	DelayedWethPermissionedGameProxy   common.Address
	DelayedWethPermissionlessGameProxy common.Address
	EthLockboxProxy                    common.Address `evm:"ethLockboxProxy"`
}

type OpChainAltDAContracts struct {
	AltDAChallengeProxy common.Address
	AltDAChallengeImpl  common.Address
}

type OpChainLegacyContracts struct {
	L2OutputOracleProxy common.Address
}
