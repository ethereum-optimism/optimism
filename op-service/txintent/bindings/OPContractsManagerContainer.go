package bindings

import (
	"github.com/ethereum/go-ethereum/common"
)

// OPContractsManagerContainerImplementations mirrors the Solidity struct
// OPContractsManagerContainer.Implementations. Field order MUST match the
// Solidity declaration so ABI decoding lines up.
type OPContractsManagerContainerImplementations struct {
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
	FaultDisputeGameImpl             common.Address
	PermissionedDisputeGameImpl      common.Address
	SuperFaultDisputeGameImpl        common.Address
	SuperPermissionedDisputeGameImpl common.Address
	ZkDisputeGameImpl                common.Address
	StorageSetterImpl                common.Address
}

type OPContractsManagerContainer struct {
	Implementations func() TypedCall[OPContractsManagerContainerImplementations] `sol:"implementations"`
}

func NewOPContractsManagerContainer(opts ...CallFactoryOption) *OPContractsManagerContainer {
	c := NewBindings[OPContractsManagerContainer](opts...)
	return &c
}
