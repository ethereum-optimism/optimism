package stack

import (
	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type SyncTester interface {
	Common
	ChainID() eth.ChainID
	API() apis.SyncTester

	APIWithSession(sessionID string) apis.SyncTester
}
