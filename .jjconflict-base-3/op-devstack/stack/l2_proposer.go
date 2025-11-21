package stack

import "github.com/ethereum-optimism/optimism/op-service/eth"

const L2ProposerKind Kind = "L2Proposer"

func NewL2ProposerID(key string) ComponentID {
	return ComponentID{
		Kind: L2ProposerKind,
		Key:  key,
	}
}

func SortL2Proposers(elems []L2Proposer) []L2Proposer {
	return copyAndSort(elems, func(a, b L2Proposer) bool {
		return isLess(ComponentID(a.ID()), ComponentID(b.ID()))
	})
}

// L2Proposer is a L2 output proposer, posting claims of L2 state to L1.
type L2Proposer interface {
	Common
	ID() ComponentID
	ChainID() eth.ChainID
}
