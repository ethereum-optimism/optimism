package presets

import (
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/devnet-sdk/devstack/devtest"
	"github.com/ethereum-optimism/optimism/devnet-sdk/devstack/dsl"
	"github.com/ethereum-optimism/optimism/devnet-sdk/devstack/shim"
	"github.com/ethereum-optimism/optimism/devnet-sdk/devstack/stack"
	"github.com/ethereum-optimism/optimism/devnet-sdk/devstack/stack/match"
	"github.com/ethereum-optimism/optimism/devnet-sdk/devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
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

	L2CLNodeA *dsl.L2CLNode
	L2CLNodeB *dsl.L2CLNode

	Wallet *dsl.HDWallet

	FaucetA *dsl.Faucet
	FaucetB *dsl.Faucet

	FunderA *dsl.Funder
	FunderB *dsl.Funder
}

func NewSimpleInterop(dest *TestSetup[*SimpleInterop]) stack.Option {
	return func(orch stack.Orchestrator) {
		if _, isSysGo := orch.(*sysgo.Orchestrator); isSysGo {
			startInProcessSimpleInterop(orch)
		}
		*dest = func(t devtest.T) *SimpleInterop {
			return hydrateSimpleInterop(t, orch)
		}
	}
}

// startInProcessSimpleInterop starts a new system that meets the simple interop criteria
func startInProcessSimpleInterop(orch stack.Orchestrator) {
	var ids sysgo.DefaultInteropSystemIDs
	opt := sysgo.DefaultInteropSystem(&ids)
	opt(orch)
}

// hydrateSimpleInterop hydrates the test specific view of a shared system and selects the resources required for
// a simple interop system.
func hydrateSimpleInterop(t devtest.T, orch stack.Orchestrator) *SimpleInterop {
	system := shim.NewSystem(t)
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
		L2CLNodeA:    dsl.NewL2CLNode(l2A.L2CLNode(match.Assume(t, match.FirstL2CL))),
		L2CLNodeB:    dsl.NewL2CLNode(l2B.L2CLNode(match.Assume(t, match.FirstL2CL))),
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
