package stack

import (
	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// TestSequencer
type TestSequencer interface {
	Common

	AdminAPI() apis.TestSequencerAdminAPI
	BuildAPI() apis.TestSequencerBuildAPI
	ControlAPI(chainID eth.ChainID) apis.TestSequencerControlAPI
}
