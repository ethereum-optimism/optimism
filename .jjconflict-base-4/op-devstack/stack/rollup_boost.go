package stack

import (
	"github.com/ethereum-optimism/optimism/op-service/client"
)

// RollupBoostNodeID identifies a RollupBoost node by name and chainID, is type-safe, and can be value-copied and used as map key.
type RollupBoostNodeID ComponentID

const RollupBoostNodeKind Kind = "RollupBoostNode"

func NewRollupBoostNodeID(key string) ComponentID {
	return ComponentID{
		Kind: RollupBoostNodeKind,
		Key:  key,
	}
}

func SortRollupBoostNodes(elems []RollupBoostNode) []RollupBoostNode {
	return copyAndSort(elems, func(a, b RollupBoostNode) bool {
		return isLess(a.ID(), b.ID())
	})
}

// RollupBoostNode is a shim service between an L2 consensus-layer node and an L2 ethereum execution-layer node
type RollupBoostNode interface {
	FlashblocksClient() *client.WSClient

	L2ELNode
}
