package manager

import (
	"context"
	"fmt"

	"github.com/ethereum-optimism/optimism/oprm/components"
	"github.com/ethereum-optimism/optimism/oprm/providers/ghcli"
	"github.com/ethereum-optimism/optimism/oprm/release"
)

type ComponentVersionDiscovery struct {
	Component     components.ComponentSpec
	LatestRelease *release.MatchedRelease
	LatestRC      *release.MatchedRelease
	LatestDraftRC *release.MatchedRelease
}

func (a *App) DiscoverComponentVersions(ctx context.Context, componentIDs []string) ([]ComponentVersionDiscovery, error) {
	if len(componentIDs) == 0 {
		componentIDs = a.registry.IDs()
	}
	releaseCache := make(map[string][]ghcli.Release)
	results := make([]ComponentVersionDiscovery, 0, len(componentIDs))
	for _, id := range componentIDs {
		spec, err := a.componentSpec(id)
		if err != nil {
			return nil, err
		}
		cacheKey := spec.GitHubOwner + "/" + spec.GitHubRepo
		remoteReleases, ok := releaseCache[cacheKey]
		if !ok {
			remoteReleases, err = a.gh.ListReleases(ctx, spec.GitHubOwner, spec.GitHubRepo)
			if err != nil {
				return nil, fmt.Errorf("discover releases for %s: %w", id, err)
			}
			releaseCache[cacheKey] = remoteReleases
		}
		discovered, err := release.DiscoverLatestVersions(spec, remoteReleases)
		if err != nil {
			return nil, fmt.Errorf("discover component versions for %s: %w", id, err)
		}
		results = append(results, ComponentVersionDiscovery{
			Component:     spec,
			LatestRelease: discovered.LatestRelease,
			LatestRC:      discovered.LatestRC,
			LatestDraftRC: discovered.LatestDraftRC,
		})
	}
	return results, nil
}

func (a *App) ConfiguredComponentSpec(id string) (components.ComponentSpec, error) {
	return a.componentSpec(id)
}

func (a *App) componentSpec(id string) (components.ComponentSpec, error) {
	spec, err := a.registry.Get(id)
	if err != nil {
		return components.ComponentSpec{}, err
	}
	if spec.ID == "op-geth" {
		if a.Config.OpGeth.Owner != "" {
			spec.GitHubOwner = a.Config.OpGeth.Owner
		}
		if a.Config.OpGeth.Repo != "" {
			spec.GitHubRepo = a.Config.OpGeth.Repo
		}
		return spec, nil
	}
	if a.Config.GitHub.Owner != "" {
		spec.GitHubOwner = a.Config.GitHub.Owner
	}
	if a.Config.GitHub.Repo != "" {
		spec.GitHubRepo = a.Config.GitHub.Repo
	}
	if a.Config.BaseBranch != "" {
		spec.BaseBranch = a.Config.BaseBranch
	}
	return spec, nil
}
