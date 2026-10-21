package main

import (
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestCollectDrift type-checks testdata/toy/mod against testdata/toy/latest, which lacks
// types.Holesky, skipping the packages that reach a fork-API use against testdata/toy/upstream.
func TestCollectDrift(t *testing.T) {
	uses := toyUses(t)
	toy, modFile := toyModFile(t, "latest")
	drift, err := collectDrift(t.Context(), filepath.Join(toy, "mod"), "example.com/toy", uses,
		[]string{"-modfile=" + modFile}, []string{"./..."})
	require.NoError(t, err)
	require.Equal(t, []string{
		"h/h.go:5:17: undefined: types.Holesky",
		// Test files are checked too.
		"h/h_test.go:9:38: undefined: types.Holesky",
		// d's tests use a fork API, which doesn't affect its importers. Neither c, which imports
		// a, nor d's tests are checked.
		"i/i.go:9:23: undefined: types.Holesky",
	}, drift)
}

func TestCollectDriftFailsOnDependencyErrors(t *testing.T) {
	drift, err := collectDrift(t.Context(), "testdata/toy/baddep", "example.com/baddep", nil, nil, []string{"./..."})
	require.NoError(t, err)
	require.True(t, slices.ContainsFunc(drift, func(s string) bool {
		return strings.HasPrefix(s, "example.com/dep: ") && strings.Contains(s, "could not import example.com/nowhere")
	}), drift)
}
