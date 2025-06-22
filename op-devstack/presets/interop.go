package presets

import (
	"time"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl/proofs"
	"github.com/ethereum-optimism/optimism/op-devstack/shim"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-devstack/stack/match"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/intentbuilder"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
)

type SingleChainInterop struct {
	Log    log.Logger
	T      devtest.T
	system stack.ExtensibleSystem

	Supervisor    *dsl.Supervisor
	TestSequencer *dsl.TestSequencer
	ControlPlane  stack.ControlPlane

	L1Network *dsl.L1Network
	L1EL      *dsl.L1ELNode

	L2ChainA   *dsl.L2Network
	L2BatcherA *dsl.L2Batcher
	L2ELA      *dsl.L2ELNode
	L2CLA      *dsl.L2CLNode

	Wallet *dsl.HDWallet

	FaucetA  *dsl.Faucet
	FaucetL1 *dsl.Faucet
	FunderL1 *dsl.Funder
	FunderA  *dsl.Funder
}

func NewSingleChainInterop(t devtest.T) *SingleChainInterop {
	system := shim.NewSystem(t)
	orch := Orchestrator()
	orch.Hydrate(system)

	// At this point, any supervisor is acceptable but as the DSL gets fleshed out this should be selecting supervisors
	// that fit with specific networks and nodes. That will likely require expanding the metadata exposed by the system
	// since currently there's no way to tell which nodes are using which supervisor.
	t.Gate().GreaterOrEqual(len(system.Supervisors()), 1, "expected at least one supervisor")

	t.Gate().Equal(len(system.TestSequencers()), 1, "expected exactly one test sequencer")

	l1Net := system.L1Network(match.FirstL1Network)
	l2A := system.L2Network(match.Assume(t, match.L2ChainA))
	out := &SingleChainInterop{
		Log:           t.Logger(),
		T:             t,
		system:        system,
		TestSequencer: dsl.NewTestSequencer(system.TestSequencer(match.Assume(t, match.FirstTestSequencer))),
		Supervisor:    dsl.NewSupervisor(system.Supervisor(match.Assume(t, match.FirstSupervisor)), orch.ControlPlane()),
		ControlPlane:  orch.ControlPlane(),
		L1Network:     dsl.NewL1Network(l1Net),
		L1EL:          dsl.NewL1ELNode(l1Net.L1ELNode(match.Assume(t, match.FirstL1EL))),
		L2ChainA:      dsl.NewL2Network(l2A),
		L2ELA:         dsl.NewL2ELNode(l2A.L2ELNode(match.Assume(t, match.FirstL2EL))),
		L2CLA:         dsl.NewL2CLNode(l2A.L2CLNode(match.Assume(t, match.FirstL2CL)), orch.ControlPlane()),
		Wallet:        dsl.NewHDWallet(t, devkeys.TestMnemonic, 30),
		FaucetA:       dsl.NewFaucet(l2A.Faucet(match.Assume(t, match.FirstFaucet))),
		L2BatcherA:    dsl.NewL2Batcher(l2A.L2Batcher(match.Assume(t, match.FirstL2Batcher))),
	}
	out.FaucetL1 = dsl.NewFaucet(out.L1Network.Escape().Faucet(match.Assume(t, match.FirstFaucet)))
	out.FunderL1 = dsl.NewFunder(out.Wallet, out.FaucetL1, out.L1EL)
	out.FunderA = dsl.NewFunder(out.Wallet, out.FaucetA, out.L2ELA)
	return out
}

func (s *SingleChainInterop) L2Networks() []*dsl.L2Network {
	return []*dsl.L2Network{
		s.L2ChainA,
	}
}

func (s *SingleChainInterop) AdvanceTime(amount time.Duration) {
	ttSys, ok := s.system.(stack.TimeTravelSystem)
	s.T.Require().True(ok, "attempting to advance time on incompatible system")
	ttSys.AdvanceTime(amount)
}

// WithSingleChainInterop specifies a system that meets the SingleChainInterop criteria.
func WithSingleChainInterop() stack.CommonOption {
	return stack.MakeCommon(sysgo.DefaultSingleChainInteropSystem(&sysgo.DefaultSingleChainInteropSystemIDs{}))
}

type SimpleInterop struct {
	SingleChainInterop

	L2ChainB   *dsl.L2Network
	L2BatcherB *dsl.L2Batcher
	L2ELB      *dsl.L2ELNode
	L2CLB      *dsl.L2CLNode

	FaucetB *dsl.Faucet
	FunderB *dsl.Funder
}

func (s *SimpleInterop) L2Networks() []*dsl.L2Network {
	return []*dsl.L2Network{
		s.L2ChainA, s.L2ChainB,
	}
}

func (s *SimpleInterop) DisputeGameFactory() *proofs.DisputeGameFactory {
	return proofs.NewDisputeGameFactory(s.T, s.L1Network, s.L1EL.EthClient(), s.L2ChainA.DisputeGameFactoryProxyAddr(), s.Supervisor)
}

func (s *SingleChainInterop) StandardBridge(l2Chain *dsl.L2Network) *dsl.StandardBridge {
	return dsl.NewStandardBridge(s.T, l2Chain, s.Supervisor, s.L1EL)
}

// WithSimpleInterop specifies a system that meets the SimpleInterop criteria.
func WithSimpleInterop() stack.CommonOption {
	return stack.MakeCommon(sysgo.DefaultInteropSystem(&sysgo.DefaultInteropSystemIDs{}))
}

// WithSuperInterop specifies a super root system that meets the SimpleInterop criteria.
func WithSuperInterop() stack.CommonOption {
	return stack.MakeCommon(sysgo.DefaultInteropProofsSystem(&sysgo.DefaultInteropSystemIDs{}))
}

// WithUnscheduledInterop adds a test-gate to not run the test if the interop upgrade is scheduled.
// If the backend is sysgo, it will disable the interop configuration
func WithUnscheduledInterop() stack.CommonOption {
	return stack.Combine(
		stack.MakeCommon(sysgo.WithDeployerOptions(func(p devtest.P, keys devkeys.Keys, builder intentbuilder.Builder) {
			for _, l2 := range builder.L2s() {
				l2.WithForkAtOffset(rollup.Interop, nil)
			}
		})),
		stack.PostHydrate[stack.Orchestrator](func(sys stack.System) {
			for _, l2Net := range sys.L2Networks() {
				sys.T().Gate().Nil(l2Net.ChainConfig().InteropTime, "L2 (%s) must not have scheduled interop in chain config", l2Net.ID())
				sys.T().Gate().Nil(l2Net.RollupConfig().InteropTime, "L2 (%s) must not have scheduled interop in rollup config", l2Net.ID())
			}
		}),
	)
}

func NewSimpleInterop(t devtest.T) *SimpleInterop {
	singleChain := NewSingleChainInterop(t)
	orch := Orchestrator()
	l2B := singleChain.system.L2Network(match.Assume(t, match.L2ChainB))
	out := &SimpleInterop{
		SingleChainInterop: *singleChain,
		L2ChainB:           dsl.NewL2Network(l2B),
		L2ELB:              dsl.NewL2ELNode(l2B.L2ELNode(match.Assume(t, match.FirstL2EL))),
		L2CLB:              dsl.NewL2CLNode(l2B.L2CLNode(match.Assume(t, match.FirstL2CL)), orch.ControlPlane()),
		FaucetB:            dsl.NewFaucet(l2B.Faucet(match.Assume(t, match.FirstFaucet))),
		L2BatcherB:         dsl.NewL2Batcher(l2B.L2Batcher(match.Assume(t, match.FirstL2Batcher))),
	}
	out.FunderB = dsl.NewFunder(out.Wallet, out.FaucetB, out.L2ELB)
	return out
}

// WithSuggestedInteropActivationOffset suggests a hardfork time offset to use.
// This is applied e.g. to the deployment if running against sysgo.
func WithSuggestedInteropActivationOffset(offset uint64) stack.CommonOption {
	return stack.MakeCommon(sysgo.WithDeployerOptions(
		func(p devtest.P, keys devkeys.Keys, builder intentbuilder.Builder) {
			for _, l2Cfg := range builder.L2s() {
				l2Cfg.WithForkAtOffset(rollup.Interop, &offset)
			}
		},
	))
}

// WithSequencingWindow suggests a sequencing window to use, and checks the maximum sequencing window.
// The sequencing windows are expressed in number of L1 execution-layer blocks till sequencing window expiry.
// This is applied e.g. to the chain configuration setup if running against sysgo.
func WithSequencingWindow(suggestedSequencingWindow uint64, maxSequencingWindow uint64) stack.CommonOption {
	return stack.Combine(
		stack.MakeCommon(sysgo.WithDeployerOptions(
			sysgo.WithSequencingWindow(suggestedSequencingWindow),
		)),
		// We can't configure sysext sequencing window, so we go with whatever is configured.
		// The post-hydrate function will check that the sequencing window is within expected bounds.
		stack.PostHydrate[stack.Orchestrator](func(sys stack.System) {
			for _, l2Net := range sys.L2Networks() {
				cfg := l2Net.RollupConfig()
				l2Net.T().Gate().LessOrEqual(cfg.SeqWindowSize, maxSequencingWindow,
					"sequencing window of chain %s must fit in max sequencing window size", l2Net.ChainID())
			}
		}),
	)
}

// WithInteropNotAtGenesis adds a test-gate that checks
// if the interop hardfork is configured at a non-genesis time.
func WithInteropNotAtGenesis() stack.CommonOption {
	return stack.PostHydrate[stack.Orchestrator](func(sys stack.System) {
		for _, l2Net := range sys.L2Networks() {
			interopTime := l2Net.ChainConfig().InteropTime
			sys.T().Gate().NotNil(interopTime, "must have interop")
			sys.T().Gate().NotZero(*interopTime, "must not be at genesis")
		}
	})
}

func WithL2NetworkCount(count int) stack.CommonOption {
	return stack.PostHydrate[stack.Orchestrator](func(sys stack.System) {
		sys.T().Gate().Lenf(sys.L2Networks(), count, "Must have exactly %v chains", count)
	})
}

type MultiSupervisorInterop struct {
	SimpleInterop

	// Supervisor does not support multinode so need a additional supervisor for verifier nodes
	SupervisorSecondary *dsl.Supervisor

	L2ELA2 *dsl.L2ELNode
	L2CLA2 *dsl.L2CLNode
	L2ELB2 *dsl.L2ELNode
	L2CLB2 *dsl.L2CLNode
}

func WithMultiSupervisorInterop() stack.CommonOption {
	return stack.MakeCommon(sysgo.MultiSupervisorInteropSystem(&sysgo.MultiSupervisorInteropSystemIDs{}))
}

// NewMultiSupervisorInterop initializes below scenario:
// Two supervisor initialized, each managing two L2CLs per chains.
// Primary supervisor manages sequencer L2CLs for chain A, B.
// Secondary supervisor manages verifier L2CLs for chain A, B.
// Each L2CLs per chain is connected via P2P.
func NewMultiSupervisorInterop(t devtest.T) *MultiSupervisorInterop {
	simpleInterop := NewSimpleInterop(t)
	orch := Orchestrator()

	l2A := simpleInterop.system.L2Network(match.Assume(t, match.L2ChainA))
	l2B := simpleInterop.system.L2Network(match.Assume(t, match.L2ChainB))
	out := &MultiSupervisorInterop{
		SimpleInterop:       *simpleInterop,
		SupervisorSecondary: dsl.NewSupervisor(simpleInterop.system.Supervisor(match.Assume(t, match.SecondSupervisor)), orch.ControlPlane()),
		L2ELA2:              dsl.NewL2ELNode(l2A.L2ELNode(match.Assume(t, match.SecondL2EL))),
		L2CLA2:              dsl.NewL2CLNode(l2A.L2CLNode(match.Assume(t, match.SecondL2CL)), orch.ControlPlane()),
		L2ELB2:              dsl.NewL2ELNode(l2B.L2ELNode(match.Assume(t, match.SecondL2EL))),
		L2CLB2:              dsl.NewL2CLNode(l2B.L2CLNode(match.Assume(t, match.SecondL2CL)), orch.ControlPlane()),
	}
	return out
}
