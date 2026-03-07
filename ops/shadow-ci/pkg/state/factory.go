package state

import (
	"fmt"
	"os"

	"github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/model"
)

// FromConfig builds the appropriate Store from platform config.
// Falls back to local store if backend is empty or unrecognized.
func FromConfig(cfg model.StateStoreConfig, branch string) (Store, error) {
	switch cfg.Backend {
	case "circleci":
		token := os.Getenv("CIRCLE_TOKEN")
		if token == "" {
			// Fall back to local store when token isn't available.
			// State won't persist across pipelines but everything else works.
			fmt.Fprintf(os.Stderr, "warning: CIRCLE_TOKEN not set, falling back to local state store\n")
			dir := cfg.Local.Dir
			if dir == "" {
				dir = "/tmp/shadow-ci/state"
			}
			return NewLocalStore(dir), nil
		}
		slug := os.Getenv("CIRCLE_PROJECT_SLUG")
		if slug == "" {
			slug = "gh/ethereum-optimism/optimism"
		}

		artifactsDir := cfg.CircleCI.ArtifactsDir
		if artifactsDir == "" {
			artifactsDir = "/tmp/shadow-ci/artifacts"
		}

		return NewCircleCIStore(CircleCIStoreConfig{
			Token:        token,
			ProjectSlug:  slug,
			Branch:       branch,
			Workflow:     "shadow-ci",
			ArtifactsDir: artifactsDir,
			StatePrefix:  cfg.CircleCI.StatePrefix,
		}), nil

	case "upstash":
		url := os.Getenv("UPSTASH_REDIS_REST_URL")
		token := os.Getenv("UPSTASH_REDIS_REST_TOKEN")
		if url == "" || token == "" {
			fmt.Fprintf(os.Stderr, "warning: UPSTASH_REDIS_REST_URL/TOKEN not set, falling back to local state store\n")
			dir := cfg.Local.Dir
			if dir == "" {
				dir = "/tmp/shadow-ci/state"
			}
			return NewLocalStore(dir), nil
		}
		return NewUpstashStore(url, token, "shadow-ci:"), nil

	case "local", "":
		dir := cfg.Local.Dir
		if dir == "" {
			dir = "/tmp/shadow-ci/state"
		}
		return NewLocalStore(dir), nil

	default:
		return nil, fmt.Errorf("unknown state backend: %q", cfg.Backend)
	}
}
