package presets

import (
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/shim"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-devstack/stack/match"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
)

type Minimal struct {
	Log          log.Logger
	T            devtest.T
	ControlPlane stack.ControlPlane

	L1Network *dsl.L1Network
	L1EL      *dsl.L1ELNode

	L2Chain   *dsl.L2Network
	L2Batcher *dsl.L2Batcher
	L2EL      *dsl.L2ELNode
	L2CL      *dsl.L2CLNode

	Wallet *dsl.HDWallet

	FaucetL1 *dsl.Faucet
	FaucetL2 *dsl.Faucet
	FunderL1 *dsl.Funder
	FunderL2 *dsl.Funder
}

func (m *Minimal) L2Networks() []*dsl.L2Network {
	return []*dsl.L2Network{
		m.L2Chain,
	}
}

func (m *Minimal) StandardBridge() *dsl.StandardBridge {
	return dsl.NewStandardBridge(m.T, m.L2Chain, nil, m.L1EL)
}

func WithMinimal() stack.CommonOption {
	return stack.MakeCommon(sysgo.DefaultMinimalSystem(&sysgo.DefaultMinimalSystemIDs{}))
}

func NewMinimal(t devtest.T) *Minimal {
	system := shim.NewSystem(t)
	orch := Orchestrator()
	orch.Hydrate(system)

	return minimalFromSystem(t, system, orch)
}

func minimalFromSystem(t devtest.T, system stack.ExtensibleSystem, orch stack.Orchestrator) *Minimal {
	l1Net := system.L1Network(match.FirstL1Network)
	l2 := system.L2Network(match.Assume(t, match.L2ChainA))
	sequencerCL := l2.L2CLNode(match.Assume(t, match.WithSequencerActive(t.Ctx())))
	sequencerEL := l2.L2ELNode(match.Assume(t, match.EngineFor(sequencerCL)))
	out := &Minimal{
		Log:          t.Logger(),
		T:            t,
		ControlPlane: orch.ControlPlane(),
		L1Network:    dsl.NewL1Network(system.L1Network(match.FirstL1Network)),
		L1EL:         dsl.NewL1ELNode(l1Net.L1ELNode(match.Assume(t, match.FirstL1EL))),
		L2Chain:      dsl.NewL2Network(l2, orch.ControlPlane()),
		L2Batcher:    dsl.NewL2Batcher(l2.L2Batcher(match.Assume(t, match.FirstL2Batcher))),
		L2EL:         dsl.NewL2ELNode(sequencerEL, orch.ControlPlane()),
		L2CL:         dsl.NewL2CLNode(sequencerCL, orch.ControlPlane()),
		Wallet:       dsl.NewRandomHDWallet(t, 30), // Random for test isolation
		FaucetL2:     dsl.NewFaucet(l2.Faucet(match.Assume(t, match.FirstFaucet))),
	}
	out.FaucetL1 = dsl.NewFaucet(out.L1Network.Escape().Faucet(match.Assume(t, match.FirstFaucet)))
	out.FunderL1 = dsl.NewFunder(out.Wallet, out.FaucetL1, out.L1EL)
	out.FunderL2 = dsl.NewFunder(out.Wallet, out.FaucetL2, out.L2EL)
	return out
}
