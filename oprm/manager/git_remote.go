package manager

import (
	"context"
	"fmt"
	"sort"
	"strings"
)

func (a *App) configuredGitRemote(ctx context.Context, repoPath string, owner string, repo string) (string, error) {
	remotes, err := a.git.Remotes(ctx, repoPath)
	if err != nil {
		return "", err
	}
	expected := owner + "/" + repo
	names := make([]string, 0, len(remotes))
	for name := range remotes {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if githubRepoSlugFromRemoteURL(remotes[name]) == expected {
			return name, nil
		}
	}
	if len(names) == 0 {
		return "", fmt.Errorf("no git remotes found in %s; expected a remote for %s", repoPath, expected)
	}
	available := make([]string, 0, len(names))
	for _, name := range names {
		slug := githubRepoSlugFromRemoteURL(remotes[name])
		if slug == "" {
			slug = remotes[name]
		}
		available = append(available, fmt.Sprintf("%s=%s", name, slug))
	}
	return "", fmt.Errorf("no git remote in %s matches configured target %s; available remotes: %s", repoPath, expected, strings.Join(available, ", "))
}

func githubRepoSlugFromRemoteURL(remoteURL string) string {
	value := strings.TrimSpace(remoteURL)
	if value == "" {
		return ""
	}
	value = strings.TrimSuffix(value, ".git")
	value = strings.TrimPrefix(value, "ssh://")
	idx := strings.Index(value, "github.com")
	if idx < 0 {
		return ""
	}
	value = value[idx+len("github.com"):]
	value = strings.TrimPrefix(value, ":")
	value = strings.TrimPrefix(value, "/")
	parts := strings.Split(value, "/")
	if len(parts) < 2 {
		return ""
	}
	return parts[0] + "/" + parts[1]
}
