package tests

import (
	"github.com/ethereum-optimism/optimism/op-test/components/l1cl"
	"github.com/ethereum-optimism/optimism/op-test/components/l2"
	"github.com/ethereum-optimism/optimism/op-test/components/l2cl"
	"github.com/ethereum-optimism/optimism/op-test/components/l2el"
	"github.com/ethereum-optimism/optimism/op-test/components/superchain"
	"testing"

	"github.com/stretchr/testify/require"

	test "github.com/ethereum-optimism/optimism/op-test"
	"github.com/ethereum-optimism/optimism/op-test/components/l1"
	"github.com/ethereum-optimism/optimism/op-test/components/l1el"
)

func TestDemo(t *testing.T) {
	test.Test(t, func(t test.Testing) {
		// We request resources from the backend
		// the test will abort (skip on 0 selected parameters) if the requests fail.

		// The test framework handles parametrization of what is chosen to compose with.
		l1BackendKind := test.Select(t, "l1_backend", test.Live, test.Instant)
		l1Fork := test.Select(t, "l1_fork", l1.Forks...)
		l2BackendKind := test.Select(t, "l2_backend", test.Live, test.Instant)

		// create L1 chain, engine and beacon node
		l1Chain := l1.Request(t, l1.Kind(l1BackendKind), l1.ActiveFork(l1Fork))
		l1EL := l1el.Request(t, l1el.Kind(l1BackendKind))
		l1CL := l1cl.Request(t, l1cl.Kind(l1BackendKind), l1cl.L1EL(l1EL))

		// create a superchain to group L2s
		superChain := superchain.Request(t,
			superchain.Kind(l1BackendKind), superchain.L1Chain(l1Chain))

		// create L2 chain, op-stack engine and rollup node
		l2Chain := l2.Request(t, l2.Superchain(superChain), l2.Kind(l2BackendKind))
		l2EL := l2el.Request(t, l2el.Kind(l2BackendKind), l2el.L2Chain(l2Chain))
		l2CL := l2cl.Request(t, l2cl.Kind(l2BackendKind), l2cl.L2EL(l2EL))

		// TODO request l1CL to mine blocks

		// TODO less direct RPC/client bindings usage,
		// and more DSL-style interactions with each actor

		status, err := l2CL.RollupClient().SyncStatus(t.Ctx())
		require.NoError(t, err)
		t.Logf("L2 sync status: %v", status)

		cl := l1EL.L1Client()
		chainID, err := cl.ChainID(t.Ctx())
		require.NoError(t, err)
		t.Logf("got L1 engine on chain: %d", chainID)
	})
}
