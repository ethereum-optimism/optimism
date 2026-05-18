package presets

import (
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
)

// twoL2SupernodeInteropPeerELFromRuntime adapts the peer-EL sysgo runtime
// into the TwoL2SupernodeInteropPeerEL preset.
//
// Field convention (important for tests):
//   - TwoL2.L2A / L2B, L2ACL / L2BCL, L2ELA / L2ELB are the **supernode-fronted**
//     view per chain — L2ACL is the supernode VN proxy, L2ELA is the
//     supernode-fronted EL (i.e. the cold-start wipe target).
//   - Sibling sequencer access (CL+EL that drive the batcher and keep producing
//     blocks during a wipe) is exposed via the SequencerL2A/B fields on the
//     network DSL through their attached frontends, and used internally for
//     batcher/faucet wiring.
func twoL2SupernodeInteropPeerELFromRuntime(t devtest.T, runtime *sysgo.MultiChainRuntime) *TwoL2SupernodeInteropPeerEL {
	require := t.Require()
	chainA := runtime.Chains["l2a"]
	chainB := runtime.Chains["l2b"]
	require.NotNil(chainA, "missing l2a runtime chain")
	require.NotNil(chainB, "missing l2b runtime chain")
	require.NotNil(chainA.Followers, "missing l2a followers (supernode-fronted)")
	require.NotNil(chainB.Followers, "missing l2b followers (supernode-fronted)")

	supernodeA := chainA.Followers["supernode"]
	supernodeB := chainB.Followers["supernode"]
	require.NotNil(supernodeA, "missing l2a supernode-fronted runtime")
	require.NotNil(supernodeB, "missing l2b supernode-fronted runtime")

	l1ChainID := runtime.L1Network.ChainID()
	l2AChainID := chainA.Network.ChainID()
	l2BChainID := chainB.Network.ChainID()

	l1Network := newPresetL1Network(t, "l1", runtime.L1Network.ChainConfig())
	l1EL := newL1ELFrontend(t, "l1", l1ChainID, runtime.L1EL.UserRPC())
	l1CL := newL1CLFrontend(t, "l1", l1ChainID, runtime.L1CL.BeaconHTTPAddr(), runtime.L1CL.FakePoS())
	l1Network.AddL1ELNode(l1EL)
	l1Network.AddL1CLNode(l1CL)

	// Sibling sequencer frontends (the canonical chain.EL / chain.CL).
	l2A := newPresetL2Network(
		t,
		"l2a",
		chainA.Network.ChainConfig(),
		chainA.Network.RollupConfig(),
		chainA.Network.Deployment(),
		newKeyring(runtime.Keys, t.Require()),
		l1Network,
	)
	seqELA := newL2ELFrontend(t, "sequencer", l2AChainID, chainA.EL.UserRPC(), chainA.EL.EngineRPC(), chainA.EL.JWTPath(), chainA.Network.RollupConfig(), chainA.EL)
	seqCLA := newL2CLFrontend(t, "sequencer", l2AChainID, chainA.CL.UserRPC(), chainA.CL)
	seqCLA.attachEL(seqELA)
	l2ABatcher := newL2BatcherFrontend(t, "main", l2AChainID, chainA.Batcher.UserRPC())
	l2A.AddL2ELNode(seqELA)
	l2A.AddL2CLNode(seqCLA)
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
	seqELB := newL2ELFrontend(t, "sequencer", l2BChainID, chainB.EL.UserRPC(), chainB.EL.EngineRPC(), chainB.EL.JWTPath(), chainB.Network.RollupConfig(), chainB.EL)
	seqCLB := newL2CLFrontend(t, "sequencer", l2BChainID, chainB.CL.UserRPC(), chainB.CL)
	seqCLB.attachEL(seqELB)
	l2BBatcher := newL2BatcherFrontend(t, "main", l2BChainID, chainB.Batcher.UserRPC())
	l2B.AddL2ELNode(seqELB)
	l2B.AddL2CLNode(seqCLB)
	l2B.AddL2Batcher(l2BBatcher)

	// Supernode-fronted frontends — distinct EL + CL proxy per chain.
	supELA := newL2ELFrontend(t, supernodeA.Name, l2AChainID, supernodeA.EL.UserRPC(), supernodeA.EL.EngineRPC(), supernodeA.EL.JWTPath(), chainA.Network.RollupConfig(), supernodeA.EL)
	supCLA := newL2CLFrontend(t, "supernode", l2AChainID, supernodeA.CL.UserRPC(), supernodeA.CL)
	supCLA.attachEL(supELA)
	l2A.AddL2ELNode(supELA)
	l2A.AddL2CLNode(supCLA)

	supELB := newL2ELFrontend(t, supernodeB.Name, l2BChainID, supernodeB.EL.UserRPC(), supernodeB.EL.EngineRPC(), supernodeB.EL.JWTPath(), chainB.Network.RollupConfig(), supernodeB.EL)
	supCLB := newL2CLFrontend(t, "supernode", l2BChainID, supernodeB.CL.UserRPC(), supernodeB.CL)
	supCLB.attachEL(supELB)
	l2B.AddL2ELNode(supELB)
	l2B.AddL2CLNode(supCLB)

	faucetAFrontend := newFaucetFrontendForChain(t, runtime.FaucetService, l2AChainID)
	faucetBFrontend := newFaucetFrontendForChain(t, runtime.FaucetService, l2BChainID)
	l2A.AddFaucet(faucetAFrontend)
	l2B.AddFaucet(faucetBFrontend)
	faucetA := dsl.NewFaucet(faucetAFrontend)
	faucetB := dsl.NewFaucet(faucetBFrontend)

	l1ELDSL := dsl.NewL1ELNode(l1EL)
	l1CLDSL := dsl.NewL1CLNode(l1CL)

	// L2ELA / L2ELB on the embedded TwoL2SupernodeInterop point at the
	// **supernode-fronted** EL, since that is the EL these tests wipe and
	// the natural counterpart to L2ACL / L2BCL (the supernode VN proxy).
	supELADSL := dsl.NewL2ELNode(supELA)
	supELBDSL := dsl.NewL2ELNode(supELB)
	supCLADSL := dsl.NewL2CLNode(supCLA)
	supCLBDSL := dsl.NewL2CLNode(supCLB)

	supernode := newSupernodeFrontend(t, "supernode-two-l2-peer-el", runtime.Supernode.UserRPC())
	testSequencer := newTestSequencerFrontend(
		t,
		runtime.TestSequencer.Name,
		runtime.TestSequencer.AdminRPC,
		runtime.TestSequencer.ControlRPC,
		runtime.TestSequencer.JWTSecret,
	)

	genesisTime := chainA.Network.RollupConfig().Genesis.L2Time
	preset := &TwoL2SupernodeInteropPeerEL{
		TwoL2SupernodeInterop: TwoL2SupernodeInterop{
			TwoL2: TwoL2{
				Log:       t.Logger(),
				T:         t,
				L1Network: dsl.NewL1Network(l1Network, l1ELDSL, l1CLDSL),
				L1EL:      l1ELDSL,
				L1CL:      l1CLDSL,
				L2A:       dsl.NewL2Network(l2A, supELADSL, supCLADSL, l1ELDSL, nil, nil),
				L2B:       dsl.NewL2Network(l2B, supELBDSL, supCLBDSL, l1ELDSL, nil, nil),
				L2ACL:     supCLADSL,
				L2BCL:     supCLBDSL,
			},
			Supernode:             dsl.NewSupernodeWithTestControl(supernode, runtime.Supernode),
			TestSequencer:         dsl.NewTestSequencer(testSequencer),
			L2ELA:                 supELADSL,
			L2ELB:                 supELBDSL,
			L2BatcherA:            dsl.NewL2Batcher(l2ABatcher),
			L2BatcherB:            dsl.NewL2Batcher(l2BBatcher),
			FaucetA:               faucetA,
			FaucetB:               faucetB,
			Wallet:                dsl.NewRandomHDWallet(t, 30),
			GenesisTime:           genesisTime,
			InteropActivationTime: genesisTime + runtime.DelaySeconds,
			DelaySeconds:          runtime.DelaySeconds,
			timeTravel:            runtime.TimeTravel,
		},
		SupernodeL2AEL: supELADSL,
		SupernodeL2BEL: supELBDSL,
	}
	// Funders use the supernode-fronted ELs so that tx submission goes
	// through the same EL the tests wipe — keeps Funder semantics aligned
	// with the rest of the preset.
	preset.FunderA = dsl.NewFunder(preset.Wallet, preset.FaucetA, preset.L2ELA)
	preset.FunderB = dsl.NewFunder(preset.Wallet, preset.FaucetB, preset.L2ELB)
	return preset
}
