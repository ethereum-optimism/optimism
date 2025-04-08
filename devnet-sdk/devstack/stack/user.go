package stack

import (
	"crypto/ecdsa"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
)

// UserID identifies a User by name and chainID, is type-safe, and can be value-copied and used as map key.
type UserID idWithChain

const UserKind Kind = "User"

func (id UserID) String() string {
	return idWithChain(id).string(UserKind)
}

func (id UserID) MarshalText() ([]byte, error) {
	return idWithChain(id).marshalText(UserKind)
}

func (id *UserID) UnmarshalText(data []byte) error {
	return (*idWithChain)(id).unmarshalText(UserKind, data)
}

func SortUserIDs(ids []UserID) []UserID {
	return copyAndSort(ids, func(a, b UserID) bool {
		return lessIDWithChain(idWithChain(a), idWithChain(b))
	})
}

func SortUsers(elems []User) []User {
	return copyAndSort(elems, func(a, b User) bool {
		return lessIDWithChain(idWithChain(a.ID()), idWithChain(b.ID()))
	})
}

var _ UserMatcher = UserID{}

func (id UserID) Match(elems []User) []User {
	return findByID(id, elems)
}

// User represents a single user-key, specific to a single chain,
// with a default connection to interact with the execution-layer of said chain.
type User interface {
	Common

	ID() UserID

	Key() *ecdsa.PrivateKey
	Address() common.Address

	ChainID() eth.ChainID

	// EL is the default node used to interact with the chain
	EL() ELNode
}
