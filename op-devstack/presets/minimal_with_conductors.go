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

// once kurtosis and sysgo supports conductors, we can merge this with minimal
type MinimalWithConductors struct {
	*Minimal

	ConductorSets map[stack.L2NetworkID]dsl.ConductorSet
}

// TODO(#16418): shift this to a different sysgo constructor once the sysgo implementation supports conductors
func WithMinimalWithConductors() stack.CommonOption {
	return stack.Combine(
		stack.MakeCommon(sysgo.DefaultMinimalSystem(&sysgo.DefaultMinimalSystemIDs{})),
		// TODO(#15152): add kurtosis support
		// TODO(#16418) add sysgo support
		WithCompatibleTypes(compat.Persistent),
	)
}

func NewMinimalWithConductors(t devtest.T) *MinimalWithConductors {
	system := shim.NewSystem(t)
	orch := Orchestrator()
	orch.Hydrate(system)
	chains := system.L2Networks()
	conductorSets := make(map[stack.L2NetworkID]dsl.ConductorSet)
	for _, chain := range chains {
		chainMatcher := match.L2ChainById(chain.ID())
		l2 := system.L2Network(match.Assume(t, chainMatcher))

		conductorSets[chain.ID()] = dsl.NewConductorSet(l2.Conductors())
	}
	return &MinimalWithConductors{
		Minimal:       NewMinimal(t),
		ConductorSets: conductorSets,
	}
}
