package addresses

import "github.com/ethereum/go-ethereum/common"

// This file contains the standard structs and contract names for all L1 contracts involved in an OpChain
//   - structs are namespaced by deployment contract bundle (Superchain, Implementations and OpChain), then
//     by feature set (e.g. FaultProofs has its own set of contracts nested within the OpChainContracts struct)
//   - all contract field names are suffixed with "Impl" or "Proxy" to indicate the type of contract

type L1Contracts struct {
	SuperchainContracts
	ImplementationsContracts
	OpChainContracts
}

// SuperchainContracts struct contains all the superchain-level contracts
//   - these contracts are shared by all OpChains that are members of the same superchain
type SuperchainContracts struct {
	SuperchainProxyAdminImpl common.Address
	SuperchainConfigProxy    common.Address
	SuperchainConfigImpl     common.Address
	ProtocolVersionsProxy    common.Address
	ProtocolVersionsImpl     common.Address
}

// ImplementationsContracts struct contains all the implementation contracts for a superchain
//   - these contracts are shared by all OpChains that are members of the same superchain
//   - these contracts are not upgradable, but can be replaced by new contract releases/deployments
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

// OpChainContracts struct contains all the contracts for a specific L2 OpChain
//   - these contracts are not shared by any other OpChains
//   - these contracts are mostly proxies, which point to ImplementationContracts
//   - feature sets are represented by nested structs, which are inlined so that individual contracts
//     can be accessed directly on the OpChainContracts struct (i.e. no leaky abstraction)
type OpChainContracts struct {
	OpChainCoreContracts
	OpChainFaultProofsContracts
	OpChainAltDAContracts
	OpChainLegacyContracts
}

// OpChainCoreContracts struct contains contracts that all L2s need, regardless of feature set
type OpChainCoreContracts struct {
	OpChainProxyAdminImpl             common.Address
	OptimismPortalProxy               common.Address
	AddressManagerImpl                common.Address
	L1Erc721BridgeProxy               common.Address
	SystemConfigProxy                 common.Address
	OptimismMintableErc20FactoryProxy common.Address
	L1StandardBridgeProxy             common.Address
	L1CrossDomainMessengerProxy       common.Address
	EthLockboxProxy                   common.Address
}

type OpChainFaultProofsContracts struct {
	DisputeGameFactoryProxy            common.Address
	AnchorStateRegistryProxy           common.Address
	FaultDisputeGameImpl               common.Address
	PermissionedDisputeGameImpl        common.Address
	DelayedWethPermissionedGameProxy   common.Address
	DelayedWethPermissionlessGameProxy common.Address
}

type OpChainAltDAContracts struct {
	AltDAChallengeProxy common.Address
	AltDAChallengeImpl  common.Address
}

type OpChainLegacyContracts struct {
	L2OutputOracleProxy common.Address
}
