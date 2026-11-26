package presets

import (
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
)

// TwoL2LokahiDifferential is a [TwoL2Lokahi] system with an op-supernode that verifies the same two
// chains from the same L1, so that a test can compare lokahi with op-supernode on identical
// inputs. The supernode sequences nothing: its virtual nodes derive each chain from the batches the
// chain's batcher posts, into execution layers of their own.
type TwoL2LokahiDifferential struct {
	TwoL2Lokahi

	// LokahiRoots answers superroot_atTimestamp and supernode_syncStatus from lokahi.
	LokahiRoots *dsl.Supernode
	// Supernode answers them from op-supernode.
	Supernode *dsl.Supernode

	// SupernodeCLA and SupernodeCLB are the supernode's per-chain routes, and SupernodeELA and
	// SupernodeELB the execution layers its virtual nodes drive.
	SupernodeCLA *dsl.L2CLNode
	SupernodeCLB *dsl.L2CLNode
	SupernodeELA *dsl.L2ELNode
	SupernodeELB *dsl.L2ELNode

	// L2BatcherA and L2BatcherB post each chain's blocks, which both verifiers derive from.
	L2BatcherA *dsl.L2Batcher
	L2BatcherB *dsl.L2Batcher
}

// NewTwoL2LokahiDifferential starts a [TwoL2LokahiDifferential] system with interop active at
// genesis for the current test. It skips the test under op-node.
func NewTwoL2LokahiDifferential(t devtest.T) *TwoL2LokahiDifferential {
	skipKonaOnlyPresetOnOpNode(t)
	runtime := sysgo.NewTwoL2LokahiDifferentialRuntime(t)
	base := twoL2LokahiFromRuntime(t, runtime, 0)
	out := &TwoL2LokahiDifferential{
		TwoL2Lokahi: *base,
		LokahiRoots: dsl.NewSupernode(newSupernodeFrontend(t, "lokahi", runtime.Lokahi.UserRPC(), runtime.Lokahi)),
		Supernode:   dsl.NewSupernode(newSupernodeFrontend(t, "supernode", runtime.Supernode.UserRPC(), runtime.Supernode)),
	}
	out.SupernodeCLA, out.SupernodeELA = supernodeChainNodes(t, base.L2A, runtime.Chains["l2a"])
	out.SupernodeCLB, out.SupernodeELB = supernodeChainNodes(t, base.L2B, runtime.Chains["l2b"])
	out.L2BatcherA = chainBatcher(t, runtime.Chains["l2a"])
	out.L2BatcherB = chainBatcher(t, runtime.Chains["l2b"])
	return out
}

// supernodeChainNodes adds the supernode's route of chain and the execution layer it drives to
// network.
func supernodeChainNodes(t devtest.T, network *dsl.L2Network, chain *sysgo.MultiChainNodeRuntime) (*dsl.L2CLNode, *dsl.L2ELNode) {
	t.Require().NotNil(chain, "missing chain")
	t.Require().NotNil(chain.SupernodeCL, "chain %s has no supernode route", chain.Name)
	t.Require().NotNil(chain.SupernodeEL, "chain %s has no supernode execution layer", chain.Name)
	presetNet, ok := network.Escape().(*presetL2Network)
	t.Require().True(ok, "expected preset L2 network")
	chainID := chain.Network.ChainID()
	el := newL2ELFrontend(t, "supernode", chainID, chain.SupernodeEL.UserRPC(), chain.SupernodeEL.EngineRPC(),
		chain.SupernodeEL.JWTPath(), chain.Network.RollupConfig(), chain.SupernodeEL)
	cl := newL2CLFrontend(t, "supernode", chainID, chain.SupernodeCL.UserRPC(), chain.SupernodeCL)
	cl.attachEL(el)
	presetNet.AddL2ELNode(el)
	presetNet.AddL2CLNode(cl)
	return dsl.NewL2CLNode(cl), dsl.NewL2ELNode(el)
}

// chainBatcher is the batcher of chain.
func chainBatcher(t devtest.T, chain *sysgo.MultiChainNodeRuntime) *dsl.L2Batcher {
	t.Require().NotNil(chain.Batcher, "chain %s has no batcher", chain.Name)
	return dsl.NewL2Batcher(newL2BatcherFrontend(t, "main", chain.Network.ChainID(), chain.Batcher.UserRPC()))
}
