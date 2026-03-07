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

	return cfg, nil
}

func loadYAMLFile(path string, v any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(data, v)
}
