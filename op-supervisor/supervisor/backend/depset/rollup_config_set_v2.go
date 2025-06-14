package depset

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

type StaticRollupConfigSetV2 map[eth.ChainID]*rollup.Config

func (c StaticRollupConfigSetV2) RollupConfig(chainID eth.ChainID) *rollup.Config {
	return c[chainID]
}

func (c StaticRollupConfigSetV2) LoadRollupConfigSet(ctx context.Context) (RollupConfigSet, error) {
	return c, nil
}

func (c StaticRollupConfigSetV2) LoadRollupConfigSetV2(ctx context.Context) (RollupConfigSetV2, error) {
	return c, nil
}

var (
	_ RollupConfigSetSource = (StaticRollupConfigSetV2)(nil)
	_ RollupConfigSet       = (StaticRollupConfigSetV2)(nil)
)

func NewStaticRollupConfigSetV2(cfgs map[eth.ChainID]*rollup.Config) StaticRollupConfigSetV2 {
	return cfgs
}

// HasChain returns true if the chain is part of the rollup config set.
func (s StaticRollupConfigSetV2) HasChain(chainID eth.ChainID) bool {
	_, ok := s[chainID]
	return ok
}

// Chains returns the list of chains in the rollup config set.
func (s StaticRollupConfigSetV2) Chains() []eth.ChainID {
	ids := make([]eth.ChainID, 0, len(s))
	for id := range s {
		ids = append(ids, id)
	}
	return ids
}

// Genesis returns the genesis configuration for the given chain.
// Panics if the chain is not part of the rollup config set.
func (s StaticRollupConfigSetV2) Genesis(chainID eth.ChainID) Genesis {
	cfg, ok := s[chainID]
	if !ok {
		panic("chain not found in rollup config set")
	}
	if cfg.Genesis.L1Time == 0 {
		panic(fmt.Errorf("rollup config of chain %s was not prepared with L1 anchor block timestamp", chainID))
	}
	return Genesis{
		L1: types.BlockSeal{
			Hash:      cfg.Genesis.L1.Hash,
			Number:    cfg.Genesis.L1.Number,
			Timestamp: cfg.Genesis.L1Time,
		},
		L2: types.BlockSeal{
			Hash:      cfg.Genesis.L2.Hash,
			Number:    cfg.Genesis.L2.Number,
			Timestamp: cfg.Genesis.L2Time,
		},
	}
}

// IsInterop returns true if the Interop hardfork is active for the given chain at the given timestamp.
// Panics if the chain is not part of the rollup config set.
func (s StaticRollupConfigSetV2) IsInterop(chainID eth.ChainID, ts uint64) bool {
	cfg, ok := s[chainID]
	if !ok {
		panic("chain not found in rollup config set")
	}
	return cfg.IsInterop(ts)
}

// IsInteropActivationBlock returns true if the given timestamp is the activation block for the Interop hardfork.
// Panics if the chain is not part of the rollup config set.
func (s StaticRollupConfigSetV2) IsInteropActivationBlock(chainID eth.ChainID, ts uint64) bool {
	cfg, ok := s[chainID]
	if !ok {
		panic("chain not found in rollup config set")
	}
	return cfg.IsInteropActivationBlock(ts)
}

type RollupConfigSetSourceV2 interface {
	RollupConfigSetSource
	LoadRollupConfigSetV2(ctx context.Context) (RollupConfigSetV2, error)
}

type RollupConfigSetV2 interface {
	RollupConfigSet

	// RollupConfig the rollup-config of the given chain.
	// The user may mutate it to fix the missing L1 anchor timestamp.
	RollupConfig(chainID eth.ChainID) *rollup.Config
}

// JSONRollupConfigsLoaderV2 loads a set of op-node rollup.json configs into
// a V2 rollup config set, for use by op-node and op-supervisor code.
//
// The [PathPattern] is a glob pattern that matches the rollup.json files, e.g.
// "configs/rollup-*.json". See https://pkg.go.dev/path/filepath#Glob for more details.
//
// Unlike the V1 loader, this does not query L1 and does not patch the L1 timestamp.
// Instead, the user should patch it manually.
// Long-term we can make the L1 timestamp a part of the config, and not rely on dynamic fetching at all.
type JSONRollupConfigsLoaderV2 struct {
	PathPattern string
}

func (j *JSONRollupConfigsLoaderV2) LoadRollupConfigSet(ctx context.Context) (RollupConfigSet, error) {
	return j.LoadRollupConfigSetV2(ctx)
}

func (j *JSONRollupConfigsLoaderV2) LoadRollupConfigSetV2(ctx context.Context) (RollupConfigSetV2, error) {
	matches, err := filepath.Glob(j.PathPattern)
	if err != nil {
		return nil, fmt.Errorf("failed to glob files: %w", err)
	}
	if len(matches) == 0 {
		return nil, fmt.Errorf("cannot run with empty set of chains: %w", err)
	}

	configs := make(map[eth.ChainID]*rollup.Config)
	for _, path := range matches {
		cfg, err := LoadRollupCfg(path)
		if err != nil {
			return nil, err
		}
		chainID := eth.ChainIDFromBig(cfg.L2ChainID)
		configs[chainID] = cfg
	}
	return StaticRollupConfigSetV2(configs), nil
}

func LoadRollupCfg(path string) (*rollup.Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open rollup config %s: %w", path, err)
	}
	defer file.Close()

	var cfg rollup.Config
	if err = cfg.ParseRollupConfig(file); err != nil {
		return nil, fmt.Errorf("failed to parse rollup config %s: %w", path, err)
	}
	return &cfg, nil
}
