package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestCollectUses type-checks testdata/toy/mod, which builds against testdata/toy/fork, against
// testdata/toy/upstream instead.
func TestCollectUses(t *testing.T) {
	toy, err := filepath.Abs("testdata/toy")
	require.NoError(t, err)
	modFile := filepath.Join(t.TempDir(), "go.mod")
	require.NoError(t, os.WriteFile(modFile, []byte(
		"module example.com/toy\n\ngo 1.24\n\nrequire example.com/geth v0.0.0\n\nreplace example.com/geth => "+
			filepath.Join(toy, "upstream")+"\n"), 0o644))

	uses, err := collectUses(t.Context(), filepath.Join(toy, "mod"), "example.com/toy", "example.com/geth",
		[]string{"-modfile=" + modFile}, []string{"./..."})
	require.NoError(t, err)
	require.Equal(t, map[Use]bool{
		// Resolved through the import alias.
		{"a", "missing go-ethereum/types.NewIsthmusSigner"}: true,
		// Reported although b's dependency a fails too; c, which uses only a, stays clean.
		{"b", "missing go-ethereum/types.Tx.IsDeposit"}: true,
		// Test files are checked too, and listed apart from production code. The external test
		// package d_test counts as d's tests.
		{"d (tests)", "missing go-ethereum/types.Tx.Mint"}:          true,
		{"d (tests)", "missing go-ethereum/types.NewIsthmusSigner"}: true,
		{"e", "missing package go-ethereum/interop"}:                true,
		{"e (tests)", "missing package go-ethereum/interop"}:        true,
		// Also when the extra argument is itself a call.
		{"f", "changed signature of go-ethereum/types.NewBlock"}: true,
		// A fork method that a type no longer has to satisfy an interface.
		{"f", "missing go-ethereum/types.ChainConfig.IsCanyon"}: true,
		// A parameter type changed.
		{"g", "changed signature of go-ethereum/types.Gas"}:                                                       true,
		{"g", `unclassified: cannot use "broken" (untyped string constant) as int value in variable declaration`}: true,
	}, uses)
}

// TestCollectUsesAgainstFork type-checks testdata/toy/mod against testdata/toy/fork, the module it
// builds against. Only g's deliberate type error remains, so every other use TestCollectUses
// expects is a fork API.
func TestCollectUsesAgainstFork(t *testing.T) {
	uses, err := collectUses(t.Context(), "testdata/toy/mod", "example.com/toy", "example.com/geth", nil, []string{"./..."})
	require.NoError(t, err)
	require.Equal(t, map[Use]bool{
		{"g", `unclassified: cannot use "broken" (untyped string constant) as int value in variable declaration`}: true,
	}, uses)
}

func TestCollectUsesFailsOnUnrelatedErrors(t *testing.T) {
	_, err := collectUses(t.Context(), "testdata/toy/bad", "example.com/bad", "example.com/geth", nil, []string{"./..."})
	require.ErrorContains(t, err, "example.com/nowhere")
}

func TestCollectUsesFailsOnDependencyErrors(t *testing.T) {
	_, err := collectUses(t.Context(), "testdata/toy/baddep", "example.com/baddep", "example.com/geth", nil, []string{"./..."})
	require.ErrorContains(t, err, "example.com/dep: ")
	require.ErrorContains(t, err, "could not import example.com/nowhere")
}
