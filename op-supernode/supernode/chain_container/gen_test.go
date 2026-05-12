package chain_container

import (
	"context"
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestGenBumpsOnSetVN(t *testing.T) {
	t.Parallel()
	c := &simpleChainContainer{}
	before := c.gen.Load()
	c.setVN(newMockVirtualNode())
	require.Equal(t, before+1, c.gen.Load())
	c.setVN(newMockVirtualNode())
	require.Equal(t, before+2, c.gen.Load())
}

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
	cancel()
	_ = c.RewindEngine(ctx, 1234, eth.BlockRef{Number: 100, Hash: common.Hash{0x1}})

	require.GreaterOrEqual(t, c.gen.Load(), before+1)
}

func TestGenBumpsOnNotifyPipelineReset(t *testing.T) {
	t.Parallel()
	c := &simpleChainContainer{}
	before := c.gen.Load()
	c.NotifyPipelineReset()
	require.Equal(t, before+1, c.gen.Load())
	c.NotifyPipelineReset()
	require.Equal(t, before+2, c.gen.Load())
}
