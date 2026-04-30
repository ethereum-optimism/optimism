package stack

import (
	"github.com/ethereum/go-ethereum/common"
)

type SuperchainDeployment interface {
	SuperchainConfigAddr() common.Address
}
