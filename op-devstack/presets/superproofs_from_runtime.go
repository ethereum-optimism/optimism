package presets

import (
	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
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

func newL1ProposerEOA(t devtest.T, runtime *sysgo.MultiChainRuntime, l2ChainID eth.ChainID, l1EL *dsl.L1ELNode) *dsl.EOA {
	privateKey, err := runtime.Keys.Secret(devkeys.ProposerRole.Key(l2ChainID.ToBig()))
	t.Require().NoError(err, "failed to derive L1 proposer role key")
	return dsl.NewEOA(dsl.NewKey(t, privateKey), l1EL)
}

func simpleInteropFromSupernodeProofsRuntime(t devtest.T, runtime *sysgo.MultiChainRuntime) *SimpleInterop {
	chainA := runtime.Chains["l2a"]
	chainB := runtime.Chains["l2b"]
	t.Require().NotNil(chainA, "missing l2a superproofs chain")
	t.Require().NotNil(chainB, "missing l2b superproofs chain")
	twoL2, components := twoL2FromRuntime(t, runtime)

	supernodeFrontend := newSupernodeFrontend(t, "supernode-two-l2-system", runtime.Supernode.UserRPC(), runtime.Supernode)
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
			timeTravel:       runtime.TimeTravel,
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
			challengerConfig: runtime.L2ChallengerConfig,
		},
		L2ChainB:   twoL2.L2B,
		L2BatcherB: dsl.NewL2Batcher(components.l2BBatcher),
		L2ELB:      dsl.NewL2ELNode(components.l2BEL),
		L2CLB:      twoL2.L2BCL,
	}
	out.l1Proposer = newL1ProposerEOA(t, runtime, chainA.Network.ChainID(), out.L1EL)
	out.FunderL1 = newFunderEOA(t, runtime.Keys, out.L1EL, out.Wallet)
	out.FunderA = newFunderEOA(t, runtime.Keys, out.L2ELA, out.Wallet)
	out.FunderB = newFunderEOA(t, runtime.Keys, out.L2ELB, out.Wallet)

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

	supernodeFrontend := newSupernodeFrontend(t, "supernode-single-system-proofs", runtime.Supernode.UserRPC(), runtime.Supernode)
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

	out := &SingleChainInterop{
		Log:              t.Logger(),
		T:                t,
		timeTravel:       runtime.TimeTravel,
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
		challengerConfig: challengerCfg,
	}
	out.l1Proposer = newL1ProposerEOA(t, runtime, l2ChainID, out.L1EL)
	out.FunderL1 = newFunderEOA(t, runtime.Keys, out.L1EL, out.Wallet)
	out.FunderA = newFunderEOA(t, runtime.Keys, out.L2ELA, out.Wallet)
	return out
}
