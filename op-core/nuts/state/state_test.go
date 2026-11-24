package state

import (
	"io/fs"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
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

// TestMultiChainState checks that each multi-chain state has the interop contracts active, and
// that PreForkMultiChainState selects it for the next fork. Before Lagoon, the dependency set has
// no effect, so PreForkMultiChainState returns the single-chain state.
func TestMultiChainState(t *testing.T) {
	require.False(t, HasMultiChainState(forks.Karst))
	for _, fork := range forks.From(forks.Karst) {
		if HasMultiChainState(forks.Prev(fork)) {
			continue
		}
		single, err := PreForkState(fork)
		require.NoError(t, err)
		multi, err := PreForkMultiChainState(fork)
		require.NoError(t, err)
		require.Equalf(t, single, multi, "%s must boot both cases from the single-chain state", fork)
	}

	var featureKey [32]byte
	copy(featureKey[:], "INTEROP")
	// L1Block.isFeatureEnabled is mapping(bytes32 => bool) at storage slot 9.
	interopSlot := crypto.Keccak256Hash(featureKey[:], common.LeftPadBytes([]byte{9}, 32))
	// EIP-1967 implementation slot.
	implSlot := common.HexToHash("0x360894a13ba1a3210667c828492db98dca3e2076cc3735a920a3ca505d382bbc")
	// Check every committed multi-chain state. A fork gets its own state only after its bundle is
	// frozen, so this does not require a state for the newest fork. check-nut-locks requires it.
	names, err := fs.Glob(stateFS, "*_multichain_state.json")
	require.NoError(t, err)
	require.NotEmpty(t, names)
	for _, name := range names {
		fork := forks.Name(strings.TrimSuffix(name, "_multichain_state.json"))
		require.Truef(t, HasMultiChainState(fork), "%s has no multi-chain variant", fork)
		alloc, err := loadState(fork, name)
		require.NoError(t, err)
		require.NotEqualf(t, common.Hash{}, alloc[predeploys.L1BlockAddr].Storage[interopSlot],
			"INTEROP must be set in the %s multi-chain state", fork)
		impl := common.BytesToAddress(alloc[predeploys.CrossL2InboxAddr].Storage[implSlot].Bytes())
		require.NotEqualf(t, common.Address{}, impl, "CrossL2Inbox must have an implementation in the %s multi-chain state", fork)
		require.NotEmptyf(t, alloc[impl].Code, "CrossL2Inbox implementation must have code in the %s multi-chain state", fork)

		if next := forks.Next(fork); next != forks.None {
			selected, err := PreForkMultiChainState(next)
			require.NoError(t, err)
			require.Equalf(t, alloc, selected, "%s multi-chain case must boot from the %s multi-chain state", next, fork)
		}
	}

	// Every fork after Lagoon boots its multi-chain case from its predecessor's multi-chain state.
	for _, fork := range forks.From(forks.Lagoon)[1:] {
		_, err := PreForkMultiChainState(fork)
		require.NoErrorf(t, err, "a multi-chain pre-fork state should back %s", fork)
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
