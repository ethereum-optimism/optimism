package sysgo

import (
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
)

// NewTwoL2LokahiLightRuntime starts two chains with interop active at genesis, each run by three
// kona-nodes with their own op-reth, and one lokahi process verifying both:
//
//   - the source, which derives the chain, records its safe heads and follows lokahi for its safe
//     and finalized labels, and which is the only node lokahi reads;
//   - the sequencer, a light node whose blocks the batcher posts;
//   - the follower, a light node.
//
// A light node derives nothing: it takes its safe and finalized labels from the source's sync
// status, and imports unsafe blocks over p2p. The sequencer gossips its blocks to the source and
// the follower, and the execution layers are all peered, so a light node's execution layer can
// fetch a block it lacks, such as the source's replacement of a denied block. A test sequencer
// controls L1 and builds blocks on each chain's sequencer.
//
// With DEVSTACK_L2CL_KIND=op-reth, each kona-node runs inside its op-reth process.
func NewTwoL2LokahiLightRuntime(t devtest.T) *MultiChainRuntime {
	return newTwoL2LokahiNodesRuntime(t, "test-sequencer-lokahi-light",
		func(startNode lokahiNodeStarter, l2Net *L2Network) lokahiNodes {
			source := startNode(l2Net, LokahiSourceNode, false, "", lokahiSafeDB(t, l2Net))
			sequencer := startNode(l2Net, "sequencer", true, source.CL.UserRPC())
			follower := startNode(l2Net, LokahiFollowerNode, false, source.CL.UserRPC())
			connectL2CLPeers(t, t.Logger(), sequencer.CL, source.CL)
			connectL2CLPeers(t, t.Logger(), sequencer.CL, follower.CL)
			connectL2ELPeers(t, t.Logger(), sequencer.EL.UserRPC(), source.EL.UserRPC())
			connectL2ELPeers(t, t.Logger(), follower.EL.UserRPC(), source.EL.UserRPC())
			connectL2ELPeers(t, t.Logger(), follower.EL.UserRPC(), sequencer.EL.UserRPC())
			return lokahiNodes{sequencer: sequencer, source: source, follower: follower}
		})
}
