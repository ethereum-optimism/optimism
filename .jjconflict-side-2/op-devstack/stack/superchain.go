package stack

import (
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/depset"
	"github.com/ethereum/go-ethereum/common"
)

type SuperchainDeployment interface {
	ProtocolVersionsAddr() common.Address
	SuperchainConfigAddr() common.Address
}

const SuperchainKind Kind = "Superchain"

func NewSuperchainID(key string) ComponentID {
	return ComponentID{
		Kind: SuperchainKind,
		Key:  key,
	}
}

func SortSuperchains(elems []Superchain) []Superchain {
	return copyAndSort(elems, func(a, b Superchain) bool {
		return isLess(a.ID(), b.ID())
	})
}

// Superchain is a collection of L2 chains with common rules and shared configuration on L1
type Superchain interface {
	Common
	ID() ComponentID

	Deployment() SuperchainDeployment
	DependencySet() depset.DependencySet
}
