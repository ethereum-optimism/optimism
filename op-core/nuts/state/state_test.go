package state

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-core/forks"
	"github.com/ethereum-optimism/optimism/op-core/predeploys"
)

// TestPreForkState guards that the states the activation test relies on are
// actually committed and load — asserting each consuming fork resolves one, with a
// real predeploy present. This catches a missing or unembedded state here (a fast,
// kona-host-free check) rather than only when the activation test runs.
func TestPreForkState(t *testing.T) {
	// Every fork from karst onward boots from its predecessor's committed state.
	// forks.From(karst) auto-covers future forks: adding one without generating
	// its predecessor's state will fail here.
	for _, fork := range forks.From(forks.Karst) {
		alloc, err := PreForkState(fork)
		require.NoErrorf(t, err, "a pre-fork state should back %s", fork)
		require.NotEmpty(t, alloc)

		feeVault, found := alloc[predeploys.SequencerFeeVaultAddr]
		require.Truef(t, found, "SequencerFeeVault proxy must be present in %s pre-fork state", fork)
		require.NotEmptyf(t, feeVault.Code, "SequencerFeeVault proxy must have code in %s pre-fork state", fork)
	}
}

// TestPreForkStateWithoutPreforkState confirms PreForkState errors for every fork
// without a usable pre-fork state: bedrock..jovian inclusive.
func TestPreForkStateWithoutPreforkState(t *testing.T) {
	for _, fork := range forks.All {
		_, err := PreForkState(fork)
		require.Errorf(t, err, "PreForkState(%s) should error: no committed pre-fork state", fork)
		if fork == forks.Jovian {
			break
		}
	}
}
