package presets

import (
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type MixedSingleChainNodeFrontends struct {
	Spec sysgo.MixedSingleChainNodeSpec
	EL   *dsl.L2ELNode
	CL   *dsl.L2CLNode
}

type MixedSingleChainFrontends struct {
	L1Network     *dsl.L1Network
	L1EL          *dsl.L1ELNode
	L1CL          *dsl.L1CLNode
	L2Network     *dsl.L2Network
	L2Batcher     *dsl.L2Batcher
	Wallet        *dsl.HDWallet
	FunderL1      *dsl.FunderEOA
	FunderL2      *dsl.FunderEOA
	TestSequencer *dsl.TestSequencer
	Nodes         []MixedSingleChainNodeFrontends
}

func NewMixedSingleChainFrontends(t devtest.T, runtime *sysgo.MixedSingleChainRuntime) *MixedSingleChainFrontends {
	l1Backend := runtime.L1Network
	l2Backend := runtime.L2Network
	l1ChainID := eth.ChainIDFromBig(l1Backend.ChainConfig().ChainID)
	l2ChainID := eth.ChainIDFromBig(l2Backend.ChainConfig().ChainID)

	l1Network := newPresetL1Network(t, "l1", l1Backend.ChainConfig())
	l1EL := newL1ELFrontend(t, "l1", l1ChainID, runtime.L1EL.UserRPC())
	l1CL := newL1CLFrontend(t, "l1", l1ChainID, runtime.L1CL.BeaconHTTPAddr(), runtime.L1CL.FakePoS())
	l1Network.AddL1ELNode(l1EL)
	l1Network.AddL1CLNode(l1CL)

	l2Network := newPresetL2Network(
		t,
		"l2a",
		l2Backend.ChainConfig(),
		l2Backend.RollupConfig(),
		l2Backend.Deployment(),
		newKeyring(l2Backend.Keys(), t.Require()),
		l1Network,
	)
	l2BatcherBackend := runtime.L2Batcher
	l2Batcher := newL2BatcherFrontend(t, "main", l2ChainID, l2BatcherBackend.UserRPC())
	l2Network.AddL2Batcher(l2Batcher)

	l1ELDSL := dsl.NewL1ELNode(l1EL)
	l1CLDSL := dsl.NewL1CLNode(l1CL)

	nodes := make([]MixedSingleChainNodeFrontends, 0, len(runtime.Nodes))
	var primaryL2EL *dsl.L2ELNode
	var primaryL2CL *dsl.L2CLNode
	for _, node := range runtime.Nodes {
		l2EL := newL2ELFrontend(
			t,
			node.Spec.ELKey,
			l2ChainID,
			node.EL.UserRPC(),
			node.EL.EngineRPC(),
			node.EL.JWTPath(),
			l2Backend.RollupConfig(),
			node.EL,
		)
		l2CL := newL2CLFrontend(t, node.Spec.CLKey, l2ChainID, node.CL.UserRPC(), node.CL)
		l2CL.attachEL(l2EL)
		l2Network.AddL2ELNode(l2EL)
		l2Network.AddL2CLNode(l2CL)
		l2ELDSL := dsl.NewL2ELNode(l2EL)
		l2CLDSL := dsl.NewL2CLNode(l2CL)
		if primaryL2EL == nil && node.Spec.IsSequencer {
			primaryL2EL = l2ELDSL
			primaryL2CL = l2CLDSL
		}
		nodes = append(nodes, MixedSingleChainNodeFrontends{
			Spec: node.Spec,
			EL:   l2ELDSL,
			CL:   l2CLDSL,
		})
	}
	t.Require().NotNil(primaryL2EL, "missing primary mixed L2 EL")
	t.Require().NotNil(primaryL2CL, "missing primary mixed L2 CL")

	keys := l2Backend.Keys()
	wallet := dsl.NewRandomHDWallet(t, 30) // Random for test isolation
	funderL1 := newFunderEOA(t, keys, l1ELDSL, wallet)
	funderL2 := newFunderEOA(t, keys, primaryL2EL, wallet)

	var testSequencer *dsl.TestSequencer
	if backend := runtime.TestSequencer; backend != nil {
		t.Require().NotEmpty(backend.Name, "expected test sequencer name")
		testSequencer = dsl.NewTestSequencer(newTestSequencerFrontend(
			t,
			backend.Name,
			backend.AdminRPC,
			backend.ControlRPC,
			backend.JWTSecret,
		))
	}

	return &MixedSingleChainFrontends{
		L1Network:     dsl.NewL1Network(l1Network, l1ELDSL, l1CLDSL),
		L1EL:          l1ELDSL,
		L1CL:          l1CLDSL,
		L2Network:     dsl.NewL2Network(l2Network, primaryL2EL, primaryL2CL, l1ELDSL, nil, nil),
		L2Batcher:     dsl.NewL2Batcher(l2Batcher),
		Wallet:        wallet,
		FunderL1:      funderL1,
		FunderL2:      funderL2,
		TestSequencer: testSequencer,
		Nodes:         nodes,
	}
}
