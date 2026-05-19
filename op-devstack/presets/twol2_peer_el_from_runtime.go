package presets

import (
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
)

func twoL2SupernodeInteropPeerELFromRuntime(t devtest.T, runtime *sysgo.MultiChainRuntime) *TwoL2SupernodeInteropPeerEL {
	base := twoL2SupernodeInteropFromRuntime(t, runtime)
	seqA := addSiblingSequencerFrontend(t, base.L2A, runtime, "l2a")
	seqB := addSiblingSequencerFrontend(t, base.L2B, runtime, "l2b")

	// Sibling↔supernode-fronted peering is wired (and registered for restart
	// replay) in sysgo's addPeerELSiblingSequencer. Tell the Supernode DSL
	// which ELs need a wipe alongside the supernode data dir.
	base.Supernode.AttachWipeableELs([]*dsl.L2ELNode{base.L2ELA, base.L2ELB})

	return &TwoL2SupernodeInteropPeerEL{
		TwoL2SupernodeInterop: *base,
		SequencerL2AEL:        seqA.el,
		SequencerL2ACL:        seqA.cl,
		SequencerL2BEL:        seqB.el,
		SequencerL2BCL:        seqB.cl,
	}
}

type siblingSequencerFrontend struct {
	el *dsl.L2ELNode
	cl *dsl.L2CLNode
}

func addSiblingSequencerFrontend(t devtest.T, l2Net *dsl.L2Network, runtime *sysgo.MultiChainRuntime, chainKey string) siblingSequencerFrontend {
	chain := runtime.Chains[chainKey]
	t.Require().NotNil(chain, "missing %s runtime chain", chainKey)
	sibling := chain.Followers["sequencer"]
	t.Require().NotNil(sibling, "missing %s sibling sequencer", chainKey)

	chainID := chain.Network.ChainID()
	el := newL2ELFrontend(t, sibling.Name, chainID, sibling.EL.UserRPC(), sibling.EL.EngineRPC(), sibling.EL.JWTPath(), chain.Network.RollupConfig(), sibling.EL)
	cl := newL2CLFrontend(t, sibling.Name, chainID, sibling.CL.UserRPC(), sibling.CL)
	cl.attachEL(el)

	net, ok := l2Net.Escape().(*presetL2Network)
	t.Require().True(ok, "expected preset L2 network")
	net.AddL2ELNode(el)
	net.AddL2CLNode(cl)

	return siblingSequencerFrontend{el: dsl.NewL2ELNode(el), cl: dsl.NewL2CLNode(cl)}
}
