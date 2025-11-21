package stack

import (
	"github.com/ethereum-optimism/optimism/op-service/apis"
)

const L2ELNodeKind Kind = "L2ELNode"

func NewL2ELNodeID(key string) ComponentID {
	return ComponentID{
		Kind: L2ELNodeKind,
		Key:  key,
	}
}

func SortL2ELNodes(elems []L2ELNode) []L2ELNode {
	return copyAndSort(elems, func(a, b L2ELNode) bool {
		return isLess(a.ID(), b.ID())
	})
}

// L2ELNode is a L2 ethereum execution-layer node
type L2ELNode interface {
	L2EthClient() apis.L2EthClient
	L2EngineClient() apis.EngineClient

	ELNode
}
