package script

import (
	"golang.org/x/exp/maps"
	"path/filepath"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/op-chain-ops/foundry"
)

// State is an interface to represent cached/cacheable forge-allocs, keyed with a script-hash.
// The state may either be in memory or on disk.
// This interface prevents intermediate cached states from unnecessarily being loaded into memory.
type State interface {
	// ScriptHash identifies the origin of the state.
	// The initial empty state is identified by a zero hash.
	ScriptHash() common.Hash
	// Load loads a copy of the allocs.
	Load() (*foundry.ForgeAllocs, error)
}

// InMemoryState is a State that exists in memory.
type InMemoryState struct {
	scriptHash common.Hash
	allocs     *foundry.ForgeAllocs
	labels     map[common.Address]string
}

var _ State = (*InMemoryState)(nil)

func (s *InMemoryState) ScriptHash() common.Hash {
	return s.scriptHash
}

func (s *InMemoryState) Load() (*foundry.ForgeAllocs, error) {
	return s.allocs.Copy(), nil
}

func (s *InMemoryState) Labels() (map[common.Address]string, error) {
	return maps.Clone(s.labels), nil
}

// CachedState is a State that exists on disk.
type CachedState struct {
	dir        string
	scriptHash common.Hash
}

var _ State = (*CachedState)(nil)

func (c *CachedState) ScriptHash() common.Hash {
	return c.scriptHash
}

func (c *CachedState) Load() (*foundry.ForgeAllocs, error) {
	return foundry.LoadForgeAllocs(filepath.Join(c.dir, c.scriptHash.String()+".allocs.json"))
}
