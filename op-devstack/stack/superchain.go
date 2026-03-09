package stack

import (
	"github.com/ethereum/go-ethereum/common"
)

type SuperchainDeployment interface {
	ProtocolVersionsAddr() common.Address
	SuperchainConfigAddr() common.Address
}

// Superchain is a collection of L2 chains with common rules and shared configuration on L1
type Superchain interface {
	Common
	ID() ComponentID

	Deployment() SuperchainDeployment
}
