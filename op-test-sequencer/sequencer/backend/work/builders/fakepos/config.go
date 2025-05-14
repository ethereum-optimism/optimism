package fakepos

import (
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/fakebeacon"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/geth"
)

type Config struct {
	Geth              *geth.GethInstance
	Beacon            *fakebeacon.FakeBeacon
	FinalizedDistance uint64
	SafeDistance      uint64
	BlockTime         uint64
}
