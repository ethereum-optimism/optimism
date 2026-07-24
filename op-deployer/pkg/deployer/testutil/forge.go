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
func NewForgeClient(t *testing.T, workdir string) *forge.Client {
	client, err := forge.NewStandardClient(workdir)
	require.NoError(t, err)
	client.Timeout = 10 * time.Minute
	client.SlowBroadcast = true
	return client
}
