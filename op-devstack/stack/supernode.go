package stack

import (
	"fmt"
	"log/slog"

	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// SupernodeID identifies a Supernode by name, is type-safe, and can be value-copied and used as map key.
type SupernodeID genericID

var _ GenericID = (*SupernodeID)(nil)

const SupernodeKind Kind = "Supernode"

func NewSupernodeID(key string, chains ...eth.ChainID) SupernodeID {
	var s string
	for _, chain := range chains {
		s += chain.String()
	}
	return SupernodeID(fmt.Sprintf("%s-%s", key, s))
}

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

// InteropTestControl provides integration test control methods for the interop activity.
// This interface is for integration test control only.
type InteropTestControl interface {
	// PauseInteropActivity pauses the interop activity at the given timestamp.
	// When the interop activity attempts to process this timestamp, it returns early.
	// This function is for integration test control only.
	PauseInteropActivity(ts uint64)

	// ResumeInteropActivity clears any pause on the interop activity, allowing normal processing.
	// This function is for integration test control only.
	ResumeInteropActivity()
}
