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

type MinimalExternalEL struct {
	Log          log.Logger
	T            devtest.T
	ControlPlane stack.ControlPlane

	L1Network *dsl.L1Network
	L1EL      *dsl.L1ELNode

	L2Chain      *dsl.L2Network
	L2CL         *dsl.L2CLNode
	L2EL         *dsl.L2ELNode
	L2ELReadOnly *dsl.L2ELNode

	SyncTester *dsl.SyncTester
}

func (m *MinimalExternalEL) L2Networks() []*dsl.L2Network {
	return []*dsl.L2Network{
		m.L2Chain,
	}
}

func WithExternalELWithSuperchainRegistry(networkPreset stack.ExtNetworkConfig) stack.CommonOption {
	return stack.MakeCommon(sysgo.ExternalELSystemWithEndpointAndSuperchainRegistry(&sysgo.DefaultMinimalExternalELSystemIDs{}, networkPreset))
}

func NewMinimalExternalEL(t devtest.T) *MinimalExternalEL {
	orch := Orchestrator()
	system := shim.NewSystem(t)
	orch.Hydrate(system)

	l2 := system.L2Network(match.L2ChainA)
	verifierCL := l2.L2CLNode(match.FirstL2CL)
	syncTester := l2.SyncTester(match.FirstSyncTester)

	sys := &MinimalExternalEL{
		Log:          t.Logger(),
		T:            t,
		ControlPlane: orch.ControlPlane(),
		L1Network:    dsl.NewL1Network(system.L1Network(match.FirstL1Network)),
		L1EL:         dsl.NewL1ELNode(system.L1Network(match.FirstL1Network).L1ELNode(match.FirstL1EL)),
		L2Chain:      dsl.NewL2Network(l2, orch.ControlPlane()),
		L2CL:         dsl.NewL2CLNode(verifierCL, orch.ControlPlane()),
		L2ELReadOnly: dsl.NewL2ELNode(l2.L2ELNode(match.FirstL2EL), orch.ControlPlane()),
		L2EL:         dsl.NewL2ELNode(l2.L2ELNode(match.SecondL2EL), orch.ControlPlane()),
		SyncTester:   dsl.NewSyncTester(syncTester),
	}
	return sys
}
