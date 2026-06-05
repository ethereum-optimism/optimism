package main

import (
	"os"
	"path/filepath"
	"reflect"
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
		"lagoon": {Bundle: "op-core/nuts/bundles/interop_nut_bundle.json"},
		"karst":  {Bundle: "op-core/nuts/bundles/karst_nut_bundle.json"},
	}, root)

	require.NoError(t, err)
	require.Equal(t, []NUTBundleEncoded{
		{Fork: "karst", Path: "../../op-core/nuts/bundles/karst_nut_bundle.json"},
		{Fork: "lagoon", Path: "../../op-core/nuts/bundles/interop_nut_bundle.json"},
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

func TestOrderedNUTBundlesFromLocksRejectsDirectoryBundlePath(t *testing.T) {
	root := mockMonorepo(t)

	_, err := orderedNUTBundlesFromLocks(nuts.ForkLock{
		"karst": {Bundle: "op-core/nuts/bundles"},
	}, root)

	require.ErrorContains(t, err, "is a directory")
}

func TestOrderedNUTBundlesFromLocksDropsPreKarstForks(t *testing.T) {
	root := mockMonorepo(t)
	writeBundle(t, root, "op-core/nuts/bundles/bedrock_nut_bundle.json")
	writeBundle(t, root, "op-core/nuts/bundles/karst_nut_bundle.json")

	bundles, err := orderedNUTBundlesFromLocks(nuts.ForkLock{
		"bedrock": {Bundle: "op-core/nuts/bundles/bedrock_nut_bundle.json"},
		"karst":   {Bundle: "op-core/nuts/bundles/karst_nut_bundle.json"},
	}, root)

	require.NoError(t, err)
	require.Equal(t, []NUTBundleEncoded{
		{Fork: "karst", Path: "../../op-core/nuts/bundles/karst_nut_bundle.json"},
	}, bundles)
}

func TestNUTBundleABIRoundTrip(t *testing.T) {
	nutBundleType, err := abi.NewType("tuple[]", "", []abi.ArgumentMarshaling{
		{Name: "fork", Type: "string"},
		{Name: "path", Type: "string"},
	})
	require.NoError(t, err)

	args := abi.Arguments{{Type: nutBundleType}}
	original := []NUTBundleEncoded{
		{Fork: "karst", Path: "../../op-core/nuts/bundles/karst_nut_bundle.json"},
		{Fork: "lagoon", Path: "../../op-core/nuts/bundles/interop_nut_bundle.json"},
	}

	encoded, err := args.Pack(original)
	require.NoError(t, err)
	require.NotEmpty(t, encoded)

	unpacked, err := args.Unpack(encoded)
	require.NoError(t, err)
	require.Len(t, unpacked, 1)

	decoded := reflect.ValueOf(unpacked[0])
	require.Equal(t, reflect.Slice, decoded.Kind())
	require.Equal(t, len(original), decoded.Len())
	for i, want := range original {
		got := decoded.Index(i)
		require.Equal(t, want.Fork, got.FieldByName("Fork").String())
		require.Equal(t, want.Path, got.FieldByName("Path").String())
	}
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
