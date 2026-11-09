package testutil

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/testutils"
)

// SharedCacheDir returns a cache directory that the tests of this user share, in the temporary
// directory, so that tests that use the local contracts build copy it once, not once per test.
// op-deployer keeps the copies of the last few builds there. When the directory is not a private one
// of this user, SharedCacheDir returns a directory of the test.
func SharedCacheDir(t *testing.T) string {
	dir := filepath.Join(os.TempDir(), fmt.Sprintf("op-deployer-test-cache-%d", os.Getuid()))
	if err := os.MkdirAll(dir, 0o700); err == nil && isPrivateDir(dir) {
		return dir
	}
	return testutils.IsolatedTestDirWithAutoCleanup(t)
}
