package config

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-service/endpoint"
	"github.com/ethereum-optimism/optimism/op-service/eth"

	sttypes "github.com/ethereum-optimism/optimism/op-sync-tester/synctester/backend/types"
)

type SyncTesterEntry struct {
	ELRPC endpoint.MustRPC `yaml:"el_rpc"`

	// ChainID is used to sanity-check we are connected to the right chain,
	// and never accidentally try to use a different chain for sync tester work.
	Cfg EntryCfg `yaml:"cfg"`
}

type EntryCfg struct {
	ChainID eth.ChainID      `yaml:"chain_id"`
	Target  sttypes.FCUState `yaml:"target"`
}
type Config struct {
	// SyncTesters lists all sync testers by ID
	SyncTesters map[sttypes.SyncTesterID]*SyncTesterEntry `yaml:"synctesters,omitempty"`
}

var _ Loader = (*Config)(nil)

// Load is implemented on the Config itself,
// so that a static already-instantiated config can be used for in-process service setup,
// to bypass the YAML loading.
func (c *Config) Load(ctx context.Context) (*Config, error) {
	return c, nil
}
