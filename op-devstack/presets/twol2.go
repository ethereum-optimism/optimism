package presets

import (
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/shim"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-devstack/stack/match"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
)

// TwoL2 represents a two-L2 setup without interop considerations.
// It is useful for testing components which bridge multiple L2s without necessarily using interop.
type TwoL2 struct {
	Log          log.Logger
	T            devtest.T
	ControlPlane stack.ControlPlane

	L1Network *dsl.L1Network
	L1EL      *dsl.L1ELNode

	L2A   *dsl.L2Network
	L2B   *dsl.L2Network
	L2ACL *dsl.L2CLNode
	L2BCL *dsl.L2CLNode
}

func WithTwoL2() stack.CommonOption {
	return stack.MakeCommon(sysgo.DefaultTwoL2System(&sysgo.DefaultTwoL2SystemIDs{}))
}

func WithTwoL2Supernode() stack.CommonOption {
	return stack.MakeCommon(sysgo.DefaultSupernodeTwoL2System(&sysgo.DefaultTwoL2SystemIDs{}))
}

func NewTwoL2(t devtest.T) *TwoL2 {
	system := shim.NewSystem(t)
	orch := Orchestrator()
	orch.Hydrate(system)

	l1Net := system.L1Network(match.FirstL1Network)
	l2a := system.L2Network(match.Assume(t, match.L2ChainA))
	l2b := system.L2Network(match.Assume(t, match.L2ChainB))
	l2aCL := l2a.L2CLNode(match.Assume(t, match.WithSequencerActive(t.Ctx())))
	l2bCL := l2b.L2CLNode(match.Assume(t, match.WithSequencerActive(t.Ctx())))

	require.NotEqual(t, l2a.ChainID(), l2b.ChainID())

	return &TwoL2{
		Log:          t.Logger(),
		T:            t,
		ControlPlane: orch.ControlPlane(),
		L1Network:    dsl.NewL1Network(l1Net),
		L1EL:         dsl.NewL1ELNode(l1Net.L1ELNode(match.Assume(t, match.FirstL1EL))),
		L2A:          dsl.NewL2Network(l2a, orch.ControlPlane()),
		L2B:          dsl.NewL2Network(l2b, orch.ControlPlane()),
		L2ACL:        dsl.NewL2CLNode(l2aCL, orch.ControlPlane()),
		L2BCL:        dsl.NewL2CLNode(l2bCL, orch.ControlPlane()),
	}
}
