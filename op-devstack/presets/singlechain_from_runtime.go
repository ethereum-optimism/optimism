package presets

import (
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

func singleChainMultiNodeFromRuntime(t devtest.T, runtime *sysgo.SingleChainRuntime, runSyncChecks bool) *SingleChainMultiNode {
	minimal := minimalFromRuntime(t, runtime)
	l2ChainID := runtime.L2Network.ChainID()
	nodeB := runtime.Nodes["b"]
	t.Require().NotNil(nodeB, "missing single-chain node b")

	l2ELB := newL2ELFrontend(
		t,
		"b",
		l2ChainID,
		nodeB.EL.UserRPC(),
		nodeB.EL.EngineRPC(),
		nodeB.EL.JWTPath(),
		runtime.L2Network.RollupConfig(),
		nodeB.EL,
	)
	l2CLB := newL2CLFrontend(
		t,
		"b",
		l2ChainID,
		nodeB.CL.UserRPC(),
		nodeB.CL,
	)
	l2CLB.attachEL(l2ELB)
	l2Net, ok := minimal.L2Chain.Escape().(*presetL2Network)
	t.Require().True(ok, "expected preset L2 network")
	l2Net.AddL2ELNode(l2ELB)
	l2Net.AddL2CLNode(l2CLB)

	preset := &SingleChainMultiNode{
		Minimal: *minimal,
		L2ELB:   dsl.NewL2ELNode(l2ELB),
		L2CLB:   dsl.NewL2CLNode(l2CLB),
	}
	if runtime.P2PEnabled {
		preset.L2CLB.ManagePeer(preset.L2CL)
	}
	if runSyncChecks {
		// Ensure the follower node is in sync with the sequencer before starting tests.
		dsl.CheckAll(t,
			preset.L2CLB.MatchedFn(preset.L2CL, types.CrossSafe, 30),
			preset.L2CLB.MatchedFn(preset.L2CL, types.LocalUnsafe, 30),
		)
	}
	return preset
}

func singleChainMultiNodeWithTestSeqFromRuntime(t devtest.T, runtime *sysgo.SingleChainRuntime) *SingleChainMultiNodeWithTestSeq {
	preset := singleChainMultiNodeFromRuntime(t, runtime, false)
	testSequencer := newTestSequencerFrontend(
		t,
		runtime.TestSequencer.Name,
		runtime.TestSequencer.AdminRPC,
		runtime.TestSequencer.ControlRPC,
		runtime.TestSequencer.JWTSecret,
	)
	return &SingleChainMultiNodeWithTestSeq{
		SingleChainMultiNode: *preset,
		TestSequencer:        dsl.NewTestSequencer(testSequencer),
	}
}

func singleChainTwoVerifiersFromRuntime(t devtest.T, runtime *sysgo.SingleChainRuntime) *SingleChainTwoVerifiers {
	base := singleChainMultiNodeFromRuntime(t, runtime, false)
	l2ChainID := runtime.L2Network.ChainID()
	nodeC := runtime.Nodes["c"]
	t.Require().NotNil(nodeC, "missing single-chain node c")

	l2ELC := newL2ELFrontend(
		t,
		"c",
		l2ChainID,
		nodeC.EL.UserRPC(),
		nodeC.EL.EngineRPC(),
		nodeC.EL.JWTPath(),
		runtime.L2Network.RollupConfig(),
		nodeC.EL,
	)
	l2CLC := newL2CLFrontend(
		t,
		"c",
		l2ChainID,
		nodeC.CL.UserRPC(),
		nodeC.CL,
	)
	l2CLC.attachEL(l2ELC)
	l2Net, ok := base.L2Chain.Escape().(*presetL2Network)
	t.Require().True(ok, "expected preset L2 network")
	l2Net.AddL2ELNode(l2ELC)
	l2Net.AddL2CLNode(l2CLC)
	testSequencer := newTestSequencerFrontend(
		t,
		runtime.TestSequencer.Name,
		runtime.TestSequencer.AdminRPC,
		runtime.TestSequencer.ControlRPC,
		runtime.TestSequencer.JWTSecret,
	)
	preset := &SingleChainTwoVerifiers{
		SingleChainMultiNode: *base,
		L2ELC:                dsl.NewL2ELNode(l2ELC),
		L2CLC:                dsl.NewL2CLNode(l2CLC),
		TestSequencer:        dsl.NewTestSequencer(testSequencer),
	}
	preset.L2CLC.ManagePeer(preset.L2CL)
	preset.L2CLC.ManagePeer(preset.L2CLB)
	return preset
}

func simpleWithSyncTesterFromRuntime(t devtest.T, runtime *sysgo.SingleChainRuntime) *SimpleWithSyncTester {
	minimal := minimalFromRuntime(t, runtime)
	l2ChainID := runtime.L2Network.ChainID()
	t.Require().NotNil(runtime.SyncTester, "missing sync tester support")
	t.Require().NotNil(runtime.SyncTester.Node, "missing sync tester node")

	syncTesterName, syncTesterRPC, ok := runtime.SyncTester.Service.DefaultEndpoint(runtime.L2Network.ChainID())
	t.Require().Truef(ok, "missing sync tester for chain %s", runtime.L2Network.ChainID())
	syncTester := newSyncTesterFrontend(t, syncTesterName, l2ChainID, syncTesterRPC)

	syncTesterL2EL := newL2ELFrontend(
		t,
		"sync-tester-el",
		l2ChainID,
		runtime.SyncTester.Node.EL.UserRPC(),
		runtime.SyncTester.Node.EL.EngineRPC(),
		runtime.SyncTester.Node.EL.JWTPath(),
		runtime.L2Network.RollupConfig(),
	)
	l2CL2 := newL2CLFrontend(
		t,
		"verifier",
		l2ChainID,
		runtime.SyncTester.Node.CL.UserRPC(),
		runtime.SyncTester.Node.CL,
	)
	l2CL2.attachEL(syncTesterL2EL)
	l2Net, ok := minimal.L2Chain.Escape().(*presetL2Network)
	t.Require().True(ok, "expected preset L2 network")
	l2Net.AddSyncTester(syncTester)
	l2Net.AddL2ELNode(syncTesterL2EL)
	l2Net.AddL2CLNode(l2CL2)

	preset := &SimpleWithSyncTester{
		Minimal:        *minimal,
		SyncTester:     dsl.NewSyncTester(syncTester),
		SyncTesterL2EL: dsl.NewL2ELNode(syncTesterL2EL),
		L2CL2:          dsl.NewL2CLNode(l2CL2),
	}
	preset.L2CL2.ManagePeer(preset.L2CL)
	return preset
}

func minimalWithConductorsFromRuntime(t devtest.T, runtime *sysgo.SingleChainRuntime) *MinimalWithConductors {
	minimal := minimalFromRuntime(t, runtime)
	l2ChainID := runtime.L2Network.ChainID()
	t.Require().NotNil(runtime.Conductors, "missing conductor support")

	cAName := "sequencer"
	cBName := "b"
	cCName := "c"
	cA := newConductorFrontend(t, cAName, l2ChainID, runtime.Conductors[cAName].HTTPEndpoint())
	cB := newConductorFrontend(t, cBName, l2ChainID, runtime.Conductors[cBName].HTTPEndpoint())
	cC := newConductorFrontend(t, cCName, l2ChainID, runtime.Conductors[cCName].HTTPEndpoint())
	l2Net, ok := minimal.L2Chain.Escape().(*presetL2Network)
	t.Require().True(ok, "expected preset L2 network")
	l2Net.AddConductor(cA)
	l2Net.AddConductor(cB)
	l2Net.AddConductor(cC)

	conductors := []stack.Conductor{cA, cB, cC}
	return &MinimalWithConductors{
		Minimal: minimal,
		ConductorSets: map[eth.ChainID]dsl.ConductorSet{
			l2ChainID: dsl.NewConductorSet(conductors),
		},
	}
}
