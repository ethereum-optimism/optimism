package config

import "context"

// Loader specifies how to load a faucets config file
type Loader interface {
	Load(ctx context.Context) (*Config, error)
}
