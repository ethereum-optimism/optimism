package stack

// L2ChallengerID identifies a L2Challenger by name and chainID, is type-safe, and can be value-copied and used as map key.
type L2ChallengerID idWithChain

const L2ChallengerKind Kind = "L2Challenger"

func (id L2ChallengerID) String() string {
	return idWithChain(id).string(L2ChallengerKind)
}

func (id L2ChallengerID) MarshalText() ([]byte, error) {
	return idWithChain(id).marshalText(L2ChallengerKind)
}

func (id *L2ChallengerID) UnmarshalText(data []byte) error {
	return (*idWithChain)(id).unmarshalText(L2ChallengerKind, data)
}

func SortL2ChallengerIDs(ids []L2ChallengerID) []L2ChallengerID {
	return copyAndSort(ids, func(a, b L2ChallengerID) bool {
		return lessIDWithChain(idWithChain(a), idWithChain(b))
	})
}

type L2Challenger interface {
	Common
	ID() L2ChallengerID
}
