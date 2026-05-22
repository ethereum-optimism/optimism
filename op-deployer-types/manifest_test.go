package opdeployertypes

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-deployer-types/generated/deploysuperchain"
	"github.com/stretchr/testify/require"
)

func TestManifestHashMatchesContents(t *testing.T) {
	hash, err := Manifest.Hash()
	require.NoError(t, err)
	require.Equal(t, Manifest.SchemaHash, hash)
}

func TestGeneratedTypesCompile(t *testing.T) {
	var input deploysuperchain.Input
	input.Paused = true

	var output deploysuperchain.Output
	require.True(t, input.Paused)
	require.Empty(t, output)
}
