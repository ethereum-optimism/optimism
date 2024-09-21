package heads

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-service/ioutil"
	"github.com/ethereum-optimism/optimism/op-service/jsonutil"

	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

// HeadTracker records the current chain head pointers for a single chain.
type HeadTracker struct {
	rwLock sync.RWMutex

	path string

	current *Heads

	logger log.Logger
}

func (t *HeadTracker) CrossUnsafe(id types.ChainID) HeadPointer {
	return t.current.Get(id).CrossUnsafe
}

func (t *HeadTracker) CrossSafe(id types.ChainID) HeadPointer {
	return t.current.Get(id).CrossSafe
}

func (t *HeadTracker) CrossFinalized(id types.ChainID) HeadPointer {
	return t.current.Get(id).CrossFinalized
}

func (t *HeadTracker) LocalUnsafe(id types.ChainID) HeadPointer {
	return t.current.Get(id).Unsafe
}

func (t *HeadTracker) LocalSafe(id types.ChainID) HeadPointer {
	return t.current.Get(id).LocalSafe
}

func (t *HeadTracker) LocalFinalized(id types.ChainID) HeadPointer {
	return t.current.Get(id).LocalFinalized
}

func (t *HeadTracker) UpdateCrossUnsafe(id types.ChainID, pointer HeadPointer) error {
	return t.Apply(OperationFn(func(heads *Heads) error {
		t.logger.Info("Cross-unsafe update", "pointer", pointer)
		h := heads.Get(id)
		h.CrossUnsafe = pointer
		heads.Put(id, h)
		return nil
	}))
}

func (t *HeadTracker) UpdateCrossSafe(id types.ChainID, pointer HeadPointer) error {
	return t.Apply(OperationFn(func(heads *Heads) error {
		t.logger.Info("Cross-safe update", "pointer", pointer)
		h := heads.Get(id)
		h.CrossSafe = pointer
		heads.Put(id, h)
		return nil
	}))
}

func (t *HeadTracker) UpdateCrossFinalized(id types.ChainID, pointer HeadPointer) error {
	return t.Apply(OperationFn(func(heads *Heads) error {
		t.logger.Info("Cross-finalized update", "pointer", pointer)
		h := heads.Get(id)
		h.CrossFinalized = pointer
		heads.Put(id, h)
		return nil
	}))
}

func (t *HeadTracker) UpdateLocalUnsafe(id types.ChainID, pointer HeadPointer) error {
	return t.Apply(OperationFn(func(heads *Heads) error {
		t.logger.Info("Local-unsafe update", "pointer", pointer)
		h := heads.Get(id)
		h.Unsafe = pointer
		heads.Put(id, h)
		return nil
	}))
}

func (t *HeadTracker) UpdateLocalSafe(id types.ChainID, pointer HeadPointer) error {
	return t.Apply(OperationFn(func(heads *Heads) error {
		t.logger.Info("Local-safe update", "pointer", pointer)
		h := heads.Get(id)
		h.LocalSafe = pointer
		heads.Put(id, h)
		return nil
	}))
}

func (t *HeadTracker) UpdateLocalFinalized(id types.ChainID, pointer HeadPointer) error {
	return t.Apply(OperationFn(func(heads *Heads) error {
		t.logger.Info("Local-finalized update", "pointer", pointer)
		h := heads.Get(id)
		h.LocalFinalized = pointer
		heads.Put(id, h)
		return nil
	}))
}

func NewHeadTracker(logger log.Logger, path string) (*HeadTracker, error) {
	current := NewHeads()
	if data, err := os.ReadFile(path); errors.Is(err, os.ErrNotExist) {
		// No existing file, just use empty heads
	} else if err != nil {
		return nil, fmt.Errorf("failed to read existing heads from %v: %w", path, err)
	} else {
		if err := json.Unmarshal(data, current); err != nil {
			return nil, fmt.Errorf("invalid existing heads file %v: %w", path, err)
		}
	}
	return &HeadTracker{
		path:    path,
		current: current,
		logger:  logger,
	}, nil
}

func (t *HeadTracker) Apply(op Operation) error {
	t.rwLock.Lock()
	defer t.rwLock.Unlock()
	// Store a copy of the heads prior to changing so we can roll back if needed.
	modified := t.current.Copy()
	if err := op.Apply(modified); err != nil {
		return fmt.Errorf("operation failed: %w", err)
	}
	if err := t.write(modified); err != nil {
		return fmt.Errorf("failed to store updated heads: %w", err)
	}
	t.current = modified
	return nil
}

func (t *HeadTracker) Current() *Heads {
	t.rwLock.RLock()
	defer t.rwLock.RUnlock()
	return t.current.Copy()
}

func (t *HeadTracker) write(heads *Heads) error {
	if err := jsonutil.WriteJSON(heads, ioutil.ToAtomicFile(t.path, 0o644)); err != nil {
		return fmt.Errorf("failed to write new heads: %w", err)
	}
	return nil
}

func (t *HeadTracker) Close() error {
	return nil
}
