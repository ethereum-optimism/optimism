package chain_index_mapping

import (
	"fmt"
	"math/big"
	"sync"

	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

var (
	ErrChainIndexNotFound = fmt.Errorf("no index found for chain ID")
	ErrChainIDNotFound    = fmt.Errorf("no chain ID found for index")
)

// Map provides a bidirectional mapping between ChainID and ChainIndex.
type Map struct {
	idToIndex map[types.ChainID]types.ChainIndex
	indexToID map[types.ChainIndex]types.ChainID
	mu        sync.RWMutex
}

// New creates a new Map.
func New() *Map {
	return &Map{
		idToIndex: make(map[types.ChainID]types.ChainIndex),
		indexToID: make(map[types.ChainIndex]types.ChainID),
	}
}

// NewFromIDs creates a new Map from a slice of *big.Int.
// The slice index is used as the ChainIndex, and the *big.Int value is used as the ChainID.
func NewFromIDs(chainIDs []*big.Int) *Map {
	m := New()
	for index, id := range chainIDs {
		chainID := types.ChainIDFromBig(id)
		chainIndex := types.ChainIndex(index)
		m.Add(chainID, chainIndex)
	}
	return m
}

// Add adds a new mapping between a ChainID and a ChainIndex.
func (m *Map) Add(id types.ChainID, index types.ChainIndex) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.idToIndex[id] = index
	m.indexToID[index] = id
}

// GetIndex returns the ChainIndex for a given ChainID.
func (m *Map) GetIndex(id types.ChainID) (types.ChainIndex, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	index, ok := m.idToIndex[id]
	if !ok {
		return 0, ErrChainIndexNotFound
	}
	return index, nil
}

// GetID returns the ChainID for a given ChainIndex.
func (m *Map) GetID(index types.ChainIndex) (types.ChainID, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	id, ok := m.indexToID[index]
	if !ok {
		return types.ChainID{}, ErrChainIDNotFound
	}
	return id, nil
}
