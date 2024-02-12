package tests

import (
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

		// Backends and requested handles on actors are composable.
		// The test framework handles parametrization of what is chosen to compose with.
		// We can make functions like "SingleMinerL1", or "TwoChainInteropL2"
		// that compose the backends and actors we need in a test.
		l1BackendKind := test.Select(t, "l1_backend", test.Live, test.Instant)
		l1Backend := l1.NewBackend(t, l1BackendKind)
		l1Fork := l1.ActiveFork(test.Select(t, "l1_fork", l1.Forks...))
		l1Chain := l1Backend.RequestL1(l1.Layer1, l1Fork)
		l1ELBackend := l1el.NewBackend(t, l1Chain, l1BackendKind)
		l1EL := l1ELBackend.RequestL1EL(l1el.BuilderA, l1el.BlockBuilding(true))

		cl := l1EL.L1Client()
		chainID, err := cl.ChainID(t.Ctx())
		require.NoError(t, err)
		t.Logf("got L1 client handle, on chain ID: %d", chainID)
	})
}
