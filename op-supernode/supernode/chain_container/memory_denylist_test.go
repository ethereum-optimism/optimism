package chain_container

import (
	"sync"

	"github.com/ethereum/go-ethereum/common"
)

// MemoryDenyList is an in-memory implementation of the DenyList for testing.
// It has the same semantics as the bbolt-backed DenyList but avoids disk I/O.
type MemoryDenyList struct {
	mu   sync.RWMutex
	data map[uint64]map[common.Hash]struct{}
}

func NewMemoryDenyList() *MemoryDenyList {
	return &MemoryDenyList{
		data: make(map[uint64]map[common.Hash]struct{}),
	}
}

func (m *MemoryDenyList) Add(height uint64, payloadHash common.Hash) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.data[height] == nil {
		m.data[height] = make(map[common.Hash]struct{})
	}
	m.data[height][payloadHash] = struct{}{}
	return nil
}

func (m *MemoryDenyList) Contains(height uint64, payloadHash common.Hash) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.data[height] == nil {
		return false, nil
	}
	_, found := m.data[height][payloadHash]
	return found, nil
}

func (m *MemoryDenyList) GetDeniedHashes(height uint64) ([]common.Hash, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	hashes := m.data[height]
	if len(hashes) == 0 {
		return nil, nil
	}
	result := make([]common.Hash, 0, len(hashes))
	for h := range hashes {
		result = append(result, h)
	}
	return result, nil
}

func (m *MemoryDenyList) Close() error {
	return nil
}