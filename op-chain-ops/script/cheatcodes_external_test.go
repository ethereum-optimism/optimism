package script

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-chain-ops/foundry"
	"github.com/stretchr/testify/require"
)

func TestParseArtifactPathInput(t *testing.T) {
	t.Run("qualified path with forge artifacts prefix", func(t *testing.T) {
		name, contract, ok := parseArtifactPathInput("forge-artifacts/ScriptExample.s.sol/ScriptExample.json")
		require.True(t, ok)
		require.Equal(t, "ScriptExample.s.sol", name)
		require.Equal(t, "ScriptExample", contract)
	})

	t.Run("qualified path without prefix", func(t *testing.T) {
		name, contract, ok := parseArtifactPathInput("ScriptExample.s.sol/FooBar.json")
		require.True(t, ok)
		require.Equal(t, "ScriptExample.s.sol", name)
		require.Equal(t, "FooBar", contract)
	})

	t.Run("legacy contract identifier", func(t *testing.T) {
		_, _, ok := parseArtifactPathInput("ScriptExample")
		require.False(t, ok)
	})
}

func TestGetCodeSupportsExplicitArtifactPath(t *testing.T) {
	af := foundry.OpenArtifactsDir("./testdata/test-artifacts")
	c := &CheatCodesPrecompile{h: &Host{af: af}}

	legacyCode, err := c.GetCode("ScriptExample.s.sol:ScriptExample")
	require.NoError(t, err)

	pathCode, err := c.GetCode("forge-artifacts/ScriptExample.s.sol/ScriptExample.json")
	require.NoError(t, err)
	require.Equal(t, legacyCode, pathCode)
}
