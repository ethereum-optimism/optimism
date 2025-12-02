package depset

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

type RollupConfigSetSource interface {
	LoadRollupConfigSet(ctx context.Context) (RollupConfigSet, error)
}

// RollupConfigSet provides access to minimal rollup configuration for a set of chains.
// Implementations should panic if any of the methods, besides HasChain, are called for a chain
// that is not part of the rollup config set.
type RollupConfigSet interface {
	// HasChain returns true if the chain is part of the rollup config set.
	HasChain(chainID eth.ChainID) bool

	// Chains returns the list of chains in the rollup config set.
	Chains() []eth.ChainID

	// Genesis returns the genesis configuration for the given chain.
	// It panics if the chain is not part of the rollup config set.
	// Use HasChain first to check if the chain is part of the rollup config set if
	// guarantee of existence isn't provided by the caller context.
	Genesis(chainID eth.ChainID) Genesis

	ActivationConfig
}

type ActivationConfig interface {
	// IsInterop returns true if the Interop hardfork is active for the given chain at the given timestamp.
	// It panics if the chain is not part of the rollup config set.
	// Use HasChain first to check if the chain is part of the rollup config set if
	// guarantee of existence isn't provided by the caller context.
	IsInterop(chainID eth.ChainID, ts uint64) bool

	// IsInteropActivationBlock returns true if the given timestamp is for an Interop activation block.
	// It panics if the chain is not part of the rollup config set.
	// Use HasChain first to check if the chain is part of the rollup config set if
	// guarantee of existence isn't provided by the caller context.
	IsInteropActivationBlock(chainID eth.ChainID, ts uint64) bool
}

type StaticRollupConfigSet map[eth.ChainID]*StaticRollupConfig

// StaticRollupConfig provides the rollup information relevant for Interop.
// It's a trimmed down version of [rollup.Config].
type StaticRollupConfig struct {
	// Genesis anchor point of the rollup
	Genesis Genesis `json:"genesis"`

	// Seconds per L2 block
	BlockTime uint64 `json:"block_time"`

	// InteropTime sets the activation time for the Interop network upgrade.
	// Active if InteropTime != nil && L2 block timestamp >= *InteropTime, inactive otherwise.
	InteropTime *uint64 `json:"interop_time,omitempty"`
}

// Genesis provides the genesis information relevant for Interop.
// It's a trimmed down version of [rollup.Genesis].
type Genesis struct {
	// The L1 block that the rollup starts *after* (no derived transactions)
	L1 types.BlockSeal `json:"l1"`
	// The L2 block the rollup starts from (no transactions, pre-configured state, no parent)
	L2 types.BlockSeal `json:"l2"`
}

func (c *StaticRollupConfigSet) LoadRollupConfigSet(ctx context.Context) (RollupConfigSet, error) {
	return c, nil
}

var (
	_ RollupConfigSetSource = (*StaticRollupConfigSet)(nil)
	_ RollupConfigSet       = (*StaticRollupConfigSet)(nil)
)

func GenesisFromRollupGenesis(genesis *rollup.Genesis, l1Time uint64) Genesis {
	return Genesis{
		L1: types.BlockSeal{
			Hash:      genesis.L1.Hash,
			Number:    genesis.L1.Number,
			Timestamp: l1Time,
		},
		L2: types.BlockSeal{
			Hash:      genesis.L2.Hash,
			Number:    genesis.L2.Number,
			Timestamp: genesis.L2Time,
		},
	}
}

func StaticRollupConfigFromRollupConfig(cfg *rollup.Config, l1Time uint64) *StaticRollupConfig {
	return &StaticRollupConfig{
		Genesis:     GenesisFromRollupGenesis(&cfg.Genesis, l1Time),
		BlockTime:   cfg.BlockTime,
		InteropTime: cfg.InteropTime,
	}
}

// IsInterop returns true if the Interop hardfork is active at or past the given timestamp.
func (c *StaticRollupConfig) IsInterop(ts uint64) bool {
	return c.InteropTime != nil && ts >= *c.InteropTime
}

func (c *StaticRollupConfig) IsInteropActivationBlock(ts uint64) bool {
	return c.IsInterop(ts) &&
		ts >= c.BlockTime &&
		!c.IsInterop(ts-c.BlockTime)
}

func NewStaticRollupConfigSet(cfgs map[eth.ChainID]*StaticRollupConfig) StaticRollupConfigSet {
	return cfgs
}

type ChainTimestamper interface {
	Timestamp(id eth.ChainID) uint64
}

type StaticTimestamp uint64

func (ts StaticTimestamp) Timestamp(eth.ChainID) uint64 { return uint64(ts) }

func StaticRollupConfigSetFromRollupConfigMap(rcfgs map[eth.ChainID]*rollup.Config, l1Timestamps ChainTimestamper) StaticRollupConfigSet {
	cfgs := make(map[eth.ChainID]*StaticRollupConfig, len(rcfgs))
	for id, cfg := range rcfgs {
		cfgs[id] = StaticRollupConfigFromRollupConfig(cfg, l1Timestamps.Timestamp(id))
	}
	return NewStaticRollupConfigSet(cfgs)
}

// HasChain returns true if the chain is part of the rollup config set.
func (s StaticRollupConfigSet) HasChain(chainID eth.ChainID) bool {
	_, ok := s[chainID]
	return ok
}

// Chains returns the list of chains in the rollup config set.
func (s StaticRollupConfigSet) Chains() []eth.ChainID {
	ids := make([]eth.ChainID, 0, len(s))
	for id := range s {
		ids = append(ids, id)
	}
	return ids
}

// Genesis returns the genesis configuration for the given chain.
// Panics if the chain is not part of the rollup config set.
func (s StaticRollupConfigSet) Genesis(chainID eth.ChainID) Genesis {
	cfg, ok := s[chainID]
	if !ok {
		panic("chain not found in rollup config set")
	}
	return cfg.Genesis
}

// IsInterop returns true if the Interop hardfork is active for the given chain at the given timestamp.
// Panics if the chain is not part of the rollup config set.
func (s StaticRollupConfigSet) IsInterop(chainID eth.ChainID, ts uint64) bool {
	cfg, ok := s[chainID]
	if !ok {
		panic("chain not found in rollup config set")
	}
	return cfg.IsInterop(ts)
}

// IsInteropActivationBlock returns true if the given timestamp is the activation block for the Interop hardfork.
// Panics if the chain is not part of the rollup config set.
func (s StaticRollupConfigSet) IsInteropActivationBlock(chainID eth.ChainID, ts uint64) bool {
	cfg, ok := s[chainID]
	if !ok {
		panic("chain not found in rollup config set")
	}
	return cfg.IsInteropActivationBlock(ts)
}
