package stack

import "github.com/ethereum-optimism/optimism/op-service/eth"

const L1NetworkKind Kind = "L1Network"

func NewL1NetworkID(key string) ComponentID {
	return ComponentID{
		Kind: L1NetworkKind,
		Key:  key,
	}
}

func SortL1Networks(elems []L1Network) []L1Network {
	return copyAndSort(elems, func(a, b L1Network) bool {
		return a.ID().ToBig().Cmp(b.ID().ToBig()) > 0
	})
}

// L1Network represents a L1 chain, a collection of configuration and node resources.
type L1Network interface {
	Network
	ID() eth.ChainID

	L1ELNode(m L1ELMatcher) L1ELNode
	L1CLNode(m L1CLMatcher) L1CLNode

	L1ELNodes() []L1ELNode
	L1CLNodes() []L1CLNode
}

type ExtensibleL1Network interface {
	ExtensibleNetwork
	L1Network
	AddL1ELNode(v L1ELNode)
	AddL1CLNode(v L1CLNode)
}
