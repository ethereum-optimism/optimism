package dsl

import (
	"fmt"
	"strings"

	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/retry"
	"github.com/stretchr/testify/require"
)

// L2Batcher wraps a stack.L2Batcher interface for DSL operations
type L2Batcher struct {
	commonImpl
	inner stack.L2Batcher
}

// NewL2Batcher creates a new L2Batcher DSL wrapper
func NewL2Batcher(inner stack.L2Batcher) *L2Batcher {
	return &L2Batcher{
		commonImpl: commonFromT(inner.T()),
		inner:      inner,
	}
}

func (b *L2Batcher) String() string {
	return b.inner.ID().String()
}

// Escape returns the underlying stack.L2Batcher
func (b *L2Batcher) Escape() stack.L2Batcher {
	return b.inner
}

func (b *L2Batcher) ActivityAPI() apis.BatcherActivity {
	return b.inner.ActivityAPI()
}

func (b *L2Batcher) Stop() {
	err := retry.Do0(b.ctx, 3, retry.Exponential(), func() error {
		err := b.Escape().ActivityAPI().StopBatcher(b.ctx)
		if err != nil && strings.Contains(err.Error(), "batcher is not running") {
			return nil
		}
		return err
	})
	require.NoError(b.t, err, fmt.Sprintf("Expected to be able to call StopBatcher API on chain %s, but got error", b.inner.ID().ChainID()))
}

func (b *L2Batcher) Start() {
	err := retry.Do0(b.ctx, 3, retry.Exponential(), func() error {
		return b.inner.ActivityAPI().StartBatcher(b.ctx)
	})
	require.NoError(b.t, err, fmt.Sprintf("Expected to be able to call StartBatcher API on chain %s, but got error", b.inner.ID().ChainID()))
}

// PauseAtBlock pauses the batcher at the specified block number.
// The batcher will process up to and including blockNum, but won't see any blocks beyond it.
// Returns the highest block number the batcher will see.
// This function is for integration test control only.
// Only works if the underlying batcher implements PausableBatcher.
func (b *L2Batcher) PauseAtBlock(blockNum uint64) uint64 {
	pausable, ok := b.inner.(stack.PausableBatcher)
	b.require.True(ok, "batcher does not implement PausableBatcher")
	pauseBlock := pausable.PauseAtBlock(blockNum)
	err := b.ActivityAPI().FlushBatcher(b.ctx)
	b.require.NoError(err, "Failed to flush batcher")
	return pauseBlock
}

// Unpause resumes normal batcher operation, allowing it to see all available blocks.
// This function is for integration test control only.
// Only works if the underlying batcher implements PausableBatcher.
func (b *L2Batcher) Unpause() {
	pausable, ok := b.inner.(stack.PausableBatcher)
	b.require.True(ok, "batcher does not implement PausableBatcher")
	pausable.Unpause()
}

// IsPaused returns true if the batcher is currently paused, and the block number it's paused at.
// Returns (false, 0) if not paused.
// This function is for integration test control only.
// Only works if the underlying batcher implements PausableBatcher.
func (b *L2Batcher) IsPaused() (bool, uint64) {
	pausable, ok := b.inner.(stack.PausableBatcher)
	b.require.True(ok, "batcher does not implement PausableBatcher")
	return pausable.IsPaused()
}
