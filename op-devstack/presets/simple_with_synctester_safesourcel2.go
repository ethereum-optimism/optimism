package presets

import (
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/shim"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-devstack/stack/match"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
)

type SimpleWithSyncTesterSafeSourceL2 struct {
	Minimal

	SyncTester     *dsl.SyncTester
	SyncTesterL2EL *dsl.L2ELNode
	L2CL2          *dsl.L2CLNode
}

func WithSimpleWithSyncTesterSafeSourceL2() stack.CommonOption {
	return stack.MakeCommon(sysgo.DefaultSimpleSystemWithSyncTesterSafeSourceL2(&sysgo.DefaultSimpleSystemWithSyncTesterSafeSourceL2IDs{}))
}

func NewSimpleWithSyncTesterSafeSourceL2(t devtest.T) *SimpleWithSyncTesterSafeSourceL2 {
	system := shim.NewSystem(t)
	orch := Orchestrator()
	orch.Hydrate(system)
	minimal := minimalFromSystem(t, system, orch)
	l2 := system.L2Network(match.L2ChainA)
	syncTester := l2.SyncTester(match.FirstSyncTester)

	// L2CL2 connected to L2EL initialized by sync tester, with safe-source=l2
	l2CL2 := l2.L2CLNode(match.SecondL2CL)
	// L2EL initialized by sync tester
	syncTesterL2EL := l2.L2ELNode(match.SecondL2EL)

	return &SimpleWithSyncTesterSafeSourceL2{
		Minimal:        *minimal,
		SyncTester:     dsl.NewSyncTester(syncTester),
		SyncTesterL2EL: dsl.NewL2ELNode(syncTesterL2EL, orch.ControlPlane()),
		L2CL2:          dsl.NewL2CLNode(l2CL2, orch.ControlPlane()),
	}
}
