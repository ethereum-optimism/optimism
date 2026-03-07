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
			return nil, fmt.Errorf("CIRCLE_TOKEN env var required for circleci state backend")
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
