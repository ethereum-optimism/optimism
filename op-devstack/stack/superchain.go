package stack

import (
	"github.com/ethereum/go-ethereum/common"
)

type SuperchainDeployment interface {
	ProtocolVersionsAddr() common.Address
	SuperchainConfigAddr() common.Address
}
