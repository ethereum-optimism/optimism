package stack

import "github.com/ethereum/go-ethereum/common"

type SuperchainDeployment interface {
	ProtocolVersionsAddr() common.Address
	SuperchainConfigAddr() common.Address
}

// SuperchainID identifies a Superchain by name, is type-safe, and can be value-copied and used as map key.
type SuperchainID genericID

const SuperchainKind Kind = "Superchain"

func (id SuperchainID) String() string {
	return genericID(id).string(SuperchainKind)
}

func (id SuperchainID) MarshalText() ([]byte, error) {
	return genericID(id).marshalText(SuperchainKind)
}

func (id *SuperchainID) UnmarshalText(data []byte) error {
	return (*genericID)(id).unmarshalText(SuperchainKind, data)
}

func SortSuperchainIDs(ids []SuperchainID) []SuperchainID {
	return copyAndSortCmp(ids)
}

func SortSuperchains(elems []Superchain) []Superchain {
	return copyAndSort(elems, lessElemOrdered[SuperchainID, Superchain])
}

var _ SuperchainMatcher = SuperchainID("")

func (id SuperchainID) Match(elems []Superchain) []Superchain {
	return findByID(id, elems)
}

// Superchain is a collection of L2 chains with common rules and shared configuration on L1
type Superchain interface {
	Common
	ID() SuperchainID

	Deployment() SuperchainDeployment
}
