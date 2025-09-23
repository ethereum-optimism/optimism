package stack

import (
	"log/slog"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// L2ChallengerID identifies a L2Challenger by name and chainID, is type-safe, and can be value-copied and used as map key.
type L2ChallengerID idWithChain

var _ IDWithChain = (*L2ChallengerID)(nil)

const L2ChallengerKind Kind = "L2Challenger"

func NewL2ChallengerID(key string, chainID eth.ChainID) L2ChallengerID {
	return L2ChallengerID{
		key:     key,
		chainID: chainID,
	}
}

func (id L2ChallengerID) String() string {
	return idWithChain(id).string(L2ChallengerKind)
}

func (id L2ChallengerID) ChainID() eth.ChainID {
	return idWithChain(id).chainID
}

func (id L2ChallengerID) Kind() Kind {
	return L2ChallengerKind
}

func (id L2ChallengerID) Key() string {
	return id.key
}

func (id L2ChallengerID) LogValue() slog.Value {
	return slog.StringValue(id.String())
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

func SortL2Challengers(elems []L2Challenger) []L2Challenger {
	return copyAndSort(elems, func(a, b L2Challenger) bool {
		return lessIDWithChain(idWithChain(a.ID()), idWithChain(b.ID()))
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
