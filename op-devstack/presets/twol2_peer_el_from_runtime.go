package presets

import (
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
)

func twoL2SupernodeInteropPeerELFromRuntime(t devtest.T, runtime *sysgo.MultiChainRuntime) *TwoL2SupernodeInteropPeerEL {
	base := twoL2SupernodeInteropFromRuntime(t, runtime)
	seqA := addSequencerFrontend(t, base.L2A, runtime, "l2a")
	seqB := addSequencerFrontend(t, base.L2B, runtime, "l2b")

	// Sequencer↔verifier peering is wired (and registered for restart replay)
	// in sysgo's addPeerELSequencer. Tell the Supernode DSL which verifier
	// ELs need a wipe alongside the supernode data dir.
	base.Supernode.AttachWipeableELs([]*dsl.L2ELNode{base.L2ELA, base.L2ELB})

	return &TwoL2SupernodeInteropPeerEL{
		TwoL2SupernodeInterop: *base,
		SequencerL2AEL:        seqA.el,
		SequencerL2ACL:        seqA.cl,
		SequencerL2BEL:        seqB.el,
		SequencerL2BCL:        seqB.cl,
	}
}

type sequencerFrontend struct {
	el *dsl.L2ELNode
	cl *dsl.L2CLNode
}

func addSequencerFrontend(t devtest.T, l2Net *dsl.L2Network, runtime *sysgo.MultiChainRuntime, chainKey string) sequencerFrontend {
	chain := runtime.Chains[chainKey]
	t.Require().NotNil(chain, "missing %s runtime chain", chainKey)
	seq := chain.Followers["sequencer"]
	t.Require().NotNil(seq, "missing %s sequencer", chainKey)

	chainID := chain.Network.ChainID()
	el := newL2ELFrontend(t, seq.Name, chainID, seq.EL.UserRPC(), seq.EL.EngineRPC(), seq.EL.JWTPath(), chain.Network.RollupConfig(), seq.EL)
	cl := newL2CLFrontend(t, seq.Name, chainID, seq.CL.UserRPC(), seq.CL)
	cl.attachEL(el)

	net, ok := l2Net.Escape().(*presetL2Network)
	t.Require().True(ok, "expected preset L2 network")
	net.AddL2ELNode(el)
	net.AddL2CLNode(cl)

	return sequencerFrontend{el: dsl.NewL2ELNode(el), cl: dsl.NewL2CLNode(cl)}
}
