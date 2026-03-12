package presets

import (
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
)

func singleChainInteropFromRuntime(t devtest.T, runtime *sysgo.MultiChainRuntime) *SingleChainInterop {
	chainA := runtime.Chains["l2a"]
	t.Require().NotNil(chainA, "missing l2a interop chain")
	l1ChainID := runtime.L1Network.ChainID()
	l2ChainID := chainA.Network.ChainID()

	l1Network := newPresetL1Network(t, "l1", runtime.L1Network.ChainConfig())
	l1EL := newL1ELFrontend(t, "l1", l1ChainID, runtime.L1EL.UserRPC())
	l1CL := newL1CLFrontend(t, "l1", l1ChainID, runtime.L1CL.BeaconHTTPAddr(), runtime.L1CL.FakePoS())
	l1Network.AddL1ELNode(l1EL)
	l1Network.AddL1CLNode(l1CL)

	l2Chain := newPresetL2Network(
		t,
		"l2a",
		chainA.Network.ChainConfig(),
		chainA.Network.RollupConfig(),
		chainA.Network.Deployment(),
		newKeyring(runtime.Keys, t.Require()),
		l1Network,
	)

	l2EL := newL2ELFrontend(
		t,
		"sequencer",
		l2ChainID,
		chainA.EL.UserRPC(),
		chainA.EL.EngineRPC(),
		chainA.EL.JWTPath(),
		chainA.Network.RollupConfig(),
		chainA.EL,
	)
	l2CL := newL2CLFrontend(
		t,
		"sequencer",
		l2ChainID,
		chainA.CL.UserRPC(),
		chainA.CL,
	)
	l2CL.attachEL(l2EL)
	l2Batcher := newL2BatcherFrontend(t, "main", l2ChainID, chainA.Batcher.UserRPC())
	l2Chain.AddL2ELNode(l2EL)
	l2Chain.AddL2CLNode(l2CL)
	l2Chain.AddL2Batcher(l2Batcher)

	supervisor := newSupervisorFrontend(t, "1-primary", runtime.PrimarySupervisor.UserRPC(), runtime.PrimarySupervisor)
	testSequencer := newTestSequencerFrontend(
		t,
		runtime.TestSequencer.Name,
		runtime.TestSequencer.AdminRPC,
		runtime.TestSequencer.ControlRPC,
		runtime.TestSequencer.JWTSecret,
	)
	l1ELDSL := dsl.NewL1ELNode(l1EL)
	l1CLDSL := dsl.NewL1CLNode(l1CL)
	l2ELDSL := dsl.NewL2ELNode(l2EL)
	l2CLDSL := dsl.NewL2CLNode(l2CL)

	faucetAFrontend := newFaucetFrontendForChain(t, runtime.FaucetService, l2ChainID)
	faucetL1Frontend := newFaucetFrontendForChain(t, runtime.FaucetService, l1ChainID)
	out := &SingleChainInterop{
		Log:              t.Logger(),
		T:                t,
		timeTravel:       runtime.TimeTravel,
		Supervisor:       dsl.NewSupervisor(supervisor),
		SuperRoots:       nil,
		TestSequencer:    dsl.NewTestSequencer(testSequencer),
		L1Network:        dsl.NewL1Network(l1Network, l1ELDSL, l1CLDSL),
		L1EL:             l1ELDSL,
		L1CL:             l1CLDSL,
		L2ChainA:         dsl.NewL2Network(l2Chain, l2ELDSL, l2CLDSL, l1ELDSL, nil, nil),
		L2BatcherA:       dsl.NewL2Batcher(l2Batcher),
		L2ELA:            l2ELDSL,
		L2CLA:            l2CLDSL,
		Wallet:           dsl.NewRandomHDWallet(t, 30),
		FaucetA:          dsl.NewFaucet(faucetAFrontend),
		FaucetL1:         dsl.NewFaucet(faucetL1Frontend),
		challengerConfig: runtime.L2ChallengerConfig,
	}
	l1Network.AddFaucet(faucetL1Frontend)
	l2Chain.AddFaucet(faucetAFrontend)
	out.FunderL1 = dsl.NewFunder(out.Wallet, out.FaucetL1, out.L1EL)
	out.FunderA = dsl.NewFunder(out.Wallet, out.FaucetA, out.L2ELA)
	return out
}

func simpleInteropFromRuntime(t devtest.T, runtime *sysgo.MultiChainRuntime) *SimpleInterop {
	singleChain := singleChainInteropFromRuntime(t, runtime)
	chainB := runtime.Chains["l2b"]
	t.Require().NotNil(chainB, "missing l2b interop chain")
	l2BChainID := chainB.Network.ChainID()

	l1Network, ok := singleChain.L1Network.Escape().(*presetL1Network)
	t.Require().True(ok, "expected preset L1 network")

	l2B := newPresetL2Network(
		t,
		"l2b",
		chainB.Network.ChainConfig(),
		chainB.Network.RollupConfig(),
		chainB.Network.Deployment(),
		newKeyring(runtime.Keys, t.Require()),
		l1Network,
	)

	l2BEL := newL2ELFrontend(
		t,
		"sequencer",
		l2BChainID,
		chainB.EL.UserRPC(),
		chainB.EL.EngineRPC(),
		chainB.EL.JWTPath(),
		chainB.Network.RollupConfig(),
		chainB.EL,
	)
	l2BCL := newL2CLFrontend(t, "sequencer", l2BChainID, chainB.CL.UserRPC(), chainB.CL)
	l2BCL.attachEL(l2BEL)
	l2BBatcher := newL2BatcherFrontend(t, "main", l2BChainID, chainB.Batcher.UserRPC())
	l2B.AddL2ELNode(l2BEL)
	l2B.AddL2CLNode(l2BCL)
	l2B.AddL2Batcher(l2BBatcher)

	l2BELDSL := dsl.NewL2ELNode(l2BEL)
	l2BCLDSL := dsl.NewL2CLNode(l2BCL)

	faucetBFrontend := newFaucetFrontendForChain(t, runtime.FaucetService, l2BChainID)
	out := &SimpleInterop{
		SingleChainInterop: *singleChain,
		L2ChainB:           dsl.NewL2Network(l2B, l2BELDSL, l2BCLDSL, singleChain.L1EL, nil, nil),
		L2BatcherB:         dsl.NewL2Batcher(l2BBatcher),
		L2ELB:              l2BELDSL,
		L2CLB:              l2BCLDSL,
		FaucetB:            dsl.NewFaucet(faucetBFrontend),
	}
	l2B.AddFaucet(faucetBFrontend)
	out.FunderB = dsl.NewFunder(out.Wallet, out.FaucetB, out.L2ELB)
	return out
}

func multiSupervisorInteropFromRuntime(t devtest.T, runtime *sysgo.MultiChainRuntime) *MultiSupervisorInterop {
	simpleInterop := simpleInteropFromRuntime(t, runtime)
	chainA := runtime.Chains["l2a"]
	chainB := runtime.Chains["l2b"]
	t.Require().NotNil(chainA, "missing l2a interop chain")
	t.Require().NotNil(chainB, "missing l2b interop chain")
	t.Require().NotNil(chainA.Followers, "missing l2a followers")
	t.Require().NotNil(chainB.Followers, "missing l2b followers")
	l2A2 := chainA.Followers["verifier"]
	l2B2 := chainB.Followers["verifier"]
	t.Require().NotNil(l2A2, "missing l2a verifier follower")
	t.Require().NotNil(l2B2, "missing l2b verifier follower")
	l2AChainID := chainA.Network.ChainID()
	l2BChainID := chainB.Network.ChainID()

	l2ELA2 := newL2ELFrontend(
		t,
		"verifier",
		l2AChainID,
		l2A2.EL.UserRPC(),
		l2A2.EL.EngineRPC(),
		l2A2.EL.JWTPath(),
		chainA.Network.RollupConfig(),
		l2A2.EL,
	)
	l2CLA2 := newL2CLFrontend(t, "verifier", l2AChainID, l2A2.CL.UserRPC(), l2A2.CL)
	l2CLA2.attachEL(l2ELA2)

	l2ELB2 := newL2ELFrontend(
		t,
		"verifier",
		l2BChainID,
		l2B2.EL.UserRPC(),
		l2B2.EL.EngineRPC(),
		l2B2.EL.JWTPath(),
		chainB.Network.RollupConfig(),
		l2B2.EL,
	)
	l2CLB2 := newL2CLFrontend(t, "verifier", l2BChainID, l2B2.CL.UserRPC(), l2B2.CL)
	l2CLB2.attachEL(l2ELB2)

	l2ANet, ok := simpleInterop.L2ChainA.Escape().(*presetL2Network)
	t.Require().True(ok, "expected preset L2 network A")
	l2ANet.AddL2ELNode(l2ELA2)
	l2ANet.AddL2CLNode(l2CLA2)
	l2BNet, ok := simpleInterop.L2ChainB.Escape().(*presetL2Network)
	t.Require().True(ok, "expected preset L2 network B")
	l2BNet.AddL2ELNode(l2ELB2)
	l2BNet.AddL2CLNode(l2CLB2)

	supervisorSecondary := newSupervisorFrontend(
		t,
		"2-secondary",
		runtime.SecondarySupervisor.UserRPC(),
		runtime.SecondarySupervisor,
	)

	return &MultiSupervisorInterop{
		SimpleInterop:       *simpleInterop,
		SupervisorSecondary: dsl.NewSupervisor(supervisorSecondary),
		L2ELA2:              dsl.NewL2ELNode(l2ELA2),
		L2CLA2:              dsl.NewL2CLNode(l2CLA2),
		L2ELB2:              dsl.NewL2ELNode(l2ELB2),
		L2CLB2:              dsl.NewL2CLNode(l2CLB2),
	}
}
