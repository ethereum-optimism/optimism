package main

import (
	"crypto/sha256"
	"encoding/hex"
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
		Bundle: "op-node/rollup/derive/test_nut_bundle.json",
		Hash:   hashOf(content),
		Commit: "abc123",
	}
	err := validateEntry("test", entry, content)
	require.NoError(t, err)
}

func TestValidateEntry_HashMismatch(t *testing.T) {
	content := []byte(`{"transactions":[]}`)
	entry := nuts.ForkLockEntry{
		Bundle: "op-node/rollup/derive/test_nut_bundle.json",
		Hash:   "sha256:0000000000000000000000000000000000000000000000000000000000000000",
		Commit: "abc123",
	}
	err := validateEntry("test", entry, content)
	require.ErrorContains(t, err, "bundle hash mismatch")
}

func TestValidateEntry_EmptyCommit(t *testing.T) {
	content := []byte(`{"transactions":[]}`)
	entry := nuts.ForkLockEntry{
		Bundle: "op-node/rollup/derive/test_nut_bundle.json",
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
		Bundle: "op-node/rollup/derive/test_nut_bundle.json",
		Hash:   hashOf(original),
		Commit: "abc123",
	}
	err := validateEntry("test", entry, modified)
	require.ErrorContains(t, err, "bundle hash mismatch")
}
