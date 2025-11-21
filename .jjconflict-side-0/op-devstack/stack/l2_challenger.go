package stack

import (
	"github.com/ethereum-optimism/optimism/op-challenger/config"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

const L2ChallengerKind Kind = "L2Challenger"

func NewL2ChallengerID(key string) ComponentID {
	return ComponentID{
		Kind: L2ChallengerKind,
		Key:  key,
	}
}

func SortL2Challengers(elems []L2Challenger) []L2Challenger {
	return copyAndSort(elems, func(a, b L2Challenger) bool {
		return isLess(a.ID(), b.ID())
	})
}

type L2Challenger interface {
	Common
	ID() ComponentID
	ChainID() eth.ChainID
	Config() *config.Config
}
