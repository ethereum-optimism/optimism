package state

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-core/forks"
	"github.com/ethereum-optimism/optimism/op-core/predeploys"
)

// TestPreForkState checks the committed states back the forks that consume them
// and contain the predeploy at the center of the #20982 downgrade revert
// (SequencerFeeVault).
func TestPreForkState(t *testing.T) {
	// karst boots from jovian_state; lagoon boots from karst_state.
	for _, fork := range []forks.Name{forks.Karst, forks.Lagoon} {
		alloc, ok, err := PreForkState(fork)
		require.NoError(t, err)
		require.Truef(t, ok, "a pre-fork state should back %s", fork)
		require.NotEmpty(t, alloc)

		feeVault, found := alloc[predeploys.SequencerFeeVaultAddr]
		require.Truef(t, found, "SequencerFeeVault proxy must be present in %s pre-fork state", fork)
		require.NotEmptyf(t, feeVault.Code, "SequencerFeeVault proxy must have code in %s pre-fork state", fork)
	}
}

// TestPreForkStateMissing confirms a fork with no committed predecessor state
// reports ok=false (so callers fall back to building genesis from source).
func TestPreForkStateMissing(t *testing.T) {
	// bedrock has no predecessor fork, so there is no state to back it.
	_, ok, err := PreForkState(forks.Bedrock)
	require.NoError(t, err)
	require.False(t, ok)
}
