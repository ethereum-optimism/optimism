package config

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-node/rollup/engine"
	"github.com/ethereum-optimism/optimism/op-service/endpoint"
	"github.com/ethereum-optimism/optimism/op-service/eth"

	sttypes "github.com/ethereum-optimism/optimism/op-sync-tester/synctester/backend/types"
)

type SyncTesterEntry struct {
	ELRPC endpoint.MustRPC `yaml:"el_rpc"`

	// ChainID is used to sanity-check we are connected to the right chain,
	// and never accidentally try to use a different chain for sync tester work.
	ChainID eth.ChainID `yaml:"chain_id"`

	// EngineKind specifies which EL implementation to mock (geth, reth, erigon)
	// This affects behavior around sync modes and engine API responses
	EngineKind engine.Kind `yaml:"engine_kind,omitempty"`

	// SyncMode specifies the sync mode to simulate (full, snap, etc.)
	SyncMode string `yaml:"sync_mode,omitempty"`

	// NetworkType specifies the network type (mainnet, sepolia, etc.)
	// This affects regenesis behavior and other network-specific characteristics
	NetworkType string `yaml:"network_type,omitempty"`
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
