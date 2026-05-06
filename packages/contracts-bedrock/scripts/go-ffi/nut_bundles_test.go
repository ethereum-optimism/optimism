package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ethereum-optimism/optimism/op-core/nuts"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/stretchr/testify/require"
)

func TestOrderedNUTBundlesFromLocks(t *testing.T) {
	root := mockMonorepo(t)
	writeBundle(t, root, "op-core/nuts/bundles/karst_nut_bundle.json")
	writeBundle(t, root, "op-core/nuts/bundles/interop_nut_bundle.json")

	bundles, err := orderedNUTBundlesFromLocks(nuts.ForkLock{
		"interop": {Bundle: "op-core/nuts/bundles/interop_nut_bundle.json"},
		"karst":   {Bundle: "op-core/nuts/bundles/karst_nut_bundle.json"},
	}, root)

	require.NoError(t, err)
	require.Equal(t, []NUTBundleEncoded{
		{Fork: "karst", Path: "../../op-core/nuts/bundles/karst_nut_bundle.json"},
		{Fork: "interop", Path: "../../op-core/nuts/bundles/interop_nut_bundle.json"},
	}, bundles)
}

func TestOrderedNUTBundlesFromLocksRejectsUnknownFork(t *testing.T) {
	root := mockMonorepo(t)
	writeBundle(t, root, "op-core/nuts/bundles/example_nut_bundle.json")

	_, err := orderedNUTBundlesFromLocks(nuts.ForkLock{
		"example": {Bundle: "op-core/nuts/bundles/example_nut_bundle.json"},
	}, root)

	require.ErrorContains(t, err, `locked fork "example" is not in forks.All`)
}

func TestOrderedNUTBundlesFromLocksRejectsEmptyBundlePath(t *testing.T) {
	root := mockMonorepo(t)

	_, err := orderedNUTBundlesFromLocks(nuts.ForkLock{
		"karst": {Bundle: ""},
	}, root)

	require.ErrorContains(t, err, "bundle path is empty")
}

func TestOrderedNUTBundlesFromLocksRejectsEscapedBundlePath(t *testing.T) {
	root := mockMonorepo(t)

	_, err := orderedNUTBundlesFromLocks(nuts.ForkLock{
		"karst": {Bundle: "../karst_nut_bundle.json"},
	}, root)

	require.ErrorContains(t, err, "escapes monorepo root")
}

func TestOrderedNUTBundlesFromLocksRejectsMissingBundleFile(t *testing.T) {
	root := mockMonorepo(t)

	_, err := orderedNUTBundlesFromLocks(nuts.ForkLock{
		"karst": {Bundle: "op-core/nuts/bundles/missing_nut_bundle.json"},
	}, root)

	require.ErrorContains(t, err, "checking bundle file")
}

func TestNUTBundleABIEncoding(t *testing.T) {
	nutBundleType, err := abi.NewType("tuple[]", "", []abi.ArgumentMarshaling{
		{Name: "fork", Type: "string"},
		{Name: "path", Type: "string"},
	})
	require.NoError(t, err)

	args := abi.Arguments{{Type: nutBundleType}}
	encoded, err := args.Pack([]NUTBundleEncoded{
		{Fork: "karst", Path: "../../op-core/nuts/bundles/karst_nut_bundle.json"},
	})
	require.NoError(t, err)
	require.NotEmpty(t, encoded)
}

func mockMonorepo(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "packages", "contracts-bedrock"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "op-core", "nuts", "bundles"), 0o755))
	return root
}

func writeBundle(t *testing.T, root string, path string) {
	t.Helper()

	absPath := filepath.Join(root, path)
	require.NoError(t, os.MkdirAll(filepath.Dir(absPath), 0o755))
	require.NoError(t, os.WriteFile(absPath, []byte("[]"), 0o644))
}
