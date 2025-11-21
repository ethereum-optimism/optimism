package stack

import (
	"github.com/ethereum-optimism/optimism/op-service/apis"
)

// SupervisorID identifies a Supervisor by name and chainID, is type-safe, and can be value-copied and used as map key.
type SupervisorID ComponentID

const SupervisorKind Kind = "Supervisor"

func NewSupervisorID(key string) ComponentID {
	return ComponentID{
		Kind: SupervisorKind,
		Key:  key,
	}
}

func SortSupervisors(elems []Supervisor) []Supervisor {
	return copyAndSort(elems, func(a, b Supervisor) bool {
		return isLess(a.ID(), b.ID())
	})
}

// Supervisor is an interop service, used to cross-verify messages between chains.
type Supervisor interface {
	Common
	ID() ComponentID

	AdminAPI() apis.SupervisorAdminAPI
	QueryAPI() apis.SupervisorQueryAPI
}
