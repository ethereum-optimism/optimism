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
	t.Run("chainID", func(t *testing.T) {
		require.Equal(t, eth.ChainID{}, ChainIDFromContext(ctx), "none")
		require.Equal(t, chainA, ChainIDFromContext(ContextWithChainID(ctx, chainA)), "lookup")
		require.Equal(t, chainB, ChainIDFromContext(ContextWithChainID(ContextWithChainID(ctx, chainA), chainB)), "priority")
	})
	t.Run("kind", func(t *testing.T) {
		require.Equal(t, Kind(""), KindFromContext(ctx), "none")
		require.Equal(t, L2BatcherKind, KindFromContext(ContextWithKind(ctx, L2BatcherKind)), "lookup")
		require.Equal(t, L2ProposerKind, KindFromContext(ContextWithKind(ContextWithKind(ctx, L2BatcherKind), L2ProposerKind)), "priority")
	})
	t.Run("id", func(t *testing.T) {
		require.Equal(t, ComponentID{}, ComponentIDFromContext(ctx), "none")
		require.Equal(t, NewSuperchainID(""), ComponentIDFromContext(ctx), "none")
		id1 := NewL2BatcherID("batcherA", chainA)
		ctx1 := ContextWithComponentID(ctx, id1)
		require.Equal(t, L2BatcherKind, KindFromContext(ctx1), "lookup kind")
		require.Equal(t, chainA, ChainIDFromContext(ctx1), "lookup chainID")
		require.Equal(t, id1, ComponentIDFromContext(ctx1), "lookup ID")
		// now overlay another different kind of ID on top
		id2 := NewSuperchainID("foobar")
		ctx2 := ContextWithComponentID(ctx1, id2)
		require.Equal(t, SuperchainKind, KindFromContext(ctx2), "lookup kind")
		require.Equal(t, chainA, ChainIDFromContext(ctx2), "chainID still preserved")
		require.Equal(t, id2, ComponentIDFromContext(ctx2), "lookup ID")
		require.Equal(t, ComponentID{}, ComponentIDFromContext(ctx2), "batcher ID not available")
	})
}

func TestLogFilter(t *testing.T) {
	ctx := context.Background()
	chainA := eth.ChainIDFromUInt64(900)
	chainB := eth.ChainIDFromUInt64(901)
	t.Run("chainID", func(t *testing.T) {
		fn := ChainIDSelector(chainA).Mute()
		require.Equal(t, tri.Undefined, fn(ctx, log.LevelDebug), "regular context should be false")
		require.Equal(t, tri.False, fn(ContextWithChainID(ctx, chainA), log.LevelDebug), "detected chain should be muted")
		require.Equal(t, tri.Undefined, fn(ContextWithChainID(ctx, chainB), log.LevelDebug), "different chain should be shown")
	})
	t.Run("kind", func(t *testing.T) {
		fn := KindSelector(L2BatcherKind).Mute()
		require.Equal(t, tri.Undefined, fn(ctx, log.LevelDebug), "regular context should be false")
		require.Equal(t, tri.False, fn(ContextWithKind(ctx, L2BatcherKind), log.LevelDebug), "detected kind should be muted")
		require.Equal(t, tri.Undefined, fn(ContextWithKind(ctx, L2ProposerKind), log.LevelDebug), "different kind should be shown")
	})
	t.Run("id", func(t *testing.T) {
		id1 := NewL2BatcherID("batcherA", chainA)
		fn := IDSelector(id1).Mute()
		require.Equal(t, tri.Undefined, fn(ctx, log.LevelDebug), "regular context should be false")
		require.Equal(t, tri.False, fn(ContextWithComponentID(ctx, id1), log.LevelDebug), "detected id should be muted")
		id2 := NewSuperchainID("foobar")
		require.Equal(t, tri.Undefined, fn(ContextWithComponentID(ctx, id2), log.LevelDebug), "different id should be shown")
	})
}
