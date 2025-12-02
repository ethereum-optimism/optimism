package stack

import (
	"log/slog"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// L2ProposerID identifies a L2Proposer by name and chainID, is type-safe, and can be value-copied and used as map key.
type L2ProposerID idWithChain

var _ IDWithChain = (*L2ProposerID)(nil)

const L2ProposerKind Kind = "L2Proposer"

func NewL2ProposerID(key string, chainID eth.ChainID) L2ProposerID {
	return L2ProposerID{
		key:     key,
		chainID: chainID,
	}
}

func (id L2ProposerID) String() string {
	return idWithChain(id).string(L2ProposerKind)
}

func (id L2ProposerID) ChainID() eth.ChainID {
	return idWithChain(id).chainID
}

func (id L2ProposerID) Kind() Kind {
	return L2ProposerKind
}

func (id L2ProposerID) Key() string {
	return id.key
}

func (id L2ProposerID) LogValue() slog.Value {
	return slog.StringValue(id.String())
}

func (id L2ProposerID) MarshalText() ([]byte, error) {
	return idWithChain(id).marshalText(L2ProposerKind)
}

func (id *L2ProposerID) UnmarshalText(data []byte) error {
	return (*idWithChain)(id).unmarshalText(L2ProposerKind, data)
}

func SortL2ProposerIDs(ids []L2ProposerID) []L2ProposerID {
	return copyAndSort(ids, func(a, b L2ProposerID) bool {
		return lessIDWithChain(idWithChain(a), idWithChain(b))
	})
}

func SortL2Proposers(elems []L2Proposer) []L2Proposer {
	return copyAndSort(elems, func(a, b L2Proposer) bool {
		return lessIDWithChain(idWithChain(a.ID()), idWithChain(b.ID()))
	})
}

var _ L2ProposerMatcher = L2ProposerID{}

func (id L2ProposerID) Match(elems []L2Proposer) []L2Proposer {
	return findByID(id, elems)
}

// L2Proposer is a L2 output proposer, posting claims of L2 state to L1.
type L2Proposer interface {
	Common
	ID() L2ProposerID
}
