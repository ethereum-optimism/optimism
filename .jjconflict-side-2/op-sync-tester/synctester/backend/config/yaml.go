package config

import (
	"bytes"
	"context"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// YamlLoader is a Loader that loads a sync tester configuration from a YAML file path.
type YamlLoader struct {
	Path string
}

var _ Loader = (*YamlLoader)(nil)

func (l *YamlLoader) Load(ctx context.Context) (*Config, error) {
	data, err := os.ReadFile(l.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}
	var out Config
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true) // ensure all the fields are known. Config correctness is critical.
	if err := dec.Decode(&out); err != nil {
		return nil, fmt.Errorf("failed to parse config YAML: %w", err)
	}
	return &out, nil
}
