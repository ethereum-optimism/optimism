package sysgo

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestFindMonorepoRoot_EnvOverride covers the DEVSTACK_MONOREPO_ROOT branch, which no in-tree test
// exercises (the in-repo suites rely on the cwd-relative walk). It is the seam out-of-tree
// acceptance suites depend on, so it earns explicit in-repo coverage.
func TestFindMonorepoRoot_EnvOverride(t *testing.T) {
	const marker = "packages/contracts-bedrock"

	// A checkout containing the marker: root is returned, normalized with a trailing slash even
	// though the env value has none.
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, marker), 0o755))
	t.Setenv("DEVSTACK_MONOREPO_ROOT", root)

	got, err := findMonorepoRoot(marker)
	require.NoError(t, err)
	require.Equal(t, root+"/", got)

	// A directory that does not contain the marker: hard error rather than a silent fallback.
	t.Setenv("DEVSTACK_MONOREPO_ROOT", t.TempDir())
	_, err = findMonorepoRoot(marker)
	require.Error(t, err)
}
