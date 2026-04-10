package manager

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/oprm/journal"
	"github.com/ethereum-optimism/optimism/oprm/providers/ghcli"
	"github.com/stretchr/testify/require"
)

func TestDiscoverComponentVersionsUsesConfigAwareRepos(t *testing.T) {
	cfg := DefaultConfig()
	cfg.GitHub.Owner = "nonsense"
	cfg.GitHub.Repo = "optimism"
	cfg.OpGeth.Owner = "nonsense"
	cfg.OpGeth.Repo = "op-geth"

	gh := &fakeGHProvider{
		releasesByRepo: map[string][]ghcli.Release{
			"nonsense/optimism": {
				{TagName: "op-node/v1.2.3"},
				{TagName: "op-node/v1.2.4-rc.2", PreRelease: true},
				{TagName: "op-batcher/v0.9.1"},
			},
			"nonsense/op-geth": {
				{TagName: "v1.101605.0"},
				{TagName: "v1.101605.1-rc.1", PreRelease: true},
			},
		},
	}
	app := NewWithProviders(cfg, io.Discard, io.Discard, journal.NewStore(t.TempDir()), &fakeGitProvider{}, gh, func() time.Time {
		return time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
	})

	items, err := app.DiscoverComponentVersions(context.Background(), []string{"op-node", "op-geth"})
	require.NoError(t, err)
	require.Len(t, items, 2)
	require.Equal(t, "nonsense", items[0].Component.GitHubOwner)
	require.Equal(t, "optimism", items[0].Component.GitHubRepo)
	require.Equal(t, "v1.2.3", items[0].LatestRelease.Version)
	require.Equal(t, "v1.2.4-rc.2", items[0].LatestRC.Version)
	require.Equal(t, "nonsense", items[1].Component.GitHubOwner)
	require.Equal(t, "op-geth", items[1].Component.GitHubRepo)
	require.Equal(t, "v1.101605.0", items[1].LatestRelease.Version)
	require.Equal(t, "v1.101605.1-rc.1", items[1].LatestRC.Version)
	require.Equal(t, 2, gh.listCalls)
}
