package stack

import (
	"log/slog"
)

// L2ChallengerID identifies a L2Challenger by name and chainID, is type-safe, and can be value-copied and used as map key.
type L2ChallengerID genericID

var _ GenericID = (*L2ChallengerID)(nil)

const L2ChallengerKind Kind = "L2Challenger"

func (id L2ChallengerID) String() string {
	return genericID(id).string(L2ChallengerKind)
}

func (id L2ChallengerID) Kind() Kind {
	return L2ChallengerKind
}

func (id L2ChallengerID) LogValue() slog.Value {
	return slog.StringValue(id.String())
}

func (id L2ChallengerID) MarshalText() ([]byte, error) {
	return genericID(id).marshalText(L2ChallengerKind)
}

func (id *L2ChallengerID) UnmarshalText(data []byte) error {
	return (*genericID)(id).unmarshalText(L2ChallengerKind, data)
}

func SortL2ChallengerIDs(ids []L2ChallengerID) []L2ChallengerID {
	return copyAndSortCmp(ids)
}

func SortL2Challengers(elems []L2Challenger) []L2Challenger {
	return copyAndSort(elems, lessElemOrdered[L2ChallengerID, L2Challenger])
}

var _ L2ChallengerMatcher = L2ChallengerID("")

func (id L2ChallengerID) Match(elems []L2Challenger) []L2Challenger {
	return findByID(id, elems)
}

type L2Challenger interface {
	Common
	ID() L2ChallengerID
}
