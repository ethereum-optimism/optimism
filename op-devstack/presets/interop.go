package presets

import (
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/shim"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-devstack/stack/match"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/intentbuilder"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
)

type SimpleInterop struct {
	Log          log.Logger
	T            devtest.T
	Supervisor   *dsl.Supervisor
	Sequencer    *dsl.Sequencer
	ControlPlane stack.ControlPlane

	L1Network *dsl.L1Network

	L2ChainA *dsl.L2Network
	L2ChainB *dsl.L2Network

	L2BatcherA *dsl.L2Batcher
	L2BatcherB *dsl.L2Batcher

	L2ELA *dsl.L2ELNode
	L2ELB *dsl.L2ELNode

	L2CLA *dsl.L2CLNode
	L2CLB *dsl.L2CLNode

	Wallet *dsl.HDWallet

	FaucetA *dsl.Faucet
	FaucetB *dsl.Faucet

	FunderA *dsl.Funder
	FunderB *dsl.Funder
}

func (s *SimpleInterop) L2Networks() []*dsl.L2Network {
	return []*dsl.L2Network{
		s.L2ChainA, s.L2ChainB,
	}
}

// startInProcessSimpleInterop starts a new system that meets the simple interop criteria
func startInProcessSimpleInterop() stack.Option[*sysgo.Orchestrator] {
	var ids sysgo.DefaultInteropSystemIDs
	return sysgo.DefaultInteropSystem(&ids)
}

func ConfigureSimpleInterop() stack.CommonOption {
	if globalBackend == SysGo {
		return stack.MakeCommon(startInProcessSimpleInterop())
	}
	return nil
}

func NewSimpleInterop(t devtest.T) *SimpleInterop {
	system := shim.NewSystem(t)
	orch := Orchestrator()
	orch.Hydrate(system)

	t.Gate().GreaterOrEqual(len(system.Supervisors()), 1, "expected at least one supervisor")
	// At this point, any supervisor is acceptable but as the DSL gets fleshed out this should be selecting supervisors
	// that fit with specific networks and nodes. That will likely require expanding the metadata exposed by the system
	// since currently there's no way to tell which nodes are using which supervisor.

	t.Gate().Equal(len(system.Sequencers()), 1, "expected exactly one sequencer")

	l2A := system.L2Network(match.Assume(t, match.L2ChainA))
	l2B := system.L2Network(match.Assume(t, match.L2ChainB))
	out := &SimpleInterop{
		Log:          t.Logger(),
		T:            t,
		Sequencer:    dsl.NewSequencer(system.Sequencer(match.Assume(t, match.FirstSequencer))),
		Supervisor:   dsl.NewSupervisor(system.Supervisor(match.Assume(t, match.FirstSupervisor))),
		ControlPlane: orch.ControlPlane(),
		L1Network:    dsl.NewL1Network(system.L1Network(match.FirstL1Network)),
		L2ChainA:     dsl.NewL2Network(l2A),
		L2ChainB:     dsl.NewL2Network(l2B),
		L2ELA:        dsl.NewL2ELNode(l2A.L2ELNode(match.Assume(t, match.FirstL2EL))),
		L2ELB:        dsl.NewL2ELNode(l2B.L2ELNode(match.Assume(t, match.FirstL2EL))),
		L2CLA:        dsl.NewL2CLNode(l2A.L2CLNode(match.Assume(t, match.FirstL2CL)), orch.ControlPlane()),
		L2CLB:        dsl.NewL2CLNode(l2B.L2CLNode(match.Assume(t, match.FirstL2CL)), orch.ControlPlane()),
		Wallet:       dsl.NewHDWallet(t, devkeys.TestMnemonic, 30),
		FaucetA:      dsl.NewFaucet(l2A.Faucet(match.Assume(t, match.FirstFaucet))),
		FaucetB:      dsl.NewFaucet(l2B.Faucet(match.Assume(t, match.FirstFaucet))),
		L2BatcherA:   dsl.NewL2Batcher(l2A.L2Batcher(match.Assume(t, match.FirstL2Batcher))),
		L2BatcherB:   dsl.NewL2Batcher(l2B.L2Batcher(match.Assume(t, match.FirstL2Batcher))),
	}
	out.FunderA = dsl.NewFunder(out.Wallet, out.FaucetA, out.L2ELA)
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
