package main

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"

	"github.com/ethereum-optimism/optimism/op-core/nuts"
	"github.com/stretchr/testify/require"
)

func hashOf(data []byte) string {
	h := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(h[:])
}

func TestValidateEntry_MatchingHash(t *testing.T) {
	content := []byte(`{"transactions":[]}`)
	entry := nuts.ForkLockEntry{
		Bundle: "op-core/nuts/bundles/test_nut_bundle.json",
		Hash:   hashOf(content),
		Commit: "abc123",
	}
	err := validateEntry("test", entry, content)
	require.NoError(t, err)
}

func TestValidateEntry_HashMismatch(t *testing.T) {
	content := []byte(`{"transactions":[]}`)
	entry := nuts.ForkLockEntry{
		Bundle: "op-core/nuts/bundles/test_nut_bundle.json",
		Hash:   "sha256:0000000000000000000000000000000000000000000000000000000000000000",
		Commit: "abc123",
	}
	err := validateEntry("test", entry, content)
	require.ErrorContains(t, err, "bundle hash mismatch")
}

func TestValidateEntry_EmptyCommit(t *testing.T) {
	content := []byte(`{"transactions":[]}`)
	entry := nuts.ForkLockEntry{
		Bundle: "op-core/nuts/bundles/test_nut_bundle.json",
		Hash:   hashOf(content),
		Commit: "",
	}
	err := validateEntry("test", entry, content)
	require.ErrorContains(t, err, "no commit recorded")
}

func TestValidateEntry_ModifiedBundle(t *testing.T) {
	original := []byte(`{"transactions":[{"intent":"deploy"}]}`)
	modified := []byte(`{"transactions":[{"intent":"modified"}]}`)
	entry := nuts.ForkLockEntry{
		Bundle: "op-core/nuts/bundles/test_nut_bundle.json",
		Hash:   hashOf(original),
		Commit: "abc123",
	}
	err := validateEntry("test", entry, modified)
	require.ErrorContains(t, err, "bundle hash mismatch")
}

func TestCheckPreForkState(t *testing.T) {
	writeStates := func(t *testing.T, names ...string) string {
		root := t.TempDir()
		dir := filepath.Join(root, "op-core", "nuts", "state")
		require.NoError(t, os.MkdirAll(dir, 0o755))
		for _, name := range names {
			require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte("{}"), 0o644))
		}
		return root
	}

	t.Run("single-chain fork needs only its state", func(t *testing.T) {
		root := writeStates(t, "karst_state.json")
		require.NoError(t, checkPreForkState(root, "karst"))
	})
	t.Run("multi-chain fork with both states", func(t *testing.T) {
		root := writeStates(t, "lagoon_state.json", "lagoon_multichain_state.json")
		require.NoError(t, checkPreForkState(root, "lagoon"))
	})
	t.Run("multi-chain fork without its multi-chain state", func(t *testing.T) {
		root := writeStates(t, "lagoon_state.json")
		require.ErrorContains(t, checkPreForkState(root, "lagoon"), "lagoon_multichain_state.json")
	})
	t.Run("fork without its state", func(t *testing.T) {
		root := writeStates(t, "lagoon_multichain_state.json")
		require.ErrorContains(t, checkPreForkState(root, "lagoon"), "lagoon_state.json")
	})
}
