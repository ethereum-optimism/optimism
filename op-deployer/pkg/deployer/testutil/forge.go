package testutil

import (
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/forge"
	"github.com/stretchr/testify/require"
)

// NewForgeClient returns a forge client hardened for devnet tests: each
// invocation is bounded so a stuck forge fails the test instead of hanging
// until the CI job timeout, and broadcasts run with --slow.
//
// The timeout only needs to be an order of magnitude below the 90-minute job
// watchdog, not tight: the worst healthy forge-using test on develop finishes
// in under 10 seconds, but the bound also has to absorb episodic CI host
// contention (documented multi-second-to-minute stalls), --slow broadcasts
// waiting on a receipt per tx, and a cold forge binary download, all of which
// a too-tight value would convert into new flakes.
func NewForgeClient(t *testing.T, workdir string) *forge.Client {
	client, err := forge.NewStandardClient(workdir)
	require.NoError(t, err)
	client.Timeout = 5 * time.Minute
	client.SlowBroadcast = true
	return client
}
