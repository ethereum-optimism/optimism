package stack

import (
	conductorRpc "github.com/ethereum-optimism/optimism/op-conductor/rpc"
)

type ConductorID genericID

const ConductorKind Kind = "Conductor"

func (id ConductorID) String() string {
	return genericID(id).string(ConductorKind)
}

func (id ConductorID) MarshalText() ([]byte, error) {
	return genericID(id).marshalText(ConductorKind)
}

func (id *ConductorID) UnmarshalText(data []byte) error {
	return (*genericID)(id).unmarshalText(ConductorKind, data)
}

func SortConductorIDs(ids []ConductorID) []ConductorID {
	return copyAndSortCmp(ids)
}

func SortConductors(elems []Conductor) []Conductor {
	return copyAndSort(elems, lessElemOrdered[ConductorID, Conductor])
}

var _ ConductorMatcher = ConductorID("")

func (id ConductorID) Match(elems []Conductor) []Conductor {
	return findByID(id, elems)
}

type Conductor interface {
	Common
	ID() ConductorID

	RpcAPI() conductorRpc.API
}
