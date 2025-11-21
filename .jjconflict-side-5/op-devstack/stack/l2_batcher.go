package stack

import (
	"fmt"

	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

const L2BatcherKind Kind = "L2Batcher"

func NewL2BatcherID(key string, chainID eth.ChainID) ComponentID {
	return ComponentID{
		Kind: L2BatcherKind,
		Key:  fmt.Sprintf("%s-%s", key, chainID),
	}
}

func SortL2Batchers(elems []L2Batcher) []L2Batcher {
	return copyAndSort(elems, func(a, b L2Batcher) bool {
		return isLess(a.ID(), b.ID())
	})
}

// L2Batcher represents an L2 batch-submission service, posting L2 data of an L2 to L1.
type L2Batcher interface {
	Common
	ID() ComponentID
	ChainID() eth.ChainID
	ActivityAPI() apis.BatcherActivity
}
