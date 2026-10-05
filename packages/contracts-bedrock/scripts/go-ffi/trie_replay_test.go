package main

import (
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTrieReplayReproducesCorpusAcrossReads(t *testing.T) {
	seed := "0x" + strings.Repeat("ab", 32)
	one, err := trieReplayReader(seed, []string{"trie", valid}, "true", "cicoverage")
	require.NoError(t, err)
	two, err := trieReplayReader(seed, []string{"trie", valid}, "true", "cicoverage")
	require.NoError(t, err)
	want := make([]byte, 4096)
	_, err = io.ReadFull(one, want)
	require.NoError(t, err)
	got := make([]byte, len(want))
	for _, span := range [][2]int{{0, 1}, {1, 63}, {63, 127}, {127, 4096}} {
		_, err = io.ReadFull(two, got[span[0]:span[1]])
		require.NoError(t, err)
	}
	require.Equal(t, want, got)
	for _, input := range []struct {
		seed string
		args []string
	}{{"0x" + strings.Repeat("cd", 32), []string{"trie", valid}}, {seed, []string{"trie", corruptedProof}}} {
		other, err := trieReplayReader(input.seed, input.args, "true", "cicoverage")
		require.NoError(t, err)
		_, err = io.ReadFull(other, got)
		require.NoError(t, err)
		require.NotEqual(t, want, got)
	}
}

func TestTrieReplayRequiresExplicitValidCoverageInput(t *testing.T) {
	reader, err := trieReplayReader("", []string{"trie", valid}, "false", "default")
	require.NoError(t, err)
	require.Nil(t, reader)
	seed := "0x" + strings.Repeat("ab", 32)
	for _, input := range []struct{ seed, ci, profile string }{
		{seed, "false", "cicoverage"}, {seed, "true", "ci"},
		{"invalid", "true", "cicoverage"}, {"ab", "true", "cicoverage"},
	} {
		_, err := trieReplayReader(input.seed, nil, input.ci, input.profile)
		require.Error(t, err)
	}
}
