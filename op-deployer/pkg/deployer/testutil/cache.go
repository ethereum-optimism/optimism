package testutil

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// SharedCacheDir returns a cache directory that the tests of this user share, in the temporary
// directory, so that tests that use the local contracts build copy it once, not once per test.
// op-deployer keeps the copies of the last few builds there.
func SharedCacheDir(t *testing.T) string {
	dir := filepath.Join(os.TempDir(), fmt.Sprintf("op-deployer-test-cache-%d", os.Getuid()))
	require.NoError(t, os.MkdirAll(dir, 0o755))
	return dir
}
