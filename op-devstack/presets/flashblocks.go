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

	ConductorSets               map[stack.L2NetworkID]dsl.ConductorSet
	FlashblocksBuilderSets      map[stack.L2NetworkID]dsl.FlashblocksBuilderSet
	FlashblocksWebsocketProxies map[stack.L2NetworkID]dsl.FlashblocksWebsocketProxySet

	Faucets   map[stack.L2NetworkID]*dsl.Faucet
	Funders   map[stack.L2NetworkID]*dsl.Funder
	L2ELNodes map[stack.L2NetworkID]*dsl.L2ELNode
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

	minimalPreset := NewMinimal(t)

	conductorSets := make(map[stack.L2NetworkID]dsl.ConductorSet)
	flashblocksBuilderSets := make(map[stack.L2NetworkID]dsl.FlashblocksBuilderSet)
	faucets := make(map[stack.L2NetworkID]*dsl.Faucet)
	funders := make(map[stack.L2NetworkID]*dsl.Funder)
	l2ELNodes := make(map[stack.L2NetworkID]*dsl.L2ELNode)
	flashblocksWebsocketProxies := make(map[stack.L2NetworkID]dsl.FlashblocksWebsocketProxySet)

	for _, chain := range chains {
		chainMatcher := match.L2ChainById(chain.ID())
		l2 := system.L2Network(match.Assume(t, chainMatcher))
		firstELNode := dsl.NewL2ELNode(l2.L2ELNode(match.FirstL2EL), orch.ControlPlane())
		firstFaucet := dsl.NewFaucet(l2.Faucet(match.Assume(t, match.FirstFaucet)))

		conductorSets[chain.ID()] = dsl.NewConductorSet(l2.Conductors())
		flashblocksBuilderSets[chain.ID()] = dsl.NewFlashblocksBuilderSet(l2.FlashblocksBuilders())
		flashblocksWebsocketProxies[chain.ID()] = dsl.NewFlashblocksWebsocketProxySet(l2.FlashblocksWebsocketProxies())

		faucets[chain.ID()] = firstFaucet
		funders[chain.ID()] = dsl.NewFunder(minimalPreset.Wallet, firstFaucet, firstELNode)
		l2ELNodes[chain.ID()] = firstELNode
	}
	return &SimpleFlashblocks{
		Minimal:                     minimalPreset,
		ConductorSets:               conductorSets,
		FlashblocksBuilderSets:      flashblocksBuilderSets,
		FlashblocksWebsocketProxies: flashblocksWebsocketProxies,
		Faucets:                     faucets,
		Funders:                     funders,
		L2ELNodes:                   l2ELNodes,
	}
}
