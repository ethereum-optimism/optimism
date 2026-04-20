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

func TestKonaBundleMirror(t *testing.T) {
	got := konaBundleMirror("op-core/nuts/bundles/karst_nut_bundle.json")
	want := filepath.Join("rust", "kona", "crates", "protocol", "hardforks", "bundles", "karst_nut_bundle.json")
	require.Equal(t, want, got)
}

// writeMirror creates a fake kona-hardforks mirror at the expected relative path
// beneath root. Returns the full path for convenience.
func writeMirror(t *testing.T, root string, bundleRel string, content []byte) string {
	t.Helper()
	rel := konaBundleMirror(bundleRel)
	full := filepath.Join(root, rel)
	require.NoError(t, os.MkdirAll(filepath.Dir(full), 0o755))
	require.NoError(t, os.WriteFile(full, content, 0o600))
	return full
}

func TestCheckKonaMirror_Matches(t *testing.T) {
	root := t.TempDir()
	bundleRel := "op-core/nuts/bundles/test_nut_bundle.json"
	content := []byte(`{"transactions":[]}`)
	writeMirror(t, root, bundleRel, content)

	require.NoError(t, checkKonaMirror(root, "test", content, bundleRel))
}

func TestCheckKonaMirror_Missing(t *testing.T) {
	root := t.TempDir()
	bundleRel := "op-core/nuts/bundles/test_nut_bundle.json"
	content := []byte(`{"transactions":[]}`)

	err := checkKonaMirror(root, "test", content, bundleRel)
	require.ErrorContains(t, err, "reading kona bundle mirror")
	require.ErrorContains(t, err, "nut-snapshot-for test")
}

func TestCheckKonaMirror_Drift(t *testing.T) {
	root := t.TempDir()
	bundleRel := "op-core/nuts/bundles/test_nut_bundle.json"
	canonical := []byte(`{"transactions":[{"intent":"deploy"}]}`)
	mirror := []byte(`{"transactions":[{"intent":"drifted"}]}`)
	writeMirror(t, root, bundleRel, mirror)

	err := checkKonaMirror(root, "test", canonical, bundleRel)
	require.ErrorContains(t, err, "differs from canonical")
}
