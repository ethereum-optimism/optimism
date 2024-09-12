package heads

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"

	"github.com/ethereum-optimism/optimism/op-service/ioutil"
	"github.com/ethereum-optimism/optimism/op-service/jsonutil"
)

// HeadTracker records the current chain head pointers for a single chain.
type HeadTracker struct {
	rwLock sync.RWMutex

	path string

	current *Heads
}

func NewHeadTracker(path string) (*HeadTracker, error) {
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
