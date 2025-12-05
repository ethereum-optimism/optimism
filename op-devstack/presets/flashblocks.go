package presets

import (
	"time"

	challengerConfig "github.com/ethereum-optimism/optimism/op-challenger/config"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl/proofs"
	"github.com/ethereum-optimism/optimism/op-devstack/shim"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-devstack/stack/match"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
)

type SingleChainWithFlashblocks struct {
	*Minimal

	L2OPRBuilder  *dsl.OPRBuilderNode
	L2RollupBoost *dsl.RollupBoostNode
	TestSequencer *dsl.TestSequencer
}

func (m *SingleChainWithFlashblocks) L2Networks() []*dsl.L2Network {
	return []*dsl.L2Network{
		m.L2Chain,
	}
}

func (m *SingleChainWithFlashblocks) StandardBridge() *dsl.StandardBridge {
	return dsl.NewStandardBridge(m.T, m.L2Chain, nil, m.L1EL)
}

func (m *SingleChainWithFlashblocks) DisputeGameFactory() *proofs.DisputeGameFactory {
	return proofs.NewDisputeGameFactory(m.T, m.L1Network, m.L1EL.EthClient(), m.L2Chain.DisputeGameFactoryProxyAddr(), m.L2CL, m.L2EL, nil, m.challengerConfig)
}

func (m *SingleChainWithFlashblocks) AdvanceTime(amount time.Duration) {
	ttSys, ok := m.system.(stack.TimeTravelSystem)
	m.T.Require().True(ok, "attempting to advance time on incompatible system")
	ttSys.AdvanceTime(amount)
}

func WithSingleChainSystemWithFlashblocks() stack.CommonOption {
	return stack.MakeCommon(sysgo.DefaultSingleChainSystemWithFlashblocks(&sysgo.SingleChainSystemWithFlashblocksIDs{}))
}

func NewSingleChainWithFlashblocks(t devtest.T) *SingleChainWithFlashblocks {
	system := shim.NewSystem(t)
	orch := Orchestrator()
	orch.Hydrate(system)
	l1Net := system.L1Network(match.FirstL1Network)
	l2 := system.L2Network(match.Assume(t, match.L2ChainA))
	sequencerCL := l2.L2CLNode(match.Assume(t, match.WithSequencerActive(t.Ctx())))
	sequencerEL := l2.L2ELNode(match.Assume(t, match.EngineFor(sequencerCL)))
	var challengerCfg *challengerConfig.Config
	if len(l2.L2Challengers()) > 0 {
		challengerCfg = l2.L2Challengers()[0].Config()
	}

	out := &SingleChainWithFlashblocks{
		L2OPRBuilder:  dsl.NewOPRBuilderNode(l2.OPRBuilderNode(match.Assume(t, match.FirstOPRBuilderNode)), orch.ControlPlane()),
		L2RollupBoost: dsl.NewRollupBoostNode(l2.RollupBoostNode(match.Assume(t, match.FirstRollupBoostNode)), orch.ControlPlane()),
		Minimal: &Minimal{
			Log:              t.Logger(),
			T:                t,
			ControlPlane:     orch.ControlPlane(),
			system:           system,
			L1Network:        dsl.NewL1Network(system.L1Network(match.FirstL1Network)),
			L1EL:             dsl.NewL1ELNode(l1Net.L1ELNode(match.Assume(t, match.FirstL1EL))),
			L2Chain:          dsl.NewL2Network(l2, orch.ControlPlane()),
			L2Batcher:        dsl.NewL2Batcher(l2.L2Batcher(match.Assume(t, match.FirstL2Batcher))),
			L2EL:             dsl.NewL2ELNode(sequencerEL, orch.ControlPlane()),
			L2CL:             dsl.NewL2CLNode(sequencerCL, orch.ControlPlane()),
			Wallet:           dsl.NewRandomHDWallet(t, 30), // Random for test isolation
			FaucetL2:         dsl.NewFaucet(l2.Faucet(match.Assume(t, match.FirstFaucet))),
			challengerConfig: challengerCfg,
		},
		TestSequencer: dsl.NewTestSequencer(system.TestSequencer(match.Assume(t, match.FirstTestSequencer))),
	}
	out.FaucetL1 = dsl.NewFaucet(out.L1Network.Escape().Faucet(match.Assume(t, match.FirstFaucet)))
	out.FunderL1 = dsl.NewFunder(out.Wallet, out.FaucetL1, out.L1EL)
	out.FunderL2 = dsl.NewFunder(out.Wallet, out.FaucetL2, out.L2EL)
	return out
}
