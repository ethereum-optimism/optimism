package stack

import (
	"github.com/ethereum-optimism/optimism/op-service/apis"
)

type SyncTester interface {
	Common
	ID() ComponentID
	API() apis.SyncTester

	APIWithSession(sessionID string) apis.SyncTester
}
