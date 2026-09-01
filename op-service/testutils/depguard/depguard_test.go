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

// TestFindForbiddenChainsExcept is the positive control for the allowlist
// variant: an allowlist guard that can only ever pass is worthless, so prove
// both of its failure modes fire.
func TestFindForbiddenChainsExcept(t *testing.T) {
	const self = "github.com/ethereum-optimism/optimism/op-service/testutils/depguard"

	t.Run("reports a non-allowed package", func(t *testing.T) {
		offenders, stale, err := findForbiddenChainsExcept(".", nil, "go/token")
		require.NoError(t, err)
		require.NotEmpty(t, offenders)
		require.Contains(t, offenders[0], " -> go/token")
		require.Empty(t, stale)
	})

	t.Run("allowlisted package is exempt", func(t *testing.T) {
		offenders, stale, err := findForbiddenChainsExcept(".", []string{self}, "go/token")
		require.NoError(t, err)
		require.Empty(t, offenders)
		require.Empty(t, stale)
	})

	t.Run("stale allowlist entry is reported", func(t *testing.T) {
		// Allowed, but reaches nothing forbidden — the entry has outlived its reason.
		offenders, stale, err := findForbiddenChainsExcept(".", []string{self}, "example.com/definitely/not/imported")
		require.NoError(t, err)
		require.Empty(t, offenders)
		require.Len(t, stale, 1)
		require.Contains(t, stale[0], self)
	})

	t.Run("broken pattern cannot silently pass", func(t *testing.T) {
		_, _, err := findForbiddenChainsExcept("./no-such-package", nil, "go/token")
		require.Error(t, err)
	})
}
