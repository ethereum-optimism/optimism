package presets

import (
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/shim"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-devstack/stack/match"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-service/apis"
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

// WithTwoL2SupernodeInterop specifies a two-L2 system using a shared supernode with interop enabled.
// Use delaySeconds=0 for interop at genesis, or a positive value to test the transition from
// normal safety to interop-verified safety.
func WithTwoL2SupernodeInterop(delaySeconds uint64) stack.CommonOption {
	return stack.MakeCommon(sysgo.DefaultSupernodeInteropTwoL2System(&sysgo.DefaultTwoL2SystemIDs{}, delaySeconds))
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

// TwoL2SupernodeInterop represents a two-L2 setup with a shared supernode that has interop enabled.
// This allows testing of cross-chain message verification at each timestamp.
// Use delaySeconds=0 for interop at genesis, or a positive value to test the transition.
type TwoL2SupernodeInterop struct {
	TwoL2

	// Supernode provides access to the shared supernode for interop operations
	Supernode *dsl.Supernode

	// L2ELA and L2ELB provide access to the EL nodes for transaction submission
	L2ELA *dsl.L2ELNode
	L2ELB *dsl.L2ELNode

	// L2BatcherA and L2BatcherB provide access to the batchers for pausing/resuming
	L2BatcherA *dsl.L2Batcher
	L2BatcherB *dsl.L2Batcher

	// Faucets for funding test accounts
	FaucetA *dsl.Faucet
	FaucetB *dsl.Faucet

	// Wallet for test account management
	Wallet *dsl.HDWallet

	// Funders for creating funded EOAs
	FunderA *dsl.Funder
	FunderB *dsl.Funder

	// GenesisTime is the genesis timestamp of the L2 chains
	GenesisTime uint64

	// InteropActivationTime is the timestamp when interop becomes active
	InteropActivationTime uint64

	// DelaySeconds is the delay from genesis to interop activation
	DelaySeconds uint64

	// system holds the underlying system for advanced operations
	system stack.ExtensibleSystem
}

// AdvanceTime advances the time-travel clock if enabled.
func (s *TwoL2SupernodeInterop) AdvanceTime(amount time.Duration) {
	ttSys, ok := s.system.(stack.TimeTravelSystem)
	s.T.Require().True(ok, "attempting to advance time on incompatible system")
	ttSys.AdvanceTime(amount)
}

// SuperNodeClient returns an API for calling supernode-specific RPC methods
// like superroot_atTimestamp.
func (s *TwoL2SupernodeInterop) SuperNodeClient() apis.SupernodeQueryAPI {
	return s.Supernode.QueryAPI()
}

// NewTwoL2SupernodeInterop creates a TwoL2SupernodeInterop preset for acceptance tests.
// Use delaySeconds=0 for interop at genesis, or a positive value to test the transition.
// The delaySeconds must match what was passed to WithTwoL2SupernodeInterop in TestMain.
func NewTwoL2SupernodeInterop(t devtest.T, delaySeconds uint64) *TwoL2SupernodeInterop {
	system := shim.NewSystem(t)
	orch := Orchestrator()
	orch.Hydrate(system)

	l1Net := system.L1Network(match.FirstL1Network)
	l2a := system.L2Network(match.Assume(t, match.L2ChainA))
	l2b := system.L2Network(match.Assume(t, match.L2ChainB))
	l2aCL := l2a.L2CLNode(match.Assume(t, match.WithSequencerActive(t.Ctx())))
	l2bCL := l2b.L2CLNode(match.Assume(t, match.WithSequencerActive(t.Ctx())))

	require.NotEqual(t, l2a.ChainID(), l2b.ChainID())

	// Get genesis time from the DSL wrapper
	l2aNet := dsl.NewL2Network(l2a, orch.ControlPlane())
	genesisTime := l2aNet.Escape().RollupConfig().Genesis.L2Time

	// Get the supernode and its test control
	stackSupernode := system.Supernode(match.Assume(t, match.FirstSupernode))
	var testControl stack.InteropTestControl
	if sysgoOrch, ok := orch.(*sysgo.Orchestrator); ok {
		testControl = sysgoOrch.InteropTestControl(stackSupernode.ID())
	}

	out := &TwoL2SupernodeInterop{
		TwoL2: TwoL2{
			Log:          t.Logger(),
			T:            t,
			ControlPlane: orch.ControlPlane(),
			L1Network:    dsl.NewL1Network(l1Net),
			L1EL:         dsl.NewL1ELNode(l1Net.L1ELNode(match.Assume(t, match.FirstL1EL))),
			L2A:          l2aNet,
			L2B:          dsl.NewL2Network(l2b, orch.ControlPlane()),
			L2ACL:        dsl.NewL2CLNode(l2aCL, orch.ControlPlane()),
			L2BCL:        dsl.NewL2CLNode(l2bCL, orch.ControlPlane()),
		},
		Supernode:             dsl.NewSupernodeWithTestControl(stackSupernode, testControl),
		L2ELA:                 dsl.NewL2ELNode(l2a.L2ELNode(match.Assume(t, match.FirstL2EL)), orch.ControlPlane()),
		L2ELB:                 dsl.NewL2ELNode(l2b.L2ELNode(match.Assume(t, match.FirstL2EL)), orch.ControlPlane()),
		L2BatcherA:            dsl.NewL2Batcher(l2a.L2Batcher(match.Assume(t, match.FirstL2Batcher))),
		L2BatcherB:            dsl.NewL2Batcher(l2b.L2Batcher(match.Assume(t, match.FirstL2Batcher))),
		FaucetA:               dsl.NewFaucet(l2a.Faucet(match.Assume(t, match.FirstFaucet))),
		FaucetB:               dsl.NewFaucet(l2b.Faucet(match.Assume(t, match.FirstFaucet))),
		Wallet:                dsl.NewRandomHDWallet(t, 30),
		GenesisTime:           genesisTime,
		InteropActivationTime: genesisTime + delaySeconds,
		DelaySeconds:          delaySeconds,
		system:                system,
	}
	out.FunderA = dsl.NewFunder(out.Wallet, out.FaucetA, out.L2ELA)
	out.FunderB = dsl.NewFunder(out.Wallet, out.FaucetB, out.L2ELB)
	return out
}
