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
	L2CL    *dsl.L2CLNode
	L2EL    *dsl.L2ELNode

	SyncTester *dsl.SyncTester
}

func (m *MinimalExternalEL) L2Networks() []*dsl.L2Network {
	return []*dsl.L2Network{
		m.L2Chain,
	}
}

func WithMinimalExternalELWithSuperchainRegistry(l1CLBeaconRPC, l1ELRPC, l2ELRPC string, l1ChainID eth.ChainID, networkName string, fcu eth.FCUState) stack.CommonOption {
	return stack.MakeCommon(sysgo.DefaultMinimalExternalELSystemWithEndpointAndSuperchainRegistry(&sysgo.DefaultMinimalExternalELSystemIDs{}, l1CLBeaconRPC, l1ELRPC, l2ELRPC, l1ChainID, networkName, fcu))
}

func NewMinimalExternalELWithExternalL1(t devtest.T) *MinimalExternalEL {
	system := shim.NewSystem(t)
	orch := Orchestrator()
	orch.Hydrate(system)

	l2 := system.L2Network(match.Assume(t, match.L2ChainA))
	verifierCL := l2.L2CLNode(match.FirstL2CL)
	syncTester := l2.SyncTester(match.Assume(t, match.FirstSyncTester))

	return &MinimalExternalEL{
		Log:          t.Logger(),
		T:            t,
		ControlPlane: orch.ControlPlane(),
		L1Network:    dsl.NewL1Network(system.L1Network(match.FirstL1Network)),
		L1EL:         dsl.NewL1ELNode(system.L1Network(match.FirstL1Network).L1ELNode(match.FirstL1EL)),
		L2Chain:      dsl.NewL2Network(l2, orch.ControlPlane()),
		L2CL:         dsl.NewL2CLNode(verifierCL, orch.ControlPlane()),
		L2EL:         dsl.NewL2ELNode(l2.L2ELNode(match.FirstL2EL), orch.ControlPlane()),
		SyncTester:   dsl.NewSyncTester(syncTester),
	}
}
