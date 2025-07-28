package presets

import (
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/shim"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
)

type MultiELNodes struct {
	Minimal

	ELs []stack.L2ELNodeID
}

func WithMultiELNodes(n int) stack.CommonOption {
	return stack.MakeCommon(sysgo.DefaultSingleChainMultiELNodesSystem(&sysgo.DefaultSingleChainMultiELNodesSystemIDs{}, n))
}

func NewMultiELNodes(t devtest.T, n int) *MultiELNodes {
	system := shim.NewSystem(t)
	orch := Orchestrator()
	orch.Hydrate(system)
	minimal := minimalFromSystem(t, system, orch)
	ids := sysgo.NewDefaultSingleChainMultiELNodesSystemIDs(sysgo.DefaultL1ID, sysgo.DefaultL2AID, n)
	return &MultiELNodes{
		Minimal: *minimal,
		ELs:     ids.L2ELs,
	}
}
