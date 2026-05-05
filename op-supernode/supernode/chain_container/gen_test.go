package chain_container

import (
	"context"
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

// TestGenBumpsOnSetVN proves the chain container's generation counter
// increments every time a new virtual node is installed. The
// superroot_atTimestamp handler reads Generation() at the start and end of
// its gather to detect VN restarts; if this assertion regresses, mid-gather
// VN restarts go undetected.
func TestGenBumpsOnSetVN(t *testing.T) {
	t.Parallel()
	c := &simpleChainContainer{}
	before := c.gen.Load()
	c.setVN(newMockVirtualNode())
	require.Equal(t, before+1, c.gen.Load())
	c.setVN(newMockVirtualNode())
	require.Equal(t, before+2, c.gen.Load())
}

// TestGenBumpsOnRewindEngineEntry proves the chain container's generation
// counter increments as soon as RewindEngine claims c.resetting — before any
// engine-mutating work runs. A gather that observes pre-rewind state at
// start and post-rewind state at end will see the gen difference at its
// end check and discard the result.
func TestGenBumpsOnRewindEngineEntry(t *testing.T) {
	t.Parallel()
	mockVN := newMockVirtualNode()
	mockEngine := newMockEngineController()
	mockEngine.rewindFunc = func(ctx context.Context, ts uint64) error {
		return context.Canceled
	}
	c := &simpleChainContainer{
		log:    createTestLogger(t),
		engine: mockEngine,
		vn:     mockVN,
	}
	before := c.gen.Load()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // ensure the retry loop exits quickly
	_ = c.RewindEngine(ctx, 1234, eth.BlockRef{Number: 100, Hash: common.Hash{0x1}})

	require.GreaterOrEqual(t, c.gen.Load(), before+1)
}

// TestGenBumpsOnNotifyPipelineReset proves NotifyPipelineReset
// (rollup.SuperAuthority) bumps the generation counter. The StatusTracker
// invokes this from the inner derivation pipeline's rollup.ResetEvent
// handler, which is what surfaces non-RewindEngine resets — channel decode
// errors, blob fetch failures, attribute mismatches, etc. — to in-flight
// gathers.
func TestGenBumpsOnNotifyPipelineReset(t *testing.T) {
	t.Parallel()
	c := &simpleChainContainer{}
	before := c.gen.Load()
	c.NotifyPipelineReset()
	require.Equal(t, before+1, c.gen.Load())
	c.NotifyPipelineReset()
	require.Equal(t, before+2, c.gen.Load())
}
