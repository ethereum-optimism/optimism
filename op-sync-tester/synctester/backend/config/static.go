package config

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-service/endpoint"
)

type Config struct {
	ELRPC endpoint.MustRPC `yaml:"el_rpc"`
}

var _ Loader = (*Config)(nil)

// Load is implemented on the Config itself,
// so that a static already-instantiated config can be used for in-process service setup,
// to bypass the YAML loading.
func (c *Config) Load(ctx context.Context) (*Config, error) {
	return c, nil
}
