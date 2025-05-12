package stack

// L2ChallengerID identifies a L2Challenger by name and chainID, is type-safe, and can be value-copied and used as map key.
type L2ChallengerID idWithCluster

const L2ChallengerKind Kind = "L2Challenger"

func (id L2ChallengerID) String() string {
	return idWithCluster(id).string(L2ChallengerKind)
}

func (id L2ChallengerID) MarshalText() ([]byte, error) {
	return idWithCluster(id).marshalText(L2ChallengerKind)
}

func (id *L2ChallengerID) UnmarshalText(data []byte) error {
	return (*idWithCluster)(id).unmarshalText(L2ChallengerKind, data)
}

func SortL2ChallengerIDs(ids []L2ChallengerID) []L2ChallengerID {
	return copyAndSort(ids, func(a, b L2ChallengerID) bool {
		return lessIDWithCluster(idWithCluster(a), idWithCluster(b))
	})
}

func SortL2Challengers(elems []L2Challenger) []L2Challenger {
	return copyAndSort(elems, func(a, b L2Challenger) bool {
		return lessIDWithCluster(idWithCluster(a.ID()), idWithCluster(b.ID()))
	})
}

var _ L2ChallengerMatcher = L2ChallengerID{}

func (id L2ChallengerID) Match(elems []L2Challenger) []L2Challenger {
	return findByID(id, elems)
}

type L2Challenger interface {
	Common
	ID() L2ChallengerID
}
