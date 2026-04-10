package release

import (
	"testing"

	"github.com/ethereum-optimism/optimism/oprm/components"
	"github.com/ethereum-optimism/optimism/oprm/providers/ghcli"
	"github.com/stretchr/testify/require"
)

func TestExtractComponentVersion(t *testing.T) {
	registry := components.NewRegistry()

	opNode := registry.MustGet("op-node")
	version, ok := ExtractComponentVersion(opNode, "op-node/v1.2.3")
	require.True(t, ok)
	require.Equal(t, "v1.2.3", version)

	_, ok = ExtractComponentVersion(opNode, "op-batcher/v1.2.3")
	require.False(t, ok)

	opGeth := registry.MustGet("op-geth")
	version, ok = ExtractComponentVersion(opGeth, "v1.101605.0-rc.1")
	require.True(t, ok)
	require.Equal(t, "v1.101605.0-rc.1", version)
}

func TestDiscoverLatestVersionsForMonorepoComponent(t *testing.T) {
	spec := components.NewRegistry().MustGet("op-node")
	releases := []ghcli.Release{
		{TagName: "op-node/v1.2.3", URL: "https://example/op-node/v1.2.3"},
		{TagName: "op-node/v1.2.4-rc.1", URL: "https://example/op-node/v1.2.4-rc.1", PreRelease: true},
		{TagName: "op-node/v1.2.4-rc.2", URL: "https://example/op-node/v1.2.4-rc.2", PreRelease: true},
		{TagName: "op-node/v1.2.2"},
		{TagName: "op-batcher/v9.9.9"},
		{TagName: "op-node/v9.9.9", Draft: true},
	}

	discovered, err := DiscoverLatestVersions(spec, releases)
	require.NoError(t, err)
	require.NotNil(t, discovered.LatestRelease)
	require.Equal(t, "v1.2.3", discovered.LatestRelease.Version)
	require.NotNil(t, discovered.LatestRC)
	require.Equal(t, "v1.2.4-rc.2", discovered.LatestRC.Version)
	require.True(t, discovered.LatestRC.PreRelease)
}

func TestDiscoverLatestVersionsForOpGeth(t *testing.T) {
	spec := components.NewRegistry().MustGet("op-geth")
	releases := []ghcli.Release{
		{TagName: "v1.101605.0", URL: "https://example/op-geth/v1.101605.0"},
		{TagName: "v1.101605.1-rc.1", URL: "https://example/op-geth/v1.101605.1-rc.1", PreRelease: true},
		{TagName: "v1.101605.1-rc.3", URL: "https://example/op-geth/v1.101605.1-rc.3", PreRelease: true},
		{TagName: "v1.101604.9"},
	}

	discovered, err := DiscoverLatestVersions(spec, releases)
	require.NoError(t, err)
	require.Equal(t, "v1.101605.0", discovered.LatestRelease.Version)
	require.Equal(t, "v1.101605.1-rc.3", discovered.LatestRC.Version)
}

func TestDiscoverLatestVersionsIncludesDraftRCs(t *testing.T) {
	spec := components.NewRegistry().MustGet("op-node")
	releases := []ghcli.Release{
		{TagName: "op-node/v1.16.11", URL: "https://example/op-node/v1.16.11"},
		{TagName: "op-node/v1.16.12-rc.1", URL: "https://example/op-node/v1.16.12-rc.1", Draft: true},
	}

	discovered, err := DiscoverLatestVersions(spec, releases)
	require.NoError(t, err)
	require.NotNil(t, discovered.LatestRC)
	require.Equal(t, "v1.16.12-rc.1", discovered.LatestRC.Version)
	require.True(t, discovered.LatestRC.Draft)
}

func TestDiscoverLatestVersionsRejectsMalformedMatchingTag(t *testing.T) {
	spec := components.NewRegistry().MustGet("op-node")
	_, err := DiscoverLatestVersions(spec, []ghcli.Release{{TagName: "op-node/not-a-version"}})
	require.Error(t, err)
}
