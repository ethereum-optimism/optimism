package config

import "context"

// Loader specifies how to load a sync test config file
type Loader interface {
	Load(ctx context.Context) (*Config, error)
}
