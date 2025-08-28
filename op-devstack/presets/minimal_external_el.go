package presets

import (
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/shim"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-devstack/stack/match"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type MinimalExternalEL struct {
	Log          log.Logger
	T            devtest.T
	ControlPlane stack.ControlPlane

	L1Network *dsl.L1Network
	L1EL      *dsl.L1ELNode

	L2Chain *dsl.L2Network
	// L2Batcher *dsl.L2Batcher
	L2CL *dsl.L2CLNode

	SyncTester *dsl.SyncTester

	Wallet *dsl.HDWallet

	// FaucetL1 *dsl.Faucet
	// FunderL1 *dsl.Funder
}

func (m *MinimalExternalEL) L2Networks() []*dsl.L2Network {
	return []*dsl.L2Network{
		m.L2Chain,
	}
}

func (m *MinimalExternalEL) StandardBridge() *dsl.StandardBridge {
	return dsl.NewStandardBridge(m.T, m.L2Chain, nil, m.L1EL)
}

func WithMinimalExternalELWithSuperchainRegistry(endpointRPC string, networkName string, fcus eth.FCUState) stack.CommonOption {
	return stack.MakeCommon(sysgo.DefaultMinimalExternalELSystemWithEndpointAndSuperchainRegistry(&sysgo.DefaultMinimalExternalELSystemIDs{}, endpointRPC, networkName, fcus))
}

func NewMinimalExternalEL(t devtest.T) *MinimalExternalEL {
	system := shim.NewSystem(t)
	orch := Orchestrator()
	orch.Hydrate(system)

	return minimalExternalELFromSystem(t, system, orch)
}

func minimalExternalELFromSystem(t devtest.T, system stack.ExtensibleSystem, orch stack.Orchestrator) *MinimalExternalEL {
	l1Net := system.L1Network(match.FirstL1Network)
	l2 := system.L2Network(match.Assume(t, match.L2ChainA))
	sequencerCL := l2.L2CLNode(match.Assume(t, match.WithSequencerActive(t.Ctx())))
	syncTester := l2.SyncTester(match.Assume(t, match.FirstSyncTester))

	out := &MinimalExternalEL{
		Log:          t.Logger(),
		T:            t,
		ControlPlane: orch.ControlPlane(),
		L1Network:    dsl.NewL1Network(system.L1Network(match.FirstL1Network)),
		L1EL:         dsl.NewL1ELNode(l1Net.L1ELNode(match.Assume(t, match.FirstL1EL))),
		L2Chain:      dsl.NewL2Network(l2, orch.ControlPlane()),
		// L2Batcher:    dsl.NewL2Batcher(l2.L2Batcher(match.Assume(t, match.FirstL2Batcher))),
		L2CL:       dsl.NewL2CLNode(sequencerCL, orch.ControlPlane()),
		SyncTester: dsl.NewSyncTester(syncTester),
		Wallet:     dsl.NewRandomHDWallet(t, 30),
	}
	// out.FaucetL1 = dsl.NewFaucet(out.L1Network.Escape().Faucet(match.Assume(t, match.FirstFaucet)))
	// out.FunderL1 = dsl.NewFunder(out.Wallet, out.FaucetL1, out.L1EL)
	return out
}
