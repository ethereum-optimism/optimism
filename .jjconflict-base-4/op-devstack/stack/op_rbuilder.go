package stack

import (
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

const OPRBuilderNodeKind Kind = "OPRBuilderNode"

func NewOPRBuilderNodeID(key string, chainID eth.ChainID) ComponentID {
	return ComponentID{
		Kind: OPRBuilderNodeKind,
		Key:  key,
	}
}

func SortOPRBuilderNodes(elems []OPRBuilderNode) []OPRBuilderNode {
	return copyAndSort(elems, func(a, b OPRBuilderNode) bool {
		return isLess(a.ID(), b.ID())
	})
}

// L2ELNode is a L2 ethereum execution-layer node
type OPRBuilderNode interface {
	FlashblocksClient() *client.WSClient

	L2ELNode
}
