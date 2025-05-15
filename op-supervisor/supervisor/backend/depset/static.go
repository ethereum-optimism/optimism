package depset

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"sort"

	"github.com/BurntSushi/toml"
	"github.com/ethereum-optimism/optimism/op-node/params"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

var errDuplicateChainIndex = errors.New("duplicate chain index")

type StaticConfigDependency struct {
	// ChainIndex is the unique short identifier of this chain.
	ChainIndex types.ChainIndex `json:"chainIndex" toml:"chain_index"`

	// ActivationTime is when the chain becomes part of the dependency set.
	// This is the minimum timestamp of the inclusion of an executing message.
	ActivationTime uint64 `json:"activationTime" toml:"activation_time"`

	// HistoryMinTime is what the lower bound of data is to store.
	// This is the minimum timestamp of an initiating message to be accessible to others.
	// This is set to 0 when all data since genesis is executable.
	HistoryMinTime uint64 `json:"historyMinTime" toml:"history_min_time"`
}

// StaticConfigDependencySet statically declares a DependencySet.
// It can be used as a DependencySetSource itself, by simply returning the itself when loading the set.
type StaticConfigDependencySet struct {
	// dependency info per chain
	dependencies map[eth.ChainID]*StaticConfigDependency
	// cached mapping of chain index to chain ID
	indexToID map[types.ChainIndex]eth.ChainID
	// cached list of chain IDs, sorted by ID value
	chainIDs []eth.ChainID
	// overrideMessageExpiryWindow is the message expiry window to use for this dependency set
	overrideMessageExpiryWindow uint64
}

func NewStaticConfigDependencySet(dependencies map[eth.ChainID]*StaticConfigDependency) (*StaticConfigDependencySet, error) {
	out := &StaticConfigDependencySet{dependencies: dependencies}
	if err := out.hydrate(); err != nil {
		return nil, err
	}
	return out, nil
}

// NewStaticConfigDependencySetWithMessageExpiryOverride creates a new StaticConfigDependencySet with a message expiry window override.
// To be used only for testing.
func NewStaticConfigDependencySetWithMessageExpiryOverride(dependencies map[eth.ChainID]*StaticConfigDependency, overrideMessageExpiryWindow uint64) (*StaticConfigDependencySet, error) {
	out := &StaticConfigDependencySet{dependencies: dependencies, overrideMessageExpiryWindow: overrideMessageExpiryWindow}
	if err := out.hydrate(); err != nil {
		return nil, err
	}
	return out, nil
}

// minStaticConfigDependencySet is a util for JSON/TOML encoding/decoding,
// for just the minimal set of attributes that matter,
// while wrapping the decoding functionality with additional hydration step.
type minStaticConfigDependencySet struct {
	Dependencies                map[string]*StaticConfigDependency `json:"dependencies" toml:"dependencies"`
	OverrideMessageExpiryWindow uint64                             `json:"overrideMessageExpiryWindow,omitempty" toml:"override_message_expiry_window,omitempty"`
}

func (ds *StaticConfigDependencySet) MarshalJSON() ([]byte, error) {
	// Convert map keys to strings
	stringMap := make(map[string]*StaticConfigDependency)
	for id, dep := range ds.dependencies {
		stringMap[id.String()] = dep
	}

	out := &minStaticConfigDependencySet{
		Dependencies:                stringMap,
		OverrideMessageExpiryWindow: ds.overrideMessageExpiryWindow,
	}
	return json.Marshal(out)
}

func (ds *StaticConfigDependencySet) UnmarshalJSON(data []byte) error {
	var v minStaticConfigDependencySet
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}

	// Convert string keys back to ChainID
	ds.dependencies = make(map[eth.ChainID]*StaticConfigDependency)
	for idStr, dep := range v.Dependencies {
		id, err := eth.ParseDecimalChainID(idStr)
		if err != nil {
			return fmt.Errorf("invalid chain ID in JSON: %w", err)
		}
		ds.dependencies[id] = dep
	}

	ds.overrideMessageExpiryWindow = v.OverrideMessageExpiryWindow
	return ds.hydrate()
}

func (ds *StaticConfigDependencySet) MarshalTOML() ([]byte, error) {
	// Convert map keys (ChainID) to strings so TOML can encode the map.
	stringMap := make(map[string]*StaticConfigDependency, len(ds.dependencies))
	for id, dep := range ds.dependencies {
		stringMap[id.String()] = dep
	}

	payload := &minStaticConfigDependencySet{
		Dependencies:                stringMap,
		OverrideMessageExpiryWindow: ds.overrideMessageExpiryWindow,
	}

	var buf bytes.Buffer
	if err := toml.NewEncoder(&buf).Encode(payload); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (ds *StaticConfigDependencySet) UnmarshalTOML(v interface{}) error {
	var buf bytes.Buffer
	if err := toml.NewEncoder(&buf).Encode(v); err != nil {
		return fmt.Errorf("re-encoding TOML: %w", err)
	}

	// Decode into the minimal helper struct that has the right tags.
	var tmp minStaticConfigDependencySet
	if _, err := toml.Decode(buf.String(), &tmp); err != nil {
		return fmt.Errorf("decoding into helper struct: %w", err)
	}

	// Convert string keys back to ChainID and copy the data.
	ds.dependencies = make(map[eth.ChainID]*StaticConfigDependency, len(tmp.Dependencies))
	for idStr, dep := range tmp.Dependencies {
		id, err := eth.ParseDecimalChainID(idStr)
		if err != nil {
			return fmt.Errorf("invalid chain ID %q: %w", idStr, err)
		}
		ds.dependencies[id] = dep
	}

	ds.overrideMessageExpiryWindow = tmp.OverrideMessageExpiryWindow
	return ds.hydrate()
}

// hydrate sets all the cached values, based on the dependencies attribute
func (ds *StaticConfigDependencySet) hydrate() error {
	ds.indexToID = make(map[types.ChainIndex]eth.ChainID)
	ds.chainIDs = make([]eth.ChainID, 0, len(ds.dependencies))
	for id, dep := range ds.dependencies {
		if existing, ok := ds.indexToID[dep.ChainIndex]; ok {
			return fmt.Errorf("%w: chain %s cannot have the same index (%d) as chain %s", errDuplicateChainIndex, id, dep.ChainIndex, existing)
		}
		ds.indexToID[dep.ChainIndex] = id
		ds.chainIDs = append(ds.chainIDs, id)
	}
	sort.Slice(ds.chainIDs, func(i, j int) bool {
		return ds.chainIDs[i].Cmp(ds.chainIDs[j]) < 0
	})
	return nil
}

var _ DependencySetSource = (*StaticConfigDependencySet)(nil)

var _ DependencySet = (*StaticConfigDependencySet)(nil)

func (ds *StaticConfigDependencySet) LoadDependencySet(ctx context.Context) (DependencySet, error) {
	return ds, nil
}

func (ds *StaticConfigDependencySet) CanExecuteAt(chainID eth.ChainID, execTimestamp uint64) (bool, error) {
	dep, ok := ds.dependencies[chainID]
	if !ok {
		return false, nil
	}
	return execTimestamp >= dep.ActivationTime, nil
}

func (ds *StaticConfigDependencySet) CanInitiateAt(chainID eth.ChainID, initTimestamp uint64) (bool, error) {
	dep, ok := ds.dependencies[chainID]
	if !ok {
		return false, nil
	}
	return initTimestamp >= dep.HistoryMinTime, nil
}

func (ds *StaticConfigDependencySet) Chains() []eth.ChainID {
	return slices.Clone(ds.chainIDs)
}

func (ds *StaticConfigDependencySet) HasChain(chainID eth.ChainID) bool {
	_, ok := ds.dependencies[chainID]
	return ok
}

func (ds *StaticConfigDependencySet) ChainIndexFromID(id eth.ChainID) (types.ChainIndex, error) {
	dep, ok := ds.dependencies[id]
	if !ok {
		return 0, fmt.Errorf("failed to translate chain ID %s to chain index: %w", id, types.ErrUnknownChain)
	}
	return dep.ChainIndex, nil
}

func (ds *StaticConfigDependencySet) ChainIDFromIndex(index types.ChainIndex) (eth.ChainID, error) {
	id, ok := ds.indexToID[index]
	if !ok {
		return eth.ChainID{}, fmt.Errorf("failed to translate chain index %s to chain ID: %w", index, types.ErrUnknownChain)
	}
	return id, nil
}

func (ds *StaticConfigDependencySet) MessageExpiryWindow() uint64 {
	if ds.overrideMessageExpiryWindow == 0 {
		return params.MessageExpiryTimeSecondsInterop
	}
	return ds.overrideMessageExpiryWindow
}

// Dependencies returns a deep copy of the dependencies map
func (ds *StaticConfigDependencySet) Dependencies() map[eth.ChainID]*StaticConfigDependency {
	copied := make(map[eth.ChainID]*StaticConfigDependency, len(ds.dependencies))
	for chainId, dep := range ds.dependencies {
		copied[chainId] = &StaticConfigDependency{
			ChainIndex:     dep.ChainIndex,
			ActivationTime: dep.ActivationTime,
			HistoryMinTime: dep.HistoryMinTime,
		}
	}
	return copied
}
