package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/ethereum-optimism/optimism/op-core/nuts"
	"github.com/stretchr/testify/require"
)

func hashOf(data []byte) string {
	h := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(h[:])
}

// bundleWithExtraGas returns minimal but schema-valid bundle contents.
func bundleWithExtraGas(extraGas uint64) []byte {
	return fmt.Appendf(nil, `{"metadata":{"extraGas":%d,"version":%q},"transactions":[]}`,
		extraGas, nuts.BundleVersion)
}

func TestValidateEntry_MatchingHash(t *testing.T) {
	content := bundleWithExtraGas(0)
	entry := nuts.ForkLockEntry{
		Bundle: "op-core/nuts/bundles/test_nut_bundle.json",
		Hash:   hashOf(content),
		Commit: "abc123",
	}
	err := validateEntry("test", entry, content)
	require.NoError(t, err)
}

func TestValidateEntry_MatchingExtraGas(t *testing.T) {
	content := bundleWithExtraGas(150_000)
	entry := nuts.ForkLockEntry{
		Bundle:   "op-core/nuts/bundles/test_nut_bundle.json",
		Hash:     hashOf(content),
		Commit:   "abc123",
		ExtraGas: 150_000,
	}
	require.NoError(t, validateEntry("test", entry, content))
}

// The lock's extra_gas mirrors the bundle's own metadata.extraGas. Hand-editing either one
// alone must fail loudly rather than leave the reviewer-visible number lying about the
// activation block's gas reservation.
func TestValidateEntry_ExtraGasMismatch(t *testing.T) {
	content := bundleWithExtraGas(150_000)
	entry := nuts.ForkLockEntry{
		Bundle:   "op-core/nuts/bundles/test_nut_bundle.json",
		Hash:     hashOf(content),
		Commit:   "abc123",
		ExtraGas: 100_000,
	}
	err := validateEntry("test", entry, content)
	require.ErrorContains(t, err, "does not match the bundle's metadata.extraGas")
}

// A bundle predating the field must not carry one — a reader that predates it would silently
// under-reserve the activation block's gas limit.
func TestValidateEntry_ExtraGasAtLegacyVersion(t *testing.T) {
	content := fmt.Appendf(nil, `{"metadata":{"extraGas":150000,"version":%q},"transactions":[]}`,
		nuts.BundleVersionNoExtraGas)
	entry := nuts.ForkLockEntry{
		Bundle:   "op-core/nuts/bundles/test_nut_bundle.json",
		Hash:     hashOf(content),
		Commit:   "abc123",
		ExtraGas: 150_000,
	}
	err := validateEntry("test", entry, content)
	require.ErrorContains(t, err, "must not declare extraGas")
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
