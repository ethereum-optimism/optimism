package stack

import (
	"fmt"
	"sort"

	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// SupernodeID is kept as a semantic alias for ComponentID.
// Supernode IDs are key-only IDs with KindSupernode.
type SupernodeID = ComponentID

func NewSupernodeID(key string, chains ...eth.ChainID) SupernodeID {
	var suffix string
	for _, chain := range chains {
		suffix += chain.String()
	}
	return NewComponentIDKeyOnly(KindSupernode, fmt.Sprintf("%s-%s", key, suffix))
}

func SortSupernodeIDs(ids []SupernodeID) []SupernodeID {
	out := append([]SupernodeID(nil), ids...)
	sort.Slice(out, func(i, j int) bool {
		return out[i].Less(out[j])
	})
	return out
}

func SortSupernodes(elems []Supernode) []Supernode {
	out := append([]Supernode(nil), elems...)
	sort.Slice(out, func(i, j int) bool {
		return out[i].ID().Less(out[j].ID())
	})
	return out
}

type Supernode interface {
	Common
	ID() ComponentID
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
