package stack

import (
	"log/slog"

	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// FaucetID identifies a Faucet by name and chainID, is type-safe, and can be value-copied and used as map key.
type FaucetID idWithChain

var _ IDWithChain = (*FaucetID)(nil)

const FaucetKind Kind = "Faucet"

func NewFaucetID(key string, chainID eth.ChainID) FaucetID {
	return FaucetID{
		key:     key,
		chainID: chainID,
	}
}

func (id FaucetID) String() string {
	return idWithChain(id).string(FaucetKind)
}

func (id FaucetID) ChainID() eth.ChainID {
	return idWithChain(id).chainID
}

func (id FaucetID) Kind() Kind {
	return FaucetKind
}

func (id FaucetID) Key() string {
	return id.key
}

func (id FaucetID) LogValue() slog.Value {
	return slog.StringValue(id.String())
}

func (id FaucetID) MarshalText() ([]byte, error) {
	return idWithChain(id).marshalText(FaucetKind)
}

func (id *FaucetID) UnmarshalText(data []byte) error {
	return (*idWithChain)(id).unmarshalText(FaucetKind, data)
}

func SortFaucetIDs(ids []FaucetID) []FaucetID {
	return copyAndSort(ids, func(a, b FaucetID) bool {
		return lessIDWithChain(idWithChain(a), idWithChain(b))
	})
}

func SortFaucets(elems []Faucet) []Faucet {
	return copyAndSort(elems, func(a, b Faucet) bool {
		return lessIDWithChain(idWithChain(a.ID()), idWithChain(b.ID()))
	})
}

var _ FaucetMatcher = FaucetID{}

func (id FaucetID) Match(elems []Faucet) []Faucet {
	return findByID(id, elems)
}

type Faucet interface {
	Common
	ID() FaucetID
	API() apis.Faucet
}
