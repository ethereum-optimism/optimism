package tests

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-test/components/l1"
	"github.com/ethereum-optimism/optimism/op-test/components/l1cl"
	"github.com/ethereum-optimism/optimism/op-test/components/l1el"
	"github.com/ethereum-optimism/optimism/op-test/components/l2"
	"github.com/ethereum-optimism/optimism/op-test/components/l2cl"
	"github.com/ethereum-optimism/optimism/op-test/components/l2el"
	"github.com/ethereum-optimism/optimism/op-test/components/superchain"
	"github.com/ethereum-optimism/optimism/op-test/test"
)

func TestMain(m *testing.M) {
	test.Main(m)
}

func TestDemo(t *testing.T) {
	test.Plan(t, func(t test.Planner) {
		t.Select("example", []string{"foo", "bar"}, func(t test.Planner) {

			t.Plan("sub-plan", func(t test.Planner) {

				// We request resources from the backend
				// the test will abort (skip on 0 selected parameters) if the requests fail.

				// The test framework handles parametrization of what is chosen to compose with.
				test.Select(t, "l1_backend", []test.BackendKind{test.Live, test.Instant}, func(t test.Planner, l1BackendKind test.BackendKind) {
					test.Select(t, "l1_fork", l1.Forks, func(t test.Planner, l1Fork l1.L1Fork) {
						test.Select(t, "l2_backend", []test.BackendKind{test.Live, test.Instant}, func(t test.Planner, l2BackendKind test.BackendKind) {

							// TODO: should the resource-configuration become part of the test-plan?
							// We can compose resources, without allowing the resources to execute actual tasks.
							// They can just be unique identifiers and setting-structs.

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

							t.Run("run", func(t test.Executor) {

								// TODO request l1CL to mine blocks
								_ = l1CL.BeaconEndpoint()

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
						})
					})

				})
			})
		})
	})
}
