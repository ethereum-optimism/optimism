package model

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// LoadConfig loads the full configuration from a directory.
func LoadConfig(dir string) (*Config, error) {
	cfg := &Config{}

	if err := loadYAMLFile(filepath.Join(dir, "adapters.yaml"), &cfg.Adapters); err != nil {
		return nil, fmt.Errorf("adapters.yaml: %w", err)
	}

	if err := loadYAMLFile(filepath.Join(dir, "scoping.yaml"), &cfg.Scoping); err != nil {
		return nil, fmt.Errorf("scoping.yaml: %w", err)
	}

	if err := loadYAMLFile(filepath.Join(dir, "platform.yaml"), &cfg.Platform); err != nil {
		return nil, fmt.Errorf("platform.yaml: %w", err)
	}

	// Placement is optional — defaults to everything at PR stage.
	placementPath := filepath.Join(dir, "placement.yaml")
	if _, err := os.Stat(placementPath); err == nil {
		if err := loadYAMLFile(placementPath, &cfg.Placement); err != nil {
			return nil, fmt.Errorf("placement.yaml: %w", err)
		}
	} else {
		cfg.Placement.DefaultStage = StagePR
	}

	if err := validateConfig(cfg); err != nil {
		return nil, fmt.Errorf("config validation: %w", err)
	}

	return cfg, nil
}

// validateConfig checks for configuration errors that would cause runtime failures.
func validateConfig(cfg *Config) error {
	// Check for workspace_path overlaps within the same group.
	// Two categories sharing workspace_paths in the same group causes
	// race conditions during parallel cache restore.
	type pathOwner struct {
		category string
		group    string
	}
	seen := make(map[string]pathOwner)
	for name, cat := range cfg.Scoping.JobCategories {
		if cat.Group == "" {
			continue
		}
		for _, wp := range cat.WorkspacePaths {
			if existing, ok := seen[wp]; ok && existing.group == cat.Group {
				return fmt.Errorf(
					"workspace_path %q is declared by both %q and %q in group %q — "+
						"parallel restore will race. Remove it from one category or "+
						"make one depend on the other",
					wp, existing.category, name, cat.Group,
				)
			}
			seen[wp] = pathOwner{category: name, group: cat.Group}
		}
	}
	return nil
}

func loadYAMLFile(path string, v any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(data, v)
}
