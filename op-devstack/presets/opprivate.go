package presets

import (
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
)

// OPPrivateSkeleton is L1 and one OP Private private chain with test accounts funded in genesis:
// a kona-node + op-reth sequencer, a kona-node + op-reth RPC node that follows it, and no
// batcher, proposer or challenger.
type OPPrivateSkeleton struct {
	T devtest.T

	L1EL *dsl.L1ELNode

	L2Chain     *dsl.L2Network
	SequencerEL *dsl.L2ELNode
	SequencerCL *dsl.L2CLNode
	RPCNodeEL   *dsl.L2ELNode
	RPCNodeCL   *dsl.L2CLNode

	Wallet   *dsl.HDWallet
	FunderL2 *dsl.FunderEOA
}

// NewOPPrivateSkeleton starts the OP Private Skeleton devnet.
func NewOPPrivateSkeleton(t devtest.T, opts ...Option) *OPPrivateSkeleton {
	presetCfg, presetOpts := collectSupportedPresetConfig(t, "NewOPPrivateSkeleton", opts, optionKindDeployer|optionKindOpReth)
	out := opPrivateSkeletonFromRuntime(t, sysgo.NewOPPrivateSkeletonRuntime(t, presetCfg))
	presetOpts.applyPreset(out)
	return out
}

func opPrivateSkeletonFromRuntime(t devtest.T, rt *sysgo.OPPrivateSkeletonRuntime) *OPPrivateSkeleton {
	l1ChainID := rt.L1Network.ChainID()
	l2ChainID := rt.L2Network.ChainID()
	rollupCfg := rt.L2Network.RollupConfig()

	l1Network := newPresetL1Network(t, "l1", rt.L1Network.ChainConfig())
	l1EL := newL1ELFrontend(t, "l1", l1ChainID, rt.L1EL.UserRPC())
	l1CL := newL1CLFrontend(t, "l1", l1ChainID, rt.L1CL.BeaconHTTPAddr(), rt.L1CL.FakePoS())
	l1Network.AddL1ELNode(l1EL)
	l1Network.AddL1CLNode(l1CL)

	l2Chain := newPresetL2Network(t, "l2a", rt.L2Network.ChainConfig(), rollupCfg,
		rt.L2Network.Deployment(), newKeyring(rt.Keys, t.Require()), l1Network)
	node := func(name string, n sysgo.OPPrivateNode) (*dsl.L2ELNode, *dsl.L2CLNode) {
		el := newL2ELFrontend(t, name, l2ChainID, n.EL.UserRPC(), n.EL.EngineRPC(), n.EL.JWTPath(), rollupCfg, n.EL)
		cl := newL2CLFrontend(t, name, l2ChainID, n.CL.UserRPC(), n.CL)
		cl.attachEL(el)
		l2Chain.AddL2ELNode(el)
		l2Chain.AddL2CLNode(cl)
		return dsl.NewL2ELNode(el), dsl.NewL2CLNode(cl)
	}
	seqEL, seqCL := node("sequencer", rt.Sequencer)
	rpcEL, rpcCL := node("rpc", rt.RPCNode)
	// A restarted CL reconnects to the other one, so the RPC node keeps getting unsafe blocks.
	seqCL.ManagePeer(rpcCL)

	l1ELDSL := dsl.NewL1ELNode(l1EL)
	out := &OPPrivateSkeleton{
		T:           t,
		L1EL:        l1ELDSL,
		L2Chain:     dsl.NewL2Network(l2Chain, seqEL, seqCL, l1ELDSL, nil, nil),
		SequencerEL: seqEL,
		SequencerCL: seqCL,
		RPCNodeEL:   rpcEL,
		RPCNodeCL:   rpcCL,
		Wallet:      dsl.NewRandomHDWallet(t, 30),
	}
	out.FunderL2 = newFunderEOA(t, rt.Keys, seqEL, out.Wallet)
	return out
}
