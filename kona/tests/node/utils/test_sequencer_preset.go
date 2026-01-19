package node_utils

import (
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/shim"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-devstack/stack/match"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type MinimalWithTestSequencersPreset struct {
	*MixedOpKonaPreset

	TestSequencer dsl.TestSequencer
}

func WithMixedWithTestSequencer(l2Config L2NodeConfig) stack.CommonOption {
	if l2Config.OpSequencerNodesWithGeth == 0 && l2Config.OpSequencerNodesWithReth == 0 {
		l2Config.OpSequencerNodesWithGeth = 1
	}

	return stack.MakeCommon(DefaultMixedWithTestSequencer(&DefaultMinimalWithTestSequencerIds{}, l2Config))
}

func NewMixedOpKonaWithTestSequencer(t devtest.T) *MinimalWithTestSequencersPreset {
	system := shim.NewSystem(t)
	orch := presets.Orchestrator()
	orch.Hydrate(system)

	t.Gate().Equal(len(system.L2Networks()), 1, "expected exactly one L2 network")
	t.Gate().Equal(len(system.L1Networks()), 1, "expected exactly one L1 network")

	TestSequencer :=
		dsl.NewTestSequencer(system.TestSequencer(match.Assume(t, match.FirstTestSequencer)))

	return &MinimalWithTestSequencersPreset{
		MixedOpKonaPreset: NewMixedOpKona(t),
		TestSequencer:     *TestSequencer,
	}
}

type DefaultMinimalWithTestSequencerIds struct {
	DefaultMixedOpKonaSystemIDs DefaultMixedOpKonaSystemIDs
	TestSequencerId             stack.TestSequencerID
}

func NewDefaultMinimalWithTestSequencerIds(l2Config L2NodeConfig) DefaultMinimalWithTestSequencerIds {
	return DefaultMinimalWithTestSequencerIds{
		DefaultMixedOpKonaSystemIDs: NewDefaultMixedOpKonaSystemIDs(eth.ChainIDFromUInt64(DefaultL1ID), eth.ChainIDFromUInt64(DefaultL2ID), L2NodeConfig{
			OpSequencerNodesWithGeth: l2Config.OpSequencerNodesWithGeth,
			OpSequencerNodesWithReth: l2Config.OpSequencerNodesWithReth,
			OpNodesWithGeth:          l2Config.OpNodesWithGeth,
			OpNodesWithReth:          l2Config.OpNodesWithReth,
			KonaNodesWithGeth:        l2Config.KonaNodesWithGeth,
			KonaNodesWithReth:        l2Config.KonaNodesWithReth,
		}),
		TestSequencerId: "test-sequencer",
	}
}

func DefaultMixedWithTestSequencer(dest *DefaultMinimalWithTestSequencerIds, l2Config L2NodeConfig) stack.Option[*sysgo.Orchestrator] {

	opt := DefaultMixedOpKonaSystem(&dest.DefaultMixedOpKonaSystemIDs, L2NodeConfig{
		OpSequencerNodesWithGeth: l2Config.OpSequencerNodesWithGeth,
		OpSequencerNodesWithReth: l2Config.OpSequencerNodesWithReth,
		OpNodesWithGeth:          l2Config.OpNodesWithGeth,
		OpNodesWithReth:          l2Config.OpNodesWithReth,
		KonaNodesWithGeth:        l2Config.KonaNodesWithGeth,
		KonaNodesWithReth:        l2Config.KonaNodesWithReth,
	})

	ids := NewDefaultMinimalWithTestSequencerIds(l2Config)

	L2SequencerCLNodes := ids.DefaultMixedOpKonaSystemIDs.L2CLSequencerNodes()
	L2SequencerELNodes := ids.DefaultMixedOpKonaSystemIDs.L2ELSequencerNodes()

	opt.Add(sysgo.WithTestSequencer(ids.TestSequencerId, ids.DefaultMixedOpKonaSystemIDs.L1CL, L2SequencerCLNodes[0], ids.DefaultMixedOpKonaSystemIDs.L1EL, L2SequencerELNodes[0]))

	opt.Add(stack.Finally(func(orch *sysgo.Orchestrator) {
		*dest = ids
	}))

	return opt
}
