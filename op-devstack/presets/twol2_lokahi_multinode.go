package presets

import (
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
)

// LokahiChainNodes are the three nodes of one chain of a [TwoL2LokahiMultiNode] system, which all
// follow lokahi for their safe and finalized labels.
type LokahiChainNodes struct {
	// Sequencer builds the chain's blocks and derives it. The batcher posts its blocks.
	Sequencer   *dsl.L2CLNode
	SequencerEL *dsl.L2ELNode
	// Source derives the chain, and is the only node lokahi reads the chain from. It imports the
	// sequencer's unsafe blocks over p2p.
	Source   *dsl.L2CLNode
	SourceEL *dsl.L2ELNode
	// Follower derives the chain and imports the sequencer's unsafe blocks over p2p.
	Follower   *dsl.L2CLNode
	FollowerEL *dsl.L2ELNode
}

// CLs are the chain's nodes: the sequencer, the source and the follower.
func (n *LokahiChainNodes) CLs() []*dsl.L2CLNode {
	return []*dsl.L2CLNode{n.Sequencer, n.Source, n.Follower}
}

// ELs are the execution layers of the chain's nodes, in the order of [LokahiChainNodes.CLs].
func (n *LokahiChainNodes) ELs() []*dsl.L2ELNode {
	return []*dsl.L2ELNode{n.SequencerEL, n.SourceEL, n.FollowerEL}
}

// TwoL2LokahiMultiNode is two interop chains, each run by a sequencer, a source node lokahi reads
// and a follower, all following one lokahi process. Its test sequencer controls L1 and builds
// blocks on each chain's sequencer.
type TwoL2LokahiMultiNode struct {
	TwoL2Lokahi

	ChainA LokahiChainNodes
	ChainB LokahiChainNodes
}

// NewTwoL2LokahiMultiNode starts a [TwoL2LokahiMultiNode] system with interop active at genesis
// for the current test. It skips the test under op-node.
func NewTwoL2LokahiMultiNode(t devtest.T) *TwoL2LokahiMultiNode {
	skipKonaOnlyPresetOnOpNode(t)
	runtime := sysgo.NewTwoL2LokahiMultiNodeRuntime(t)
	base := twoL2LokahiFromRuntime(t, runtime, 0)
	return &TwoL2LokahiMultiNode{
		TwoL2Lokahi: *base,
		ChainA:      lokahiChainNodes(t, base.L2A, base.L2ACL, base.L2ELA, runtime.Chains["l2a"]),
		ChainB:      lokahiChainNodes(t, base.L2B, base.L2BCL, base.L2ELB, runtime.Chains["l2b"]),
	}
}

// lokahiChainNodes adds the source and follower of chain to network, next to its sequencer, and
// keeps both peered with the sequencer across restarts.
func lokahiChainNodes(
	t devtest.T,
	network *dsl.L2Network,
	sequencer *dsl.L2CLNode,
	sequencerEL *dsl.L2ELNode,
	chain *sysgo.MultiChainNodeRuntime,
) LokahiChainNodes {
	t.Require().NotNil(chain, "missing lokahi chain")
	presetNet, ok := network.Escape().(*presetL2Network)
	t.Require().True(ok, "expected preset L2 network")
	chainID := chain.Network.ChainID()
	node := func(name string) (*dsl.L2CLNode, *dsl.L2ELNode) {
		runtimeNode := chain.Followers[name]
		t.Require().NotNil(runtimeNode, "chain %s has no %s node", chainID, name)
		el := newL2ELFrontend(t, name, chainID, runtimeNode.EL.UserRPC(), runtimeNode.EL.EngineRPC(),
			runtimeNode.EL.JWTPath(), chain.Network.RollupConfig(), runtimeNode.EL)
		cl := newL2CLFrontend(t, name, chainID, runtimeNode.CL.UserRPC(), runtimeNode.CL)
		cl.attachEL(el)
		presetNet.AddL2ELNode(el)
		presetNet.AddL2CLNode(cl)
		return dsl.NewL2CLNode(cl), dsl.NewL2ELNode(el)
	}
	nodes := LokahiChainNodes{Sequencer: sequencer, SequencerEL: sequencerEL}
	nodes.Source, nodes.SourceEL = node(sysgo.LokahiSourceNode)
	nodes.Follower, nodes.FollowerEL = node(sysgo.LokahiFollowerNode)
	nodes.Source.ManagePeer(nodes.Sequencer)
	nodes.Follower.ManagePeer(nodes.Sequencer)
	return nodes
}
