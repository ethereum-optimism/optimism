package addresses

import "github.com/ethereum/go-ethereum/common"

type L1Roles struct {
	Superchain *SuperchainRoles
	OpChain    *OpChainRoles
}

type SuperchainRoles struct {
	SuperchainProxyAdminOwner common.Address
}

type OpChainRoles struct {
	Base        *OpChainBaseRoles
	FaultProofs *OpChainFaultProofsRoles
}

type OpChainBaseRoles struct {
	SystemConfigOwner      common.Address
	OpChainProxyAdminOwner common.Address
	Guardian               common.Address
	Proposer               common.Address
	UnsafeBlockSigner      common.Address
	BatchSubmitter         common.Address
}

type OpChainFaultProofsRoles struct {
	Challenger common.Address
}
