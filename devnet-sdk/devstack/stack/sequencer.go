package stack

import (
	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// SequencerID identifies a Sequencer by name and chainID, is type-safe, and can be value-copied and used as map key.
type SequencerID genericID

const SequencerKind Kind = "Sequencer"

func (id SequencerID) String() string {
	return genericID(id).string(SequencerKind)
}

func (id SequencerID) MarshalText() ([]byte, error) {
	return genericID(id).marshalText(SequencerKind)
}

func (id *SequencerID) UnmarshalText(data []byte) error {
	return (*genericID)(id).unmarshalText(SequencerKind, data)
}

func SortSequencerIDs(ids []SequencerID) []SequencerID {
	return copyAndSortCmp(ids)
}

func SortSequencers(elems []Sequencer) []Sequencer {
	return copyAndSort(elems, lessElemOrdered[SequencerID, Sequencer])
}

// Sequencer
type Sequencer interface {
	Common
	ID() SequencerID

	AdminAPI() apis.SequencerAdminAPI
	BuildAPI() apis.SequencerBuildAPI
	IndividualAPI(chainID eth.ChainID) apis.SequencerIndividualAPI
}
