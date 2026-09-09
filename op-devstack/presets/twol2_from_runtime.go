package presets

import (
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
)

type twoL2RuntimeComponents struct {
	l2AEL *l2ELFrontend
	l2BEL *l2ELFrontend

	l2ABatcher *l2BatcherFrontend
	l2BBatcher *l2BatcherFrontend
}

type l2RuntimePresetComponents struct {
	network *dsl.L2Network
	el      *dsl.L2ELNode
	cl      *dsl.L2CLNode
	batcher *dsl.L2Batcher

	elFrontend      *l2ELFrontend
	batcherFrontend *l2BatcherFrontend
}

type multiL2RuntimePresetComponents struct {
	l1Network *dsl.L1Network
	l1EL      *dsl.L1ELNode
	l1CL      *dsl.L1CLNode
	chains    map[string]*l2RuntimePresetComponents
}

func twoL2SupernodeFromRuntime(t devtest.T, runtime *sysgo.MultiChainRuntime) *TwoL2 {
	preset, _ := twoL2FromRuntime(t, runtime)
	return preset
}

func twoL2FromRuntime(t devtest.T, runtime *sysgo.MultiChainRuntime) (*TwoL2, *twoL2RuntimeComponents) {
	components := multiL2FromRuntime(t, runtime, "l2a", "l2b")
	chainA := components.chains["l2a"]
	chainB := components.chains["l2b"]
	preset := &TwoL2{
		Log:       t.Logger(),
		T:         t,
		L1Network: components.l1Network,
		L1EL:      components.l1EL,
		L1CL:      components.l1CL,
		L2A:       chainA.network,
		L2B:       chainB.network,
		L2ACL:     chainA.cl,
		L2BCL:     chainB.cl,
	}
	return preset, &twoL2RuntimeComponents{
		l2AEL:      chainA.elFrontend,
		l2BEL:      chainB.elFrontend,
		l2ABatcher: chainA.batcherFrontend,
		l2BBatcher: chainB.batcherFrontend,
	}
}

func multiL2FromRuntime(
	t devtest.T,
	runtime *sysgo.MultiChainRuntime,
	chainNames ...string,
) *multiL2RuntimePresetComponents {
	t.Require().Len(runtime.Chains, len(chainNames), "runtime must contain exactly the requested L2 chains")

	l1ChainID := runtime.L1Network.ChainID()
	l1Network := newPresetL1Network(t, "l1", runtime.L1Network.ChainConfig())
	l1EL := newL1ELFrontend(t, "l1", l1ChainID, runtime.L1EL.UserRPC())
	l1CL := newL1CLFrontend(t, "l1", l1ChainID, runtime.L1CL.BeaconHTTPAddr(), runtime.L1CL.FakePoS())
	l1Network.AddL1ELNode(l1EL)
	l1Network.AddL1CLNode(l1CL)
	l1ELDSL := dsl.NewL1ELNode(l1EL)
	l1CLDSL := dsl.NewL1CLNode(l1CL)

	chains := make(map[string]*l2RuntimePresetComponents, len(chainNames))
	keyring := newKeyring(runtime.Keys, t.Require())
	for _, name := range chainNames {
		runtimeChain := runtime.Chains[name]
		t.Require().NotNil(runtimeChain, "missing %s runtime chain", name)
		chainID := runtimeChain.Network.ChainID()
		l2Network := newPresetL2Network(
			t,
			name,
			runtimeChain.Network.ChainConfig(),
			runtimeChain.Network.RollupConfig(),
			runtimeChain.Network.Deployment(),
			keyring,
			l1Network,
		)
		l2EL := newL2ELFrontend(
			t,
			"sequencer",
			chainID,
			runtimeChain.EL.UserRPC(),
			runtimeChain.EL.EngineRPC(),
			runtimeChain.EL.JWTPath(),
			runtimeChain.Network.RollupConfig(),
			runtimeChain.EL,
		)
		l2CL := newL2CLFrontend(t, "sequencer", chainID, runtimeChain.CL.UserRPC(), runtimeChain.CL)
		l2CL.attachEL(l2EL)
		l2Batcher := newL2BatcherFrontend(t, "main", chainID, runtimeChain.Batcher.UserRPC())
		l2Network.AddL2ELNode(l2EL)
		l2Network.AddL2CLNode(l2CL)
		l2Network.AddL2Batcher(l2Batcher)

		l2ELDSL := dsl.NewL2ELNode(l2EL)
		l2CLDSL := dsl.NewL2CLNode(l2CL)
		chains[name] = &l2RuntimePresetComponents{
			network:         dsl.NewL2Network(l2Network, l2ELDSL, l2CLDSL, l1ELDSL, nil, nil),
			el:              l2ELDSL,
			cl:              l2CLDSL,
			batcher:         dsl.NewL2Batcher(l2Batcher),
			elFrontend:      l2EL,
			batcherFrontend: l2Batcher,
		}
	}

	return &multiL2RuntimePresetComponents{
		l1Network: dsl.NewL1Network(l1Network, l1ELDSL, l1CLDSL),
		l1EL:      l1ELDSL,
		l1CL:      l1CLDSL,
		chains:    chains,
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

	supernode := newSupernodeFrontend(t, "supernode-two-l2-system", runtime.Supernode.UserRPC(), runtime.Supernode)
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
		Wallet:                dsl.NewRandomHDWallet(t, 30),
		GenesisTime:           genesisTime,
		InteropActivationTime: genesisTime + runtime.DelaySeconds,
		DelaySeconds:          runtime.DelaySeconds,
		InteropFilter:         runtime.InteropFilter,
		timeTravel:            runtime.TimeTravel,
	}
	preset.FunderA = newFunderEOA(t, runtime.Keys, preset.L2ELA, preset.Wallet)
	preset.FunderB = newFunderEOA(t, runtime.Keys, preset.L2ELB, preset.Wallet)
	return preset
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
