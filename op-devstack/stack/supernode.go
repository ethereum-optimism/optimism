package stack

import (
	"log/slog"

	"github.com/ethereum-optimism/optimism/op-service/apis"
)

// SupernodeID identifies a Supernode by name, is type-safe, and can be value-copied and used as map key.
type SupernodeID genericID

var _ GenericID = (*SupernodeID)(nil)

const SupernodeKind Kind = "Supernode"

func (id SupernodeID) String() string {
	return genericID(id).string(SupernodeKind)
}

func (id SupernodeID) Kind() Kind {
	return SupernodeKind
}

func (id SupernodeID) LogValue() slog.Value {
	return slog.StringValue(id.String())
}

func (id SupernodeID) MarshalText() ([]byte, error) {
	return genericID(id).marshalText(SupernodeKind)
}

func (id *SupernodeID) UnmarshalText(data []byte) error {
	return (*genericID)(id).unmarshalText(SupernodeKind, data)
}

func SortSupernodeIDs(ids []SupernodeID) []SupernodeID {
	return copyAndSortCmp(ids)
}

func SortSupernodes(elems []Supernode) []Supernode {
	return copyAndSort(elems, lessElemOrdered[SupernodeID, Supernode])
}

var _ SupernodeMatcher = SupernodeID("")

func (id SupernodeID) Match(elems []Supernode) []Supernode {
	return findByID(id, elems)
}

type Supernode interface {
	Common
	ID() SupernodeID
	QueryAPI() apis.SupernodeQueryAPI
}
