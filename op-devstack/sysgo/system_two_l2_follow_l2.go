package sysgo

import (
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// DefaultTwoL2SupernodeFollowL2SystemIDs defines a two-L2 interop+supernode setup
// with one additional follow-source verifier per chain.
type DefaultTwoL2SupernodeFollowL2SystemIDs struct {
	DefaultTwoL2SystemIDs

	L2AFollowerCL stack.L2CLNodeID
	L2AFollowerEL stack.L2ELNodeID
	L2BFollowerCL stack.L2CLNodeID
	L2BFollowerEL stack.L2ELNodeID
}

func NewDefaultTwoL2SupernodeFollowL2SystemIDs(l1ID, l2AID, l2BID eth.ChainID) DefaultTwoL2SupernodeFollowL2SystemIDs {
	return DefaultTwoL2SupernodeFollowL2SystemIDs{
		DefaultTwoL2SystemIDs: NewDefaultTwoL2SystemIDs(l1ID, l2AID, l2BID),
		L2AFollowerCL:         stack.NewL2CLNodeID("follower", l2AID),
		L2AFollowerEL:         stack.NewL2ELNodeID("follower", l2AID),
		L2BFollowerCL:         stack.NewL2CLNodeID("follower", l2BID),
		L2BFollowerEL:         stack.NewL2ELNodeID("follower", l2BID),
	}
}

// DefaultTwoL2SupernodeFollowL2System runs two L2 chains with:
//   - shared supernode CL (interop enabled with configurable delay),
//   - one follow-source verifier per chain in op-node light-CL mode.
//
// The follower for each chain tracks that chain's supernode CL proxy.
func DefaultTwoL2SupernodeFollowL2System(dest *DefaultTwoL2SupernodeFollowL2SystemIDs, delaySeconds uint64) stack.Option[*Orchestrator] {
	ids := NewDefaultTwoL2SupernodeFollowL2SystemIDs(DefaultL1ID, DefaultL2AID, DefaultL2BID)

	var baseIDs DefaultTwoL2SystemIDs
	opt := stack.Combine[*Orchestrator]()

	// Build on top of the existing interop+supernode two-L2 topology.
	opt.Add(DefaultSupernodeInteropTwoL2System(&baseIDs, delaySeconds))

	// Chain A follower
	opt.Add(WithL2ELNode(ids.L2AFollowerEL))
	opt.Add(WithOpNodeFollowL2(ids.L2AFollowerCL, ids.L1CL, ids.L1EL, ids.L2AFollowerEL, ids.L2ACL))
	// TODO(#19379): The chain source is a supernode proxy CL, which does not implement opp2p_* RPCs.
	// Skip CL P2P wiring and rely on follow-source + EL P2P for data availability.
	// opt.Add(WithL2CLP2PConnection(ids.L2ACL, ids.L2AFollowerCL))
	opt.Add(WithL2ELP2PConnection(ids.L2AEL, ids.L2AFollowerEL, false))

	// Chain B follower
	opt.Add(WithL2ELNode(ids.L2BFollowerEL))
	opt.Add(WithOpNodeFollowL2(ids.L2BFollowerCL, ids.L1CL, ids.L1EL, ids.L2BFollowerEL, ids.L2BCL))
	opt.Add(WithL2ELP2PConnection(ids.L2BEL, ids.L2BFollowerEL, false))

	opt.Add(stack.Finally(func(orch *Orchestrator) {
		ids.DefaultTwoL2SystemIDs = baseIDs
		*dest = ids
	}))

	return opt
}
