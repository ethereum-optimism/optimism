package e2eutils

import (
	"os"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

// FindMonorepoRoot finds the relative path to the monorepo root
// Different tests might be nested in subdirectories of the op-e2e dir.
func FindMonorepoRoot(t *testing.T) string {
	path := "./"
	// Only search up 5 directories
	// Avoids infinite recursion if the root isn't found for some reason
	for i := 0; i < 5; i++ {
		_, err := os.Stat(path + "op-e2e")
		if errors.Is(err, os.ErrNotExist) {
			path = path + "../"
			continue
		}
		require.NoErrorf(t, err, "Failed to stat %v even though it existed", path)
		return path
	}
	t.Fatalf("Could not find monorepo root, trying up to %v", path)
	return ""
}
