package presets

import (
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
)

// NewStagedL2ELFrontend exposes an already-started product-neutral runtime EL
// through the ordinary devstack DSL without constructing a preset or starting
// ancillary services. Interactive and staged launchers use this after the
// corresponding explicit StartEL action has completed.
func NewStagedL2ELFrontend(t devtest.T, name string, network *sysgo.L2Network, node sysgo.L2ELNode) *dsl.L2ELNode {
	t.Require().NotNil(network, "staged L2 network is required")
	t.Require().NotNil(node, "staged L2 EL is required")
	return dsl.NewL2ELNode(newL2ELFrontend(
		t, name, network.ChainID(), node.UserRPC(), node.EngineRPC(), node.JWTPath(),
		network.RollupConfig(), node,
	))
}

// NewStagedL2CLFrontend exposes an already-started product-neutral runtime CL
// and attaches it to the matching staged EL frontend.
func NewStagedL2CLFrontend(t devtest.T, name string, network *sysgo.L2Network, node sysgo.L2CLNode, el *dsl.L2ELNode) *dsl.L2CLNode {
	t.Require().NotNil(network, "staged L2 network is required")
	t.Require().NotNil(node, "staged L2 CL is required")
	t.Require().NotNil(el, "staged L2 EL frontend is required")
	frontend := newL2CLFrontend(t, name, network.ChainID(), node.UserRPC(), node)
	backend, ok := el.Escape().(*l2ELFrontend)
	t.Require().True(ok, "staged L2 EL frontend has unexpected implementation")
	frontend.attachEL(backend)
	return dsl.NewL2CLNode(frontend)
}
