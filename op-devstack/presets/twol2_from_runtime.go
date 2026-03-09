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

	supernode := newSupernodeFrontend(t, "supernode-two-l2-system", runtime.Supernode.UserRPC())
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
		L2BatcherA:            dsl.NewL2Batcher(components.l2ABatcher),
		L2BatcherB:            dsl.NewL2Batcher(components.l2BBatcher),
		FaucetA:               components.faucetA,
		FaucetB:               components.faucetB,
		Wallet:                dsl.NewRandomHDWallet(t, 30),
		GenesisTime:           genesisTime,
		InteropActivationTime: genesisTime + runtime.DelaySeconds,
		DelaySeconds:          runtime.DelaySeconds,
		timeTravel:            runtime.TimeTravel,
	}
	preset.FunderA = dsl.NewFunder(preset.Wallet, preset.FaucetA, preset.L2ELA)
	preset.FunderB = dsl.NewFunder(preset.Wallet, preset.FaucetB, preset.L2ELB)
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
