package depset

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type RollupConfigSetSource interface {
	LoadRollupConfigSet(ctx context.Context) (RollupConfigSet, error)
}

// RollupConfigSet provides access to minimal rollup configuration for a set of chains.
// Implementations should panic if any of the methods, besides HasChain, are called for a chain
// that is not part of the rollup config set.
type RollupConfigSet interface {
	HasChain(chainID eth.ChainID) bool
	Chains() []eth.ChainID
	Genesis(chainID eth.ChainID) Genesis
	IsInterop(chainID eth.ChainID, ts uint64) bool
	IsInteropActivationBlock(chainID eth.ChainID, ts uint64) bool
}

type StaticRollupConfigSet struct {
	cfgs map[eth.ChainID]*StaticRollupConfig
}

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
	L1 eth.BlockID `json:"l1"`
	// The L2 block the rollup starts from (no transactions, pre-configured state)
	L2 eth.BlockID `json:"l2"`
	// Timestamp of L2 block
	L2Time uint64 `json:"l2_time"`
}

func (c *StaticRollupConfigSet) LoadRollupConfigSet(ctx context.Context) (RollupConfigSet, error) {
	return c, nil
}

var (
	_ RollupConfigSetSource = (*StaticRollupConfigSet)(nil)
	_ RollupConfigSet       = (*StaticRollupConfigSet)(nil)
)

// IsInterop returns true if the Interop hardfork is active at or past the given timestamp.
func (c *StaticRollupConfig) IsInterop(ts uint64) bool {
	return c.InteropTime != nil && ts >= *c.InteropTime
}

func (c *StaticRollupConfig) IsInteropActivationBlock(ts uint64) bool {
	return c.IsInterop(ts) &&
		ts >= c.BlockTime &&
		!c.IsInterop(ts-c.BlockTime)
}

func NewStaticRollupConfigSet(cfgs map[eth.ChainID]*StaticRollupConfig) *StaticRollupConfigSet {
	return &StaticRollupConfigSet{cfgs: cfgs}
}

// HasChain returns true if the chain is part of the rollup config set.
func (s *StaticRollupConfigSet) HasChain(chainID eth.ChainID) bool {
	_, ok := s.cfgs[chainID]
	return ok
}

// Chains returns the list of chains in the rollup config set.
func (s *StaticRollupConfigSet) Chains() []eth.ChainID {
	ids := make([]eth.ChainID, 0, len(s.cfgs))
	for id := range s.cfgs {
		ids = append(ids, id)
	}
	return ids
}

// Genesis returns the genesis configuration for the given chain.
// Panics if the chain is not part of the rollup config set.
func (s *StaticRollupConfigSet) Genesis(chainID eth.ChainID) Genesis {
	cfg, ok := s.cfgs[chainID]
	if !ok {
		panic("chain not found in rollup config set")
	}
	return cfg.Genesis
}

// IsInterop returns true if the Interop hardfork is active for the given chain at the given timestamp.
// Panics if the chain is not part of the rollup config set.
func (s *StaticRollupConfigSet) IsInterop(chainID eth.ChainID, ts uint64) bool {
	cfg, ok := s.cfgs[chainID]
	if !ok {
		panic("chain not found in rollup config set")
	}
	return cfg.IsInterop(ts)
}

// IsInteropActivationBlock returns true if the given timestamp is the activation block for the Interop hardfork.
// Panics if the chain is not part of the rollup config set.
func (s *StaticRollupConfigSet) IsInteropActivationBlock(chainID eth.ChainID, ts uint64) bool {
	cfg, ok := s.cfgs[chainID]
	if !ok {
		panic("chain not found in rollup config set")
	}
	return cfg.IsInteropActivationBlock(ts)
}
