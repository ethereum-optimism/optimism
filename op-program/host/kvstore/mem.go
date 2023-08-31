package kvstore

import (
	"sync"

	"github.com/ethereum/go-ethereum/common"
)

// MemKV implements the KV store interface in memory, backed by a regular Go map.
// This should only be used in testing, as large programs may require more pre-image data than available memory.
// MemKV is safe for concurrent use.
type MemKV struct {
	sync.RWMutex
	m map[common.Hash][]byte
}

var _ KV = (*MemKV)(nil)

func NewMemKV() *MemKV {
	return &MemKV{m: make(map[common.Hash][]byte)}
}

func (m *MemKV) Put(k common.Hash, v []byte) error {
	m.Lock()
	defer m.Unlock()
	m.m[k] = v
	return nil
}

func (m *MemKV) Get(k common.Hash) ([]byte, error) {
	m.RLock()
	defer m.RUnlock()
	v, ok := m.m[k]
	if !ok {
		return nil, ErrNotFound
	}
	return v, nil
}
