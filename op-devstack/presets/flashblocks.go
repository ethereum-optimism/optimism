package presets

import (
	"github.com/ethereum-optimism/optimism/op-devstack/compat"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/shim"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-devstack/stack/match"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
)

type SimpleFlashblocks struct {
	*Minimal

	ConductorSets          map[stack.L2NetworkID]dsl.ConductorSet
	FlashblocksBuilderSets map[stack.L2NetworkID]dsl.FlashblocksBuilderSet
}

func WithSimpleFlashblocks() stack.CommonOption {
	return stack.Combine(
		stack.MakeCommon(sysgo.DefaultMinimalSystem(&sysgo.DefaultMinimalSystemIDs{})),
		// TODO(#16450): add sysgo support for flashblocks
		// TODO(#16514): add kurtosis support for flashblocks
		WithCompatibleTypes(compat.Persistent),
	)
}

func NewSimpleFlashblocks(t devtest.T) *SimpleFlashblocks {
	system := shim.NewSystem(t)
	orch := Orchestrator()
	orch.Hydrate(system)
	chains := system.L2Networks()
	conductorSets := make(map[stack.L2NetworkID]dsl.ConductorSet)
	flashblocksBuilderSets := make(map[stack.L2NetworkID]dsl.FlashblocksBuilderSet)
	for _, chain := range chains {
		chainMatcher := match.L2ChainById(chain.ID())
		l2 := system.L2Network(match.Assume(t, chainMatcher))

		conductorSets[chain.ID()] = dsl.NewConductorSet(l2.Conductors())
		flashblocksBuilderSets[chain.ID()] = dsl.NewFlashblocksBuilderSet(l2.FlashblocksBuilders())
	}
	return &SimpleFlashblocks{
		Minimal:                NewMinimal(t),
		ConductorSets:          conductorSets,
		FlashblocksBuilderSets: flashblocksBuilderSets,
	}
}
