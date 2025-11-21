package stack

import (
	"fmt"

	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

const L1CLNodeKind Kind = "L1CLNode"

func NewL1CLNodeID(key string, chainID eth.ChainID) ComponentID {
	return ComponentID{
		Kind: L1CLNodeKind,
		Key:  fmt.Sprintf("%s-%s", key, chainID.String()),
	}
}

func SortL1CLNodes(elems []L1CLNode) []L1CLNode {
	return copyAndSort(elems, func(a, b L1CLNode) bool {
		return isLess(a.ID(), b.ID())
	})
}

// L1CLNode is a L1 ethereum consensus-layer node, aka Beacon node.
// This node may not be a full beacon node, and instead run a mock L1 consensus node.
type L1CLNode interface {
	Common
	ID() ComponentID
	ChainID() eth.ChainID

	BeaconClient() apis.BeaconClient
}
