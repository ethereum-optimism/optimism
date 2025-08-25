package presets

import (
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/shim"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-devstack/stack/match"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	sttypes "github.com/ethereum-optimism/optimism/op-sync-tester/synctester/backend/types"
)

type SimpleWithSyncTester struct {
	Minimal

	SyncTester *dsl.SyncTester
	L2CL2      *dsl.L2CLNode
}

func WithSimpleWithSyncTester(fcus sttypes.FCUState) stack.CommonOption {
	return stack.MakeCommon(sysgo.DefaultSimpleSystemWithSyncTester(&sysgo.DefaultSimpleSystemWithSyncTesterIDs{}, fcus))
}

func NewSimpleWithSyncTester(t devtest.T) *SimpleWithSyncTester {
	system := shim.NewSystem(t)
	orch := Orchestrator()
	orch.Hydrate(system)
	minimal := minimalFromSystem(t, system, orch)
	l2 := system.L2Network(match.Assume(t, match.L2ChainA))
	syncTester := l2.SyncTester(match.Assume(t, match.FirstSyncTester))
	// Get the second L2CL node (verifier) - we need to find it by name
	l2CLs := l2.L2CLNodes()
	var l2CL2 stack.L2CLNode
	for _, cl := range l2CLs {
		if cl.ID().Key() == "verifier" {
			l2CL2 = cl
			break
		}
	}
	return &SimpleWithSyncTester{
		Minimal:    *minimal,
		SyncTester: dsl.NewSyncTester(syncTester),
		L2CL2:      dsl.NewL2CLNode(l2CL2, orch.ControlPlane()),
	}
}
