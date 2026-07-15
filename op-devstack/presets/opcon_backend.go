package presets

import (
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
)

// SysgoOpConNode returns the sysgo *OpConNode launcher object backing the given
// DSL L2CL node.
//
// The preset layer fronts every CL with an RPC frontend (l2CLFrontend) and
// keeps the launcher's node object only as its unexported stack.Lifecycle
// handle, so dsl Escape() cannot reach op-con-node-specific lifecycle
// primitives (Kill, WipeDatadir) needed by crash-restart and fresh-circuit
// boot tests. This accessor lives in the presets package purely to unwrap that
// frontend; tests using it should be guarded with sysgo.SkipUnlessOpConNode,
// since it fails the test when the node is not backed by op-con-node.
func SysgoOpConNode(t devtest.T, cl *dsl.L2CLNode) *sysgo.OpConNode {
	frontend, ok := cl.Escape().(*l2CLFrontend)
	t.Require().Truef(ok, "L2CL node %s is not a preset RPC frontend", cl.Name())
	opcon, ok := frontend.lifecycle.(*sysgo.OpConNode)
	t.Require().Truef(ok, "L2CL node %s is not backed by op-con-node", cl.Name())
	return opcon
}
