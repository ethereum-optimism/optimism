package stack

import (
	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// TestSequencerID identifies a TestSequencer by name and chainID, is type-safe, and can be value-copied and used as map key.
type TestSequencerID genericID

const TestSequencerKind Kind = "TestSequencer"

func (id TestSequencerID) String() string {
	return genericID(id).string(TestSequencerKind)
}

func (id TestSequencerID) MarshalText() ([]byte, error) {
	return genericID(id).marshalText(TestSequencerKind)
}

func (id *TestSequencerID) UnmarshalText(data []byte) error {
	return (*genericID)(id).unmarshalText(TestSequencerKind, data)
}

func SortTestSequencerIDs(ids []TestSequencerID) []TestSequencerID {
	return copyAndSortCmp(ids)
}

func SortTestSequencers(elems []TestSequencer) []TestSequencer {
	return copyAndSort(elems, lessElemOrdered[TestSequencerID, TestSequencer])
}

// TestSequencer
type TestSequencer interface {
	Common
	ID() TestSequencerID

	AdminAPI() apis.TestSequencerAdminAPI
	BuildAPI() apis.TestSequencerBuildAPI
	IndividualAPI(chainID eth.ChainID) apis.TestSequencerIndividualAPI
}
