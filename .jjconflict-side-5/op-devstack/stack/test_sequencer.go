package stack

import (
	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

const TestSequencerKind Kind = "TestSequencer"

func NewTestSequencerID(key string) ComponentID {
	return ComponentID{
		Kind: TestSequencerKind,
		Key:  key,
	}
}

func SortTestSequencers(elems []TestSequencer) []TestSequencer {
	return copyAndSort(elems, func(a, b TestSequencer) bool {
		return isLess(a.ID(), b.ID())
	})
}

// TestSequencer
type TestSequencer interface {
	Common
	ID() ComponentID

	AdminAPI() apis.TestSequencerAdminAPI
	BuildAPI() apis.TestSequencerBuildAPI
	ControlAPI(chainID eth.ChainID) apis.TestSequencerControlAPI
}
