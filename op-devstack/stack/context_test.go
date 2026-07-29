package stack

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/tri"
)

func TestContext(t *testing.T) {
	ctx := context.Background()
	chainA := eth.ChainIDFromUInt64(900)
	chainB := eth.ChainIDFromUInt64(901)
	require.Equal(t, eth.ChainID{}, ChainIDFromContext(ctx), "none")
	require.Equal(t, chainA, ChainIDFromContext(ContextWithChainID(ctx, chainA)), "lookup")
	require.Equal(t, chainB, ChainIDFromContext(ContextWithChainID(ContextWithChainID(ctx, chainA), chainB)), "priority")
}

func TestLogFilter(t *testing.T) {
	ctx := context.Background()
	chainA := eth.ChainIDFromUInt64(900)
	chainB := eth.ChainIDFromUInt64(901)
	fn := ChainIDSelector(chainA).Mute()
	require.Equal(t, tri.Undefined, fn(ctx, log.LevelDebug), "regular context should be false")
	require.Equal(t, tri.False, fn(ContextWithChainID(ctx, chainA), log.LevelDebug), "detected chain should be muted")
	require.Equal(t, tri.Undefined, fn(ContextWithChainID(ctx, chainB), log.LevelDebug), "different chain should be shown")
}
