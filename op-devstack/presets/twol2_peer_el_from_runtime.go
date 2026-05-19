package presets

import (
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
)

func twoL2SupernodeInteropPeerELFromRuntime(t devtest.T, runtime *sysgo.MultiChainRuntime) *TwoL2SupernodeInteropPeerEL {
	base := twoL2SupernodeInteropFromRuntime(t, runtime)

	t.Require().NotNil(runtime.VerifierSupernode, "verifier supernode missing from runtime")
	chainA := runtime.Chains["l2a"]
	chainB := runtime.Chains["l2b"]
	t.Require().NotNil(chainA.VerifierEL, "verifier EL missing on l2a")
	t.Require().NotNil(chainB.VerifierEL, "verifier EL missing on l2b")

	verifierA := addVerifierNode(t, base.L2A, chainA)
	verifierB := addVerifierNode(t, base.L2B, chainB)

	verifierSupernode := newSupernodeFrontend(t, "supernode-two-l2-verifier", runtime.VerifierSupernode.UserRPC())
	verifier := dsl.NewSupernodeWithTestControl(verifierSupernode, runtime.VerifierSupernode)
	verifier.AttachELs([]*dsl.L2ELNode{verifierA.el, verifierB.el})

	return &TwoL2SupernodeInteropPeerEL{
		TwoL2SupernodeInterop: *base,
		VerifierSupernode:     verifier,
		VerifierL2ELA:         verifierA.el,
		VerifierL2ELB:         verifierB.el,
		VerifierL2ACL:         verifierA.cl,
		VerifierL2BCL:         verifierB.cl,
	}
}

type verifierNode struct {
	el *dsl.L2ELNode
	cl *dsl.L2CLNode
}

func addVerifierNode(t devtest.T, l2Net *dsl.L2Network, chain *sysgo.MultiChainNodeRuntime) verifierNode {
	chainID := chain.Network.ChainID()
	el := newL2ELFrontend(t, "verifier", chainID, chain.VerifierEL.UserRPC(), chain.VerifierEL.EngineRPC(), chain.VerifierEL.JWTPath(), chain.Network.RollupConfig(), chain.VerifierEL)
	cl := newL2CLFrontend(t, "verifier", chainID, chain.VerifierCL.UserRPC(), chain.VerifierCL)
	cl.attachEL(el)

	net, ok := l2Net.Escape().(*presetL2Network)
	t.Require().True(ok, "expected preset L2 network")
	net.AddL2ELNode(el)
	net.AddL2CLNode(cl)

	return verifierNode{el: dsl.NewL2ELNode(el), cl: dsl.NewL2CLNode(cl)}
}
