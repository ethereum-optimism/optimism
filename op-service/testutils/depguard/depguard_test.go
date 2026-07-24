package depguard

import (
	"testing"

	"github.com/stretchr/testify/require"

	// Imported only by this test file, to pin that test imports are exempt.
	_ "github.com/ethereum-optimism/optimism/op-service/bigs"
)

// TestFindForbiddenChains is the positive control for the guard: a broken
// package loader (e.g. wrong Mode flags yielding an empty import graph) would
// make every guard test pass silently, so prove the walk actually finds
// dependencies this package is known to have.
func TestFindForbiddenChains(t *testing.T) {
	t.Run("detects known transitive import", func(t *testing.T) {
		// go/token is not imported directly, only via x/tools/go/packages.
		chains, err := findForbiddenChains(".", "go/token")
		require.NoError(t, err)
		require.NotEmpty(t, chains)
		require.Contains(t, chains[0], " -> go/token")
	})

	t.Run("test-only imports are exempt", func(t *testing.T) {
		// Dependents of a package never build its test files, so the guard
		// walks only the production closure.
		chains, err := findForbiddenChains(".", "github.com/ethereum-optimism/optimism/op-service/bigs")
		require.NoError(t, err)
		require.Empty(t, chains)
	})

	t.Run("clean for absent import", func(t *testing.T) {
		chains, err := findForbiddenChains(".", "example.com/definitely/not/imported")
		require.NoError(t, err)
		require.Empty(t, chains)
	})

	t.Run("broken pattern cannot silently pass", func(t *testing.T) {
		chains, err := findForbiddenChains("./no-such-package", "go/token")
		if err == nil {
			require.NotEmpty(t, chains)
		}
	})
}
