package stack

// L2BatcherID identifies a L2Batcher by name and chainID, is type-safe, and can be value-copied and used as map key.
type L2BatcherID idWithChain

const L2BatcherKind Kind = "L2Batcher"

func (id L2BatcherID) String() string {
	return idWithChain(id).string(L2BatcherKind)
}

func (id L2BatcherID) MarshalText() ([]byte, error) {
	return idWithChain(id).marshalText(L2BatcherKind)
}

func (id *L2BatcherID) UnmarshalText(data []byte) error {
	return (*idWithChain)(id).unmarshalText(L2BatcherKind, data)
}

func SortL2BatcherIDs(ids []L2BatcherID) []L2BatcherID {
	return copyAndSort(ids, func(a, b L2BatcherID) bool {
		return lessIDWithChain(idWithChain(a), idWithChain(b))
	})
}

func SortL2Batchers(elems []L2Batcher) []L2Batcher {
	return copyAndSort(elems, func(a, b L2Batcher) bool {
		return lessIDWithChain(idWithChain(a.ID()), idWithChain(b.ID()))
	})
}

var _ L2BatcherMatcher = L2BatcherID{}

func (id L2BatcherID) Match(elems []L2Batcher) []L2Batcher {
	return findByID(id, elems)
}

// L2Batcher represents an L2 batch-submission service, posting L2 data of an L2 to L1.
type L2Batcher interface {
	Common
	ID() L2BatcherID

	// API to interact with batcher will be added here later
}
