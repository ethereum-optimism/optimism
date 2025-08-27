package presets

import (
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/shim"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-devstack/stack/match"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
)

type MinimalWithSyncTester struct {
	Minimal

	SyncTester *dsl.SyncTester
}

func WithMinimalWithSyncTester() stack.CommonOption {
	return stack.MakeCommon(sysgo.DefaultMinimalSystemWithSyncTester(&sysgo.DefaultMinimalSystemWithSyncTesterIDs{}))
}

func NewMinimalWithSyncTester(t devtest.T) *MinimalWithSyncTester {
	system := shim.NewSystem(t)
	orch := Orchestrator()
	orch.Hydrate(system)
	minimal := minimalFromSystem(t, system, orch)
	l2 := system.L2Network(match.Assume(t, match.L2ChainA))
	syncTester := l2.SyncTester(match.Assume(t, match.FirstSyncTester))
	return &MinimalWithSyncTester{
		Minimal:    *minimal,
		SyncTester: dsl.NewSyncTester(syncTester),
	}
}
