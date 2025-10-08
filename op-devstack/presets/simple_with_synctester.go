package presets

import (
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/shim"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-devstack/stack/match"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
)

type SimpleWithSyncTester struct {
	Minimal

	SyncTester     *dsl.SyncTester
	SyncTesterL2EL *dsl.L2ELNode
	L2CL2          *dsl.L2CLNode
}

func WithSimpleWithSyncTester() stack.CommonOption {
	return stack.MakeCommon(sysgo.DefaultSimpleSystemWithSyncTester(&sysgo.DefaultSimpleSystemWithSyncTesterIDs{}))
}

func NewSimpleWithSyncTester(t devtest.T) *SimpleWithSyncTester {
	system := shim.NewSystem(t)
	orch := Orchestrator()
	orch.Hydrate(system)
	minimal := minimalFromSystem(t, system, orch)
	l2 := system.L2Network(match.L2ChainA)
	syncTester := l2.SyncTester(match.FirstSyncTester)

	// L2CL connected to L2EL initialized by sync tester
	l2CL2 := l2.L2CLNode(match.SecondL2CL)
	// L2EL initialized by sync tester
	syncTesterL2EL := l2.L2ELNode(match.SecondL2EL)

	return &SimpleWithSyncTester{
		Minimal:        *minimal,
		SyncTester:     dsl.NewSyncTester(syncTester),
		SyncTesterL2EL: dsl.NewL2ELNode(syncTesterL2EL, orch.ControlPlane()),
		L2CL2:          dsl.NewL2CLNode(l2CL2, orch.ControlPlane()),
	}
}

func WithHardforkSequentialActivation(startFork, endFork rollup.ForkName, delta uint64) stack.CommonOption {
	return stack.MakeCommon(sysgo.WithDeployerOptions(sysgo.WithHardforkSequentialActivation(startFork, endFork, &delta)))
}
