package presets

import (
	"sort"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type twoL2RuntimeComponents struct {
	l2AEL *l2ELFrontend
	l2BEL *l2ELFrontend

	l2ABatcher *l2BatcherFrontend
	l2BBatcher *l2BatcherFrontend

	faucetA *dsl.Faucet
	faucetB *dsl.Faucet
}

func twoL2SupernodeFromRuntime(t devtest.T, runtime *sysgo.MultiChainRuntime) *TwoL2 {
	preset, _ := twoL2FromRuntime(t, runtime)
	return preset
}

func twoL2FromRuntime(t devtest.T, runtime *sysgo.MultiChainRuntime) (*TwoL2, *twoL2RuntimeComponents) {
	chainA := runtime.Chains["l2a"]
	chainB := runtime.Chains["l2b"]
	t.Require().NotNil(chainA, "missing l2a runtime chain")
	t.Require().NotNil(chainB, "missing l2b runtime chain")
	l1ChainID := runtime.L1Network.ChainID()
	l2AChainID := chainA.Network.ChainID()
	l2BChainID := chainB.Network.ChainID()

	l1Network := newPresetL1Network(t, "l1", runtime.L1Network.ChainConfig())
	l1EL := newL1ELFrontend(t, "l1", l1ChainID, runtime.L1EL.UserRPC())
	l1CL := newL1CLFrontend(t, "l1", l1ChainID, runtime.L1CL.BeaconHTTPAddr(), runtime.L1CL.FakePoS())
	l1Network.AddL1ELNode(l1EL)
	l1Network.AddL1CLNode(l1CL)

	l2A := newPresetL2Network(
		t,
		"l2a",
		chainA.Network.ChainConfig(),
		chainA.Network.RollupConfig(),
		chainA.Network.Deployment(),
		newKeyring(runtime.Keys, t.Require()),
		l1Network,
	)
	l2AEL := newL2ELFrontend(t, "sequencer", l2AChainID, chainA.EL.UserRPC(), chainA.EL.EngineRPC(), chainA.EL.JWTPath(), chainA.Network.RollupConfig(), chainA.EL)
	l2ACL := newL2CLFrontend(t, "sequencer", l2AChainID, chainA.CL.UserRPC(), chainA.CL)
	l2ACL.attachEL(l2AEL)
	l2ABatcher := newL2BatcherFrontend(t, "main", l2AChainID, chainA.Batcher.UserRPC())
	l2A.AddL2ELNode(l2AEL)
	l2A.AddL2CLNode(l2ACL)
	l2A.AddL2Batcher(l2ABatcher)

	l2B := newPresetL2Network(
		t,
		"l2b",
		chainB.Network.ChainConfig(),
		chainB.Network.RollupConfig(),
		chainB.Network.Deployment(),
		newKeyring(runtime.Keys, t.Require()),
		l1Network,
	)
	l2BEL := newL2ELFrontend(t, "sequencer", l2BChainID, chainB.EL.UserRPC(), chainB.EL.EngineRPC(), chainB.EL.JWTPath(), chainB.Network.RollupConfig(), chainB.EL)
	l2BCL := newL2CLFrontend(t, "sequencer", l2BChainID, chainB.CL.UserRPC(), chainB.CL)
	l2BCL.attachEL(l2BEL)
	l2BBatcher := newL2BatcherFrontend(t, "main", l2BChainID, chainB.Batcher.UserRPC())
	l2B.AddL2ELNode(l2BEL)
	l2B.AddL2CLNode(l2BCL)
	l2B.AddL2Batcher(l2BBatcher)

	faucetAFrontend := newFaucetFrontendForChain(t, runtime.FaucetService, l2AChainID)
	faucetBFrontend := newFaucetFrontendForChain(t, runtime.FaucetService, l2BChainID)
	l2A.AddFaucet(faucetAFrontend)
	l2B.AddFaucet(faucetBFrontend)
	faucetA := dsl.NewFaucet(faucetAFrontend)
	faucetB := dsl.NewFaucet(faucetBFrontend)

	l1ELDSL := dsl.NewL1ELNode(l1EL)
	l1CLDSL := dsl.NewL1CLNode(l1CL)
	l2AELDSL := dsl.NewL2ELNode(l2AEL)
	l2ACLDSL := dsl.NewL2CLNode(l2ACL)
	l2BELDSL := dsl.NewL2ELNode(l2BEL)
	l2BCLDSL := dsl.NewL2CLNode(l2BCL)

	preset := &TwoL2{
		Log:       t.Logger(),
		T:         t,
		L1Network: dsl.NewL1Network(l1Network, l1ELDSL, l1CLDSL),
		L1EL:      l1ELDSL,
		L1CL:      l1CLDSL,
		L2A:       dsl.NewL2Network(l2A, l2AELDSL, l2ACLDSL, l1ELDSL, nil, nil),
		L2B:       dsl.NewL2Network(l2B, l2BELDSL, l2BCLDSL, l1ELDSL, nil, nil),
		L2ACL:     l2ACLDSL,
		L2BCL:     l2BCLDSL,
	}
	return preset, &twoL2RuntimeComponents{
		l2AEL:      l2AEL,
		l2BEL:      l2BEL,
		l2ABatcher: l2ABatcher,
		l2BBatcher: l2BBatcher,
		faucetA:    faucetA,
		faucetB:    faucetB,
	}
}

func twoL2SupernodeInteropFromRuntime(t devtest.T, runtime *sysgo.MultiChainRuntime) *TwoL2SupernodeInterop {
	twoL2, components := twoL2FromRuntime(t, runtime)
	chainA := runtime.Chains["l2a"]
	chainB := runtime.Chains["l2b"]
	t.Require().NotNil(chainA, "missing l2a supernode chain")
	t.Require().NotNil(chainB, "missing l2b supernode chain")
	t.Require().NotNil(chainA.SupernodeCL, "missing l2a supernode CL")
	t.Require().NotNil(chainB.SupernodeCL, "missing l2b supernode CL")

	supernode := newSupernodeFrontend(t, "supernode-two-l2-system", runtime.Supernode.UserRPC())
	// The supernode VN drives its own EL, distinct from the sequencer's
	// (joined only by L1 + P2P) in light-sequencer presets. In virtual-sequencer
	// presets the supernode VN is itself the sequencer, so SupernodeEL == EL and
	// it reuses the chain's primary EL frontend.
	l2ASupernodeCL := newL2CLFrontend(t, "supernode", chainA.Network.ChainID(), chainA.SupernodeCL.UserRPC(), chainA.SupernodeCL)
	l2ASupernodeEL := components.l2AEL
	if chainA.SupernodeEL != nil && chainA.SupernodeEL != chainA.EL {
		l2ASupernodeEL = newL2ELFrontend(t, "supernode", chainA.Network.ChainID(), chainA.SupernodeEL.UserRPC(), chainA.SupernodeEL.EngineRPC(), chainA.SupernodeEL.JWTPath(), chainA.Network.RollupConfig(), chainA.SupernodeEL)
	}
	l2ASupernodeCL.attachEL(l2ASupernodeEL)
	l2BSupernodeCL := newL2CLFrontend(t, "supernode", chainB.Network.ChainID(), chainB.SupernodeCL.UserRPC(), chainB.SupernodeCL)
	l2BSupernodeEL := components.l2BEL
	if chainB.SupernodeEL != nil && chainB.SupernodeEL != chainB.EL {
		l2BSupernodeEL = newL2ELFrontend(t, "supernode", chainB.Network.ChainID(), chainB.SupernodeEL.UserRPC(), chainB.SupernodeEL.EngineRPC(), chainB.SupernodeEL.JWTPath(), chainB.Network.RollupConfig(), chainB.SupernodeEL)
	}
	l2BSupernodeCL.attachEL(l2BSupernodeEL)
	testSequencer := newTestSequencerFrontend(
		t,
		runtime.TestSequencer.Name,
		runtime.TestSequencer.AdminRPC,
		runtime.TestSequencer.ControlRPC,
		runtime.TestSequencer.JWTSecret,
	)

	genesisTime := twoL2.L2A.Escape().RollupConfig().Genesis.L2Time
	preset := &TwoL2SupernodeInterop{
		TwoL2: TwoL2{
			Log:       twoL2.Log,
			T:         twoL2.T,
			L1Network: twoL2.L1Network,
			L1EL:      twoL2.L1EL,
			L1CL:      twoL2.L1CL,
			L2A:       twoL2.L2A,
			L2B:       twoL2.L2B,
			L2ACL:     twoL2.L2ACL,
			L2BCL:     twoL2.L2BCL,
		},
		Supernode:             dsl.NewSupernodeWithTestControl(supernode, runtime.Supernode),
		TestSequencer:         dsl.NewTestSequencer(testSequencer),
		L2ELA:                 dsl.NewL2ELNode(components.l2AEL),
		L2ELB:                 dsl.NewL2ELNode(components.l2BEL),
		L2ASupernodeCL:        dsl.NewL2CLNode(l2ASupernodeCL),
		L2BSupernodeCL:        dsl.NewL2CLNode(l2BSupernodeCL),
		L2ASupernodeEL:        dsl.NewL2ELNode(l2ASupernodeEL),
		L2BSupernodeEL:        dsl.NewL2ELNode(l2BSupernodeEL),
		L2BatcherA:            dsl.NewL2Batcher(components.l2ABatcher),
		L2BatcherB:            dsl.NewL2Batcher(components.l2BBatcher),
		FaucetA:               components.faucetA,
		FaucetB:               components.faucetB,
		Wallet:                dsl.NewRandomHDWallet(t, 30),
		GenesisTime:           genesisTime,
		InteropActivationTime: genesisTime + runtime.DelaySeconds,
		DelaySeconds:          runtime.DelaySeconds,
		InteropFilter:         runtime.InteropFilter,
		timeTravel:            runtime.TimeTravel,
	}
	preset.FunderA = dsl.NewFunder(preset.Wallet, preset.FaucetA, preset.L2ELA)
	preset.FunderB = dsl.NewFunder(preset.Wallet, preset.FaucetB, preset.L2ELB)
	return preset
}

func twoL2SupernodeInteropWithConductorsFromRuntime(t devtest.T, runtime *sysgo.MultiChainRuntime) *TwoL2SupernodeInteropWithConductors {
	base := twoL2SupernodeInteropFromRuntime(t, runtime)
	chainA := runtime.Chains["l2a"]
	chainB := runtime.Chains["l2b"]
	t.Require().NotNil(chainA, "missing l2a supernode chain")
	t.Require().NotNil(chainB, "missing l2b supernode chain")
	t.Require().NotNil(chainA.Conductors, "missing l2a conductors")
	t.Require().NotNil(chainB.Conductors, "missing l2b conductors")

	conductorSets := map[eth.ChainID]dsl.ConductorSet{
		chainA.Network.ChainID(): addConductorsToL2Network(t, base.L2A, chainA.Network.ChainID(), chainA.Conductors),
		chainB.Network.ChainID(): addConductorsToL2Network(t, base.L2B, chainB.Network.ChainID(), chainB.Conductors),
	}
	supernodeELs := map[eth.ChainID]*dsl.L2ELNode{
		chainA.Network.ChainID(): supernodeELNode(t, chainA),
		chainB.Network.ChainID(): supernodeELNode(t, chainB),
	}
	sequencerELs := map[eth.ChainID]map[string]*dsl.L2ELNode{
		chainA.Network.ChainID(): conductorSequencerELNodes(t, chainA, base.L2ELA),
		chainB.Network.ChainID(): conductorSequencerELNodes(t, chainB, base.L2ELB),
	}
	return &TwoL2SupernodeInteropWithConductors{
		TwoL2SupernodeInterop: base,
		ConductorSets:         conductorSets,
		SupernodeELs:          supernodeELs,
		SequencerELs:          sequencerELs,
	}
}

// l2ELNodeFromRuntime wraps a sysgo EL handle into a DSL L2ELNode frontend so it can
// be queried directly (used to expose the supernode VN ELs and conductor candidate ELs
// that the base preset does not surface).
func l2ELNodeFromRuntime(t devtest.T, name string, net *sysgo.L2Network, el sysgo.L2ELNode) *dsl.L2ELNode {
	return dsl.NewL2ELNode(newL2ELFrontend(
		t,
		name,
		net.ChainID(),
		el.UserRPC(),
		el.EngineRPC(),
		el.JWTPath(),
		net.RollupConfig(),
		el,
	))
}

// supernodeELNode returns the supernode VN EL for the chain as a DSL handle.
func supernodeELNode(t devtest.T, chain *sysgo.MultiChainNodeRuntime) *dsl.L2ELNode {
	t.Require().NotNil(chain.SupernodeEL, "missing supernode EL for chain %s", chain.Name)
	return l2ELNodeFromRuntime(t, "supernode", chain.Network, chain.SupernodeEL)
}

// conductorSequencerELNodes returns every conductor-controlled sequencer EL for the
// chain, keyed by conductor name. The bootstrap leader ("sequencer") drives the chain's
// primary EL, which the base preset already exposes as leaderEL; each candidate runs its
// own EL recorded in the runtime's Followers map.
func conductorSequencerELNodes(t devtest.T, chain *sysgo.MultiChainNodeRuntime, leaderEL *dsl.L2ELNode) map[string]*dsl.L2ELNode {
	out := make(map[string]*dsl.L2ELNode, len(chain.Conductors))
	for name := range chain.Conductors {
		if name == "sequencer" {
			out[name] = leaderEL
			continue
		}
		follower := chain.Followers[name]
		t.Require().NotNil(follower, "missing follower runtime for conductor candidate %s", name)
		t.Require().NotNil(follower.EL, "missing EL for conductor candidate %s", name)
		out[name] = l2ELNodeFromRuntime(t, name, chain.Network, follower.EL)
	}
	return out
}

func addConductorsToL2Network(t devtest.T, l2 *dsl.L2Network, chainID eth.ChainID, conductors map[string]*sysgo.Conductor) dsl.ConductorSet {
	names := make([]string, 0, len(conductors))
	for name := range conductors {
		names = append(names, name)
	}
	sort.Strings(names)

	frontends := make([]stack.Conductor, 0, len(names))
	l2Net, ok := l2.Escape().(*presetL2Network)
	t.Require().True(ok, "expected preset L2 network")
	for _, name := range names {
		conductor := conductors[name]
		t.Require().NotNil(conductor, "missing conductor %s", name)
		frontend := newConductorFrontend(t, name, chainID, conductor.HTTPEndpoint(), conductor.MetricsEndpoint())
		l2Net.AddConductor(frontend)
		frontends = append(frontends, frontend)
	}
	return dsl.NewConductorSet(frontends)
}

func twoL2SupernodeFollowL2FromRuntime(t devtest.T, runtime *sysgo.MultiChainRuntime) *TwoL2SupernodeFollowL2 {
	base := twoL2SupernodeInteropFromRuntime(t, runtime)
	chainA := runtime.Chains["l2a"]
	chainB := runtime.Chains["l2b"]
	t.Require().NotNil(chainA, "missing l2a supernode chain")
	t.Require().NotNil(chainB, "missing l2b supernode chain")
	t.Require().NotNil(chainA.Followers, "missing l2a followers")
	t.Require().NotNil(chainB.Followers, "missing l2b followers")
	followerA := chainA.Followers["follower"]
	followerB := chainB.Followers["follower"]
	t.Require().NotNil(followerA, "missing l2a follower")
	t.Require().NotNil(followerB, "missing l2b follower")

	l2AFollowEL := newL2ELFrontend(
		t,
		followerA.Name,
		chainA.Network.ChainID(),
		followerA.EL.UserRPC(),
		followerA.EL.EngineRPC(),
		followerA.EL.JWTPath(),
		chainA.Network.RollupConfig(),
		followerA.EL,
	)
	l2AFollowCL := newL2CLFrontend(t, followerA.Name, chainA.Network.ChainID(), followerA.CL.UserRPC(), followerA.CL)
	l2AFollowCL.attachEL(l2AFollowEL)

	l2BFollowEL := newL2ELFrontend(
		t,
		followerB.Name,
		chainB.Network.ChainID(),
		followerB.EL.UserRPC(),
		followerB.EL.EngineRPC(),
		followerB.EL.JWTPath(),
		chainB.Network.RollupConfig(),
		followerB.EL,
	)
	l2BFollowCL := newL2CLFrontend(t, followerB.Name, chainB.Network.ChainID(), followerB.CL.UserRPC(), followerB.CL)
	l2BFollowCL.attachEL(l2BFollowEL)

	l2ANet, ok := base.L2A.Escape().(*presetL2Network)
	t.Require().True(ok, "expected preset L2 network A")
	l2ANet.AddL2ELNode(l2AFollowEL)
	l2ANet.AddL2CLNode(l2AFollowCL)

	l2BNet, ok := base.L2B.Escape().(*presetL2Network)
	t.Require().True(ok, "expected preset L2 network B")
	l2BNet.AddL2ELNode(l2BFollowEL)
	l2BNet.AddL2CLNode(l2BFollowCL)

	return &TwoL2SupernodeFollowL2{
		TwoL2SupernodeInterop: *base,
		L2AFollowEL:           dsl.NewL2ELNode(l2AFollowEL),
		L2AFollowCL:           dsl.NewL2CLNode(l2AFollowCL),
		L2BFollowEL:           dsl.NewL2ELNode(l2BFollowEL),
		L2BFollowCL:           dsl.NewL2CLNode(l2BFollowCL),
	}
}
