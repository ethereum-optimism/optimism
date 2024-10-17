package depset

import (
	"context"
	"sort"

	"golang.org/x/exp/maps"

	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

type StaticConfigDependency struct {
	// ActivationTime is when the chain becomes part of the dependency set.
	// This is the minimum timestamp of the inclusion of an executing message.
	ActivationTime uint64 `json:"activationTime"`

	// HistoryMinTime is what the lower bound of data is to store.
	// This is the minimum timestamp of an initiating message to be accessible to others.
	// This is set to 0 when all data since genesis is executable.
	HistoryMinTime uint64 `json:"historyMinTime"`
}

// StaticConfigDependencySet statically declares a DependencySet.
// It can be used as a DependencySetSource itself, by simply returning the itself when loading the set.
type StaticConfigDependencySet struct {
	Dependencies map[types.ChainID]*StaticConfigDependency `json:"dependencies"`
}

var _ DependencySetSource = (*StaticConfigDependencySet)(nil)

var _ DependencySet = (*StaticConfigDependencySet)(nil)

func (ds *StaticConfigDependencySet) LoadDependencySet(ctx context.Context) (DependencySet, error) {
	return ds, nil
}

func (ds *StaticConfigDependencySet) CanExecuteAt(chainID types.ChainID, execTimestamp uint64) (bool, error) {
	dep, ok := ds.Dependencies[chainID]
	if !ok {
		return false, nil
	}
	return execTimestamp >= dep.ActivationTime, nil
}

func (ds *StaticConfigDependencySet) CanInitiateAt(chainID types.ChainID, initTimestamp uint64) (bool, error) {
	dep, ok := ds.Dependencies[chainID]
	if !ok {
		return false, nil
	}
	return initTimestamp >= dep.HistoryMinTime, nil
}

func (ds *StaticConfigDependencySet) Chains() []types.ChainID {
	out := maps.Keys(ds.Dependencies)
	sort.Slice(out, func(i, j int) bool {
		return out[i].Cmp(out[j]) < 0
	})
	return out
}

func (ds *StaticConfigDependencySet) HasChain(chainID types.ChainID) bool {
	_, ok := ds.Dependencies[chainID]
	return ok
}
