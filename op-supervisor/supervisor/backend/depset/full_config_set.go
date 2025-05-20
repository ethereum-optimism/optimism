package depset

import (
	"context"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type FullConfigSet interface {
	RollupConfigSet
	DependencySet
}

type FullConfigSetMerged struct {
	RollupConfigSet
	DependencySet
}

func (f FullConfigSetMerged) HasChain(chainID eth.ChainID) bool {
	// Any is ok, since the FullConfigSetMerged constructor checks that the two sets contain the same chains.
	return f.DependencySet.HasChain(chainID)
}

func (f FullConfigSetMerged) Chains() []eth.ChainID {
	// Any is ok, since the FullConfigSetMerged constructor checks that the two sets contain the same chains.
	return f.DependencySet.Chains()
}

func (f FullConfigSetMerged) LoadFullConfigSet(_ context.Context) (FullConfigSet, error) {
	return f, f.CheckChains()
}

// NewFullConfigSetMerged creates a new FullConfigSetMerged from a RollupConfigSet and a DependencySet.
// It checks that the two sets contain the same chains.
func NewFullConfigSetMerged(rollupConfigSet RollupConfigSet, dependencySet DependencySet) (FullConfigSetMerged, error) {
	f := FullConfigSetMerged{
		RollupConfigSet: rollupConfigSet,
		DependencySet:   dependencySet,
	}
	return f, f.CheckChains()
}

func (f FullConfigSetMerged) CheckChains() error {
	rollupChains := make(map[eth.ChainID]struct{})
	for _, chainID := range f.RollupConfigSet.Chains() {
		rollupChains[chainID] = struct{}{}
	}

	dependencyChains := make(map[eth.ChainID]struct{})
	for _, chainID := range f.DependencySet.Chains() {
		dependencyChains[chainID] = struct{}{}
	}

	// Check that both sets contain the same chains
	for chainID := range rollupChains {
		if _, ok := dependencyChains[chainID]; !ok {
			return fmt.Errorf("chain %s in rollup config set but not in dependency set", chainID)
		}
	}
	for chainID := range dependencyChains {
		if _, ok := rollupChains[chainID]; !ok {
			return fmt.Errorf("chain %s in dependency set but not in rollup config set", chainID)
		}
	}
	return nil
}

type FullConfigSetSource interface {
	LoadFullConfigSet(ctx context.Context) (FullConfigSet, error)
}

type FullConfigSetSourceMerged struct {
	RollupConfigSetSource
	DependencySetSource
}

func (l *FullConfigSetSourceMerged) LoadFullConfigSet(ctx context.Context) (FullConfigSet, error) {
	rollupConfigSet, err := l.RollupConfigSetSource.LoadRollupConfigSet(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load rollup config set: %w", err)
	}
	dependencySet, err := l.DependencySetSource.LoadDependencySet(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load dependency set: %w", err)
	}
	return NewFullConfigSetMerged(rollupConfigSet, dependencySet)
}
