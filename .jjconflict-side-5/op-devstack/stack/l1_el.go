package stack

import (
	"fmt"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

const L1ELNodeKind Kind = "L1ELNode"

func NewL1ELNodeID(key string, chainID eth.ChainID) ComponentID {
	return ComponentID{
		Kind: L1ELNodeKind,
		Key:  fmt.Sprintf("%s-%s", key, chainID.String()),
	}
}

func SortL1ELNodes(elems []L1ELNode) []L1ELNode {
	return copyAndSort(elems, func(a, b L1ELNode) bool {
		return isLess(ComponentID(a.ID()), ComponentID(b.ID()))
	})
}

// L1ELNode is a L1 ethereum execution-layer node
type L1ELNode interface {
	ELNode
}
