package addresses

import (
	"github.com/ethereum/go-ethereum/common"
)

type L1Roles struct {
	SuperchainRoles
	OpChainRoles
}

type SuperchainRoles struct {
	SuperchainProxyAdminOwner common.Address
	SuperchainGuardian        common.Address
	ProtocolVersionsOwner     common.Address
	Challenger                common.Address
}

type OpChainRoles struct {
	OpChainCoreRoles
	OpChainFaultProofsRoles
}

type OpChainCoreRoles struct {
	SystemConfigOwner      common.Address
	OpChainProxyAdminOwner common.Address
	OpChainGuardian        common.Address
	UnsafeBlockSigner      common.Address
	BatchSubmitter         common.Address
}

type OpChainFaultProofsRoles struct {
	Proposer   common.Address
	Challenger common.Address
}
