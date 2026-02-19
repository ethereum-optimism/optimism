package node_utils

import (
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/shim"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-devstack/stack/match"
)

type MinimalWithConductors struct {
	*MixedOpKonaPreset

	ConductorSets map[stack.L2NetworkID]dsl.ConductorSet
}

func NewMixedOpKonaWithConductors(t devtest.T) *MinimalWithConductors {
	system := shim.NewSystem(t)
	orch := presets.Orchestrator()
	orch.Hydrate(system)
	chains := system.L2Networks()
	conductorSets := make(map[stack.L2NetworkID]dsl.ConductorSet)
	for _, chain := range chains {
		chainMatcher := match.L2ChainById(chain.ID())
		l2 := system.L2Network(match.Assume(t, chainMatcher))

		conductorSets[chain.ID()] = dsl.NewConductorSet(l2.Conductors())
	}
	return &MinimalWithConductors{
		MixedOpKonaPreset: NewMixedOpKona(t),
		ConductorSets:     conductorSets,
	}
}
