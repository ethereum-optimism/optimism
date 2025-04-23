package stack

import "github.com/ethereum-optimism/optimism/op-service/eth"

// L1NetworkID identifies a L1Network by name and chainID, is type-safe, and can be value-copied and used as map key.
type L1NetworkID idOnlyChainID

const L1NetworkKind Kind = "L1Network"

func (id L1NetworkID) ChainID() eth.ChainID {
	return idOnlyChainID(id).ChainID()
}

func (id L1NetworkID) String() string {
	return idOnlyChainID(id).string(L1NetworkKind)
}

func (id L1NetworkID) MarshalText() ([]byte, error) {
	return idOnlyChainID(id).marshalText(L1NetworkKind)
}

func (id *L1NetworkID) UnmarshalText(data []byte) error {
	return (*idOnlyChainID)(id).unmarshalText(L1NetworkKind, data)
}

func SortL1NetworkIDs(ids []L1NetworkID) []L1NetworkID {
	return copyAndSort(ids, func(a, b L1NetworkID) bool {
		return lessIDOnlyChainID(idOnlyChainID(a), idOnlyChainID(b))
	})
}

func SortL1Networks(elems []L1Network) []L1Network {
	return copyAndSort(elems, func(a, b L1Network) bool {
		return lessIDOnlyChainID(idOnlyChainID(a.ID()), idOnlyChainID(b.ID()))
	})
}

var _ L1NetworkMatcher = L1NetworkID{}

func (id L1NetworkID) Match(elems []L1Network) []L1Network {
	return findByID(id, elems)
}

// L1Network represents a L1 chain, a collection of configuration and node resources.
type L1Network interface {
	Network
	ID() L1NetworkID

	L1ELNode(m L1ELMatcher) L1ELNode
	L1CLNode(m L1CLMatcher) L1CLNode

	L1ELNodeIDs() []L1ELNodeID
	L1CLNodeIDs() []L1CLNodeID

	L1ELNodes() []L1ELNode
	L1CLNodes() []L1CLNode
}

type ExtensibleL1Network interface {
	ExtensibleNetwork
	L1Network
	AddL1ELNode(v L1ELNode)
	AddL1CLNode(v L1CLNode)
}
