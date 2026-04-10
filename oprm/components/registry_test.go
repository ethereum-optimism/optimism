package components

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRegistryContainsMVPComponents(t *testing.T) {
	registry := NewRegistry()
	require.Equal(t, []string{"kona-node", "op-batcher", "op-geth", "op-node", "op-reth"}, registry.IDs())

	opGeth := registry.MustGet("op-geth")
	require.Equal(t, KindExternalGo, opGeth.Kind)
	require.Equal(t, "ethereum-optimism", opGeth.GitHubOwner)
	require.Equal(t, "op-geth", opGeth.GitHubRepo)
	require.Equal(t, "optimism", opGeth.BaseBranch)

	opNode := registry.MustGet("op-node")
	require.Contains(t, opNode.ChangeScope, "op-node/**")
	require.Contains(t, opNode.ChangeScope, "op-service/**")
	require.True(t, opNode.Versioning.AutoIncrementRC)
}
