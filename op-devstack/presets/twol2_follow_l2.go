package presets

import (
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-devstack/stack/match"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
)

// TwoL2SupernodeFollowL2 extends TwoL2SupernodeInterop with one follow-source
// verifier per chain.
type TwoL2SupernodeFollowL2 struct {
	TwoL2SupernodeInterop

	L2AFollowEL *dsl.L2ELNode
	L2AFollowCL *dsl.L2CLNode
	L2BFollowEL *dsl.L2ELNode
	L2BFollowCL *dsl.L2CLNode
}

// WithTwoL2SupernodeFollowL2 specifies a two-L2 system using a shared supernode
// with interop enabled and one follow-source verifier per chain.
// Use delaySeconds=0 for interop at genesis, or a positive value to test the transition from
// normal safety to interop-verified safety.
func WithTwoL2SupernodeFollowL2(delaySeconds uint64) stack.CommonOption {
	return stack.MakeCommon(sysgo.DefaultTwoL2SupernodeFollowL2System(&sysgo.DefaultTwoL2SupernodeFollowL2SystemIDs{}, delaySeconds))
}

// NewTwoL2SupernodeFollowL2 creates a TwoL2SupernodeFollowL2 preset for acceptance tests.
// Use delaySeconds=0 for interop at genesis, or a positive value to test the transition.
// The delaySeconds must match what was passed to WithTwoL2SupernodeFollowL2 in TestMain.
func NewTwoL2SupernodeFollowL2(t devtest.T, delaySeconds uint64) *TwoL2SupernodeFollowL2 {
	base := NewTwoL2SupernodeInterop(t, delaySeconds)

	l2a := base.system.L2Network(match.L2ChainA)
	l2b := base.system.L2Network(match.L2ChainB)

	followerELAID := stack.NewL2ELNodeID("follower", l2a.ID().ChainID())
	followerCLAID := stack.NewL2CLNodeID("follower", l2a.ID().ChainID())
	followerELBID := stack.NewL2ELNodeID("follower", l2b.ID().ChainID())
	followerCLBID := stack.NewL2CLNodeID("follower", l2b.ID().ChainID())

	followerELA := l2a.L2ELNode(match.MatchElemFn[stack.L2ELNode](func(elem stack.L2ELNode) bool {
		return elem.ID() == followerELAID
	}))
	followerCLA := l2a.L2CLNode(match.MatchElemFn[stack.L2CLNode](func(elem stack.L2CLNode) bool {
		return elem.ID() == followerCLAID
	}))

	followerELB := l2b.L2ELNode(match.MatchElemFn[stack.L2ELNode](func(elem stack.L2ELNode) bool {
		return elem.ID() == followerELBID
	}))
	followerCLB := l2b.L2CLNode(match.MatchElemFn[stack.L2CLNode](func(elem stack.L2CLNode) bool {
		return elem.ID() == followerCLBID
	}))

	return &TwoL2SupernodeFollowL2{
		TwoL2SupernodeInterop: *base,
		L2AFollowEL:           dsl.NewL2ELNode(followerELA, base.ControlPlane),
		L2AFollowCL:           dsl.NewL2CLNode(followerCLA, base.ControlPlane),
		L2BFollowEL:           dsl.NewL2ELNode(followerELB, base.ControlPlane),
		L2BFollowCL:           dsl.NewL2CLNode(followerCLB, base.ControlPlane),
	}
}
