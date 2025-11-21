package stack

import (
	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

const FaucetKind Kind = "Faucet"

func NewFaucetID(key string) ComponentID {
	return ComponentID{
		Kind: FaucetKind,
		Key:  key,
	}
}

func SortFaucets(elems []Faucet) []Faucet {
	return copyAndSort(elems, func(a, b Faucet) bool {
		return isLess(a.ID(), b.ID())
	})
}

type Faucet interface {
	Common
	ID() ComponentID
	ChainID() eth.ChainID
	API() apis.Faucet
}
