package stack

import (
	"github.com/ethereum-optimism/optimism/op-service/apis"
)

// Supervisor is an interop service, used to cross-verify messages between chains.
type Supervisor interface {
	Common

	AdminAPI() apis.SupervisorAdminAPI
	QueryAPI() apis.SupervisorQueryAPI
}
