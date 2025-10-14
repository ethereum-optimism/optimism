package presets

import (
    "github.com/ethereum-optimism/optimism/op-devstack/devtest"
    "github.com/ethereum-optimism/optimism/op-devstack/dsl"
    "github.com/ethereum-optimism/optimism/op-devstack/shim"
    "github.com/ethereum-optimism/optimism/op-devstack/stack"
    "github.com/ethereum-optimism/optimism/op-devstack/stack/match"
    "github.com/ethereum-optimism/optimism/op-devstack/sysgo"
)

// MultiWithSyncTester provisions two L2 CL nodes, each backed by a distinct SyncTester EL endpoint,
// and exposes handles for tests.
type MultiWithSyncTester struct {
	Minimal

	SyncTester *dsl.SyncTester

	// Two independent SyncTester-backed EL endpoints
	SyncTesterL2ELA *dsl.L2ELNode
	SyncTesterL2ELB *dsl.L2ELNode

	// Two CLs, each wired to a different SyncTester EL endpoint
	L2CL_A *dsl.L2CLNode
	L2CL_B *dsl.L2CLNode
}

// WithMultiWithSyncTester composes a minimal system with a SyncTester service, then adds two
// SyncTester EL endpoints and two CL verifier nodes connected to them.
func WithMultiWithSyncTester() stack.CommonOption {
	// Build a combined option sequence
	opt := stack.Combine[*sysgo.Orchestrator]()

	// Base minimal system with SyncTester service (no SyncTester EL endpoints yet)
	opt.Add(sysgo.DefaultMinimalSystemWithSyncTester(&sysgo.DefaultMinimalSystemWithSyncTesterIDs{}, sysgo.DefaultSyncTesterELConfig().FCUState))

	// Create two SyncTester-backed EL nodes for the default L2 chain (DefaultL2AID)
	elIDA := stack.NewL2ELNodeID("sync-tester-el-a", sysgo.DefaultL2AID)
	elIDB := stack.NewL2ELNodeID("sync-tester-el-b", sysgo.DefaultL2AID)
	// readonlyEL param is only used for chainID discovery; pass the same chainID
	opt.Add(sysgo.WithSyncTesterL2ELNode(elIDA, elIDA))
	opt.Add(sysgo.WithSyncTesterL2ELNode(elIDB, elIDB))

	// Create two verifier CL nodes, each wired to its corresponding SyncTester EL endpoint
	clIDA := stack.NewL2CLNodeID("verifier-a", sysgo.DefaultL2AID)
	clIDB := stack.NewL2CLNodeID("verifier-b", sysgo.DefaultL2AID)
	opt.Add(sysgo.WithL2CLNode(clIDA, stack.NewL1CLNodeID("l1", sysgo.DefaultL1ID), stack.NewL1ELNodeID("l1", sysgo.DefaultL1ID), elIDA))
	opt.Add(sysgo.WithL2CLNode(clIDB, stack.NewL1CLNodeID("l1", sysgo.DefaultL1ID), stack.NewL1ELNodeID("l1", sysgo.DefaultL1ID), elIDB))

	// Optionally connect the two CLs via P2P for signaling (useful for some sync scenarios)
	opt.Add(sysgo.WithL2CLP2PConnection(clIDA, clIDB))

	return stack.MakeCommon(opt)
}

// NewMultiWithSyncTester hydrates the orchestrator-backed system and returns typed handles.
func NewMultiWithSyncTester(t devtest.T) *MultiWithSyncTester {
	system := shim.NewSystem(t)
	orch := Orchestrator()
	orch.Hydrate(system)
	minimal := minimalFromSystem(t, system, orch)

	l2 := system.L2Network(match.Assume(t, match.L2ChainA))
	syncTester := l2.SyncTester(match.Assume(t, match.FirstSyncTester))

    chainID := minimal.L2EL.ID().ChainID()
    elIDA := stack.NewL2ELNodeID("sync-tester-el-a", chainID)
    elIDB := stack.NewL2ELNodeID("sync-tester-el-b", chainID)
    clIDA := stack.NewL2CLNodeID("verifier-a", chainID)
    clIDB := stack.NewL2CLNodeID("verifier-b", chainID)

    elA := dsl.NewL2ELNode(l2.L2ELNode(match.Assume(t, elIDA)), orch.ControlPlane())
    elB := dsl.NewL2ELNode(l2.L2ELNode(match.Assume(t, elIDB)), orch.ControlPlane())
    clA := l2.L2CLNode(match.Assume(t, clIDA))
    clB := l2.L2CLNode(match.Assume(t, clIDB))

	return &MultiWithSyncTester{
		Minimal:         *minimal,
		SyncTester:      dsl.NewSyncTester(syncTester),
		SyncTesterL2ELA: elA,
		SyncTesterL2ELB: elB,
		L2CL_A:          dsl.NewL2CLNode(clA, orch.ControlPlane()),
		L2CL_B:          dsl.NewL2CLNode(clB, orch.ControlPlane()),
	}
}
