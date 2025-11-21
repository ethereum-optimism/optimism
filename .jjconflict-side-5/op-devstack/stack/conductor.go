package stack

import (
	conductorRpc "github.com/ethereum-optimism/optimism/op-conductor/rpc"
)

type ConductorID ComponentID

const ConductorKind Kind = "Conductor"

func NewConductorID(key string) ComponentID {
	return ComponentID{
		Kind: ConductorKind,
		Key:  key,
	}
}

func SortConductors(elems []Conductor) []Conductor {
	return copyAndSort(elems, func(a, b Conductor) bool {
		return isLess(a.ID(), b.ID())
	})
}

type Conductor interface {
	Common
	ID() ComponentID

	RpcAPI() conductorRpc.API
}
