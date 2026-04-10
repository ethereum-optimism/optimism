package release

import (
	"fmt"
	"strings"

	"github.com/ethereum-optimism/optimism/oprm/components"
	"github.com/ethereum-optimism/optimism/oprm/providers/ghcli"
)

type MatchedRelease struct {
	TagName    string
	Version    string
	URL        string
	Draft      bool
	PreRelease bool
}

type DiscoveredVersions struct {
	LatestRelease *MatchedRelease
	LatestRC      *MatchedRelease
	LatestDraftRC *MatchedRelease
}

func ExtractComponentVersion(spec components.ComponentSpec, tag string) (string, bool) {
	if strings.Contains(tag, "/") {
		prefix := spec.TagPrefix + "/"
		if !strings.HasPrefix(tag, prefix) {
			return "", false
		}
		return strings.TrimPrefix(tag, prefix), true
	}
	if spec.Kind == components.KindExternalGo && strings.HasPrefix(tag, spec.TagPrefix) {
		return tag, true
	}
	return "", false
}

func DiscoverLatestVersions(spec components.ComponentSpec, releases []ghcli.Release) (*DiscoveredVersions, error) {
	out := &DiscoveredVersions{}
	var latestRelease Version
	var latestRC Version
	var latestDraftRC Version

	for _, rel := range releases {
		versionText, ok := ExtractComponentVersion(spec, rel.TagName)
		if !ok {
			continue
		}
		version, err := ParseVersion(versionText)
		if err != nil {
			return nil, fmt.Errorf("parse tag %q for component %q: %w", rel.TagName, spec.ID, err)
		}
		if version.IsStableRelease() {
			if rel.Draft {
				continue
			}
			if out.LatestRelease == nil || version.Compare(latestRelease) > 0 {
				latestRelease = version
				out.LatestRelease = &MatchedRelease{TagName: rel.TagName, Version: version.String(), URL: rel.URL, Draft: rel.Draft, PreRelease: rel.PreRelease}
			}
			continue
		}
		if version.IsRC() {
			if out.LatestRC == nil || version.Compare(latestRC) > 0 {
				latestRC = version
				out.LatestRC = &MatchedRelease{TagName: rel.TagName, Version: version.String(), URL: rel.URL, Draft: rel.Draft, PreRelease: rel.PreRelease}
			}
			if rel.Draft && (out.LatestDraftRC == nil || version.Compare(latestDraftRC) > 0) {
				latestDraftRC = version
				out.LatestDraftRC = &MatchedRelease{TagName: rel.TagName, Version: version.String(), URL: rel.URL, Draft: rel.Draft, PreRelease: rel.PreRelease}
			}
		}
	}

	return out, nil
}
