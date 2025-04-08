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

// L1Network represents a L1 chain, a collection of configuration and node resources.
type L1Network interface {
	Network
	ID() L1NetworkID

	L1ELNode(id L1ELNodeID) L1ELNode
	L1CLNode(id L1CLNodeID) L1CLNode

	L1ELNodes() []L1ELNodeID
	L1CLNodes() []L1CLNodeID
}

type ExtensibleL1Network interface {
	ExtensibleNetwork
	L1Network
	AddL1ELNode(v L1ELNode)
	AddL1CLNode(v L1CLNode)
}
