package presets

import (
	challengerConfig "github.com/ethereum-optimism/optimism/op-challenger/config"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

func attachChallenger(t devtest.T, l2Net *dsl.L2Network, name string, chainID eth.ChainID, cfg *challengerConfig.Config) {
	if cfg == nil {
		return
	}
	net, ok := l2Net.Escape().(*presetL2Network)
	t.Require().True(ok, "expected preset L2 network")
	net.AddL2Challenger(newPresetL2Challenger(t, name, chainID, cfg))
}

func simpleInteropFromSupernodeProofsRuntime(t devtest.T, runtime *sysgo.MultiChainRuntime) *SimpleInterop {
	chainA := runtime.Chains["l2a"]
	chainB := runtime.Chains["l2b"]
	t.Require().NotNil(chainA, "missing l2a superproofs chain")
	t.Require().NotNil(chainB, "missing l2b superproofs chain")
	twoL2, components := twoL2FromRuntime(t, runtime)

	supernodeFrontend := newSupernodeFrontend(t, "supernode-two-l2-system", runtime.Supernode.UserRPC())
	testSequencer := newTestSequencerFrontend(
		t,
		runtime.TestSequencer.Name,
		runtime.TestSequencer.AdminRPC,
		runtime.TestSequencer.ControlRPC,
		runtime.TestSequencer.JWTSecret,
	)

	out := &SimpleInterop{
		SingleChainInterop: SingleChainInterop{
			Log:              t.Logger(),
			T:                t,
			timeTravel:       nil,
			Supervisor:       nil,
			SuperRoots:       dsl.NewSupernodeWithTestControl(supernodeFrontend, runtime.Supernode),
			TestSequencer:    dsl.NewTestSequencer(testSequencer),
			L1Network:        twoL2.L1Network,
			L1EL:             twoL2.L1EL,
			L1CL:             twoL2.L1CL,
			L2ChainA:         twoL2.L2A,
			L2BatcherA:       dsl.NewL2Batcher(components.l2ABatcher),
			L2ELA:            dsl.NewL2ELNode(components.l2AEL),
			L2CLA:            twoL2.L2ACL,
			Wallet:           dsl.NewRandomHDWallet(t, 30),
			FaucetA:          components.faucetA,
			FaucetL1:         dsl.NewFaucet(newFaucetFrontendForChain(t, runtime.FaucetService, runtime.L1Network.ChainID())),
			challengerConfig: runtime.L2ChallengerConfig,
		},
		L2ChainB:   twoL2.L2B,
		L2BatcherB: dsl.NewL2Batcher(components.l2BBatcher),
		L2ELB:      dsl.NewL2ELNode(components.l2BEL),
		L2CLB:      twoL2.L2BCL,
		FaucetB:    components.faucetB,
	}
	out.FunderL1 = dsl.NewFunder(out.Wallet, out.FaucetL1, out.L1EL)
	out.FunderA = dsl.NewFunder(out.Wallet, out.FaucetA, out.L2ELA)
	out.FunderB = dsl.NewFunder(out.Wallet, out.FaucetB, out.L2ELB)
	l1Net, ok := out.L1Network.Escape().(*presetL1Network)
	t.Require().True(ok, "expected preset L1 network")
	l1Net.AddFaucet(out.FaucetL1.Escape().(*faucetFrontend))

	attachChallenger(t, out.L2ChainA, "main", chainA.Network.ChainID(), out.challengerConfig)
	attachChallenger(t, out.L2ChainB, "main", chainB.Network.ChainID(), out.challengerConfig)
	return out
}

func singleChainInteropFromSupernodeProofsRuntime(t devtest.T, runtime *sysgo.MultiChainRuntime) *SingleChainInterop {
	chainA := runtime.Chains["l2a"]
	t.Require().NotNil(chainA, "missing l2a superproofs chain")
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
	l2CL := newL2CLFrontend(t, "sequencer", l2ChainID, chainA.CL.UserRPC(), chainA.CL)
	l2CL.attachEL(l2EL)
	l2Batcher := newL2BatcherFrontend(t, "main", l2ChainID, chainA.Batcher.UserRPC())
	l2Chain.AddL2ELNode(l2EL)
	l2Chain.AddL2CLNode(l2CL)
	l2Chain.AddL2Batcher(l2Batcher)

	challengerCfg := runtime.L2ChallengerConfig
	if challengerCfg != nil {
		l2Chain.AddL2Challenger(newPresetL2Challenger(t, "main", l2ChainID, challengerCfg))
	}

	supernodeFrontend := newSupernodeFrontend(t, "supernode-single-system-proofs", runtime.Supernode.UserRPC())
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
		timeTravel:       nil,
		Supervisor:       nil,
		SuperRoots:       dsl.NewSupernodeWithTestControl(supernodeFrontend, runtime.Supernode),
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
		challengerConfig: challengerCfg,
	}
	l1Network.AddFaucet(faucetL1Frontend)
	l2Chain.AddFaucet(faucetAFrontend)
	out.FunderL1 = dsl.NewFunder(out.Wallet, out.FaucetL1, out.L1EL)
	out.FunderA = dsl.NewFunder(out.Wallet, out.FaucetA, out.L2ELA)
	return out
}
