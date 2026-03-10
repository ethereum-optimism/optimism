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
		require.Equal(t, ComponentKind(""), KindFromContext(ctx), "none")
		require.Equal(t, KindL2Batcher, KindFromContext(ContextWithKind(ctx, KindL2Batcher)), "lookup")
		require.Equal(t, KindL2Proposer, KindFromContext(ContextWithKind(ContextWithKind(ctx, KindL2Batcher), KindL2Proposer)), "priority")
	})
	t.Run("id", func(t *testing.T) {
		require.Equal(t, ComponentID{}, IDFromContext[ComponentID](ctx), "none")
		id1 := NewL2BatcherID("batcherA", chainA)
		ctx1 := ContextWithID(ctx, id1)
		require.Equal(t, KindL2Batcher, KindFromContext(ctx1), "lookup kind")
		require.Equal(t, chainA, ChainIDFromContext(ctx1), "lookup chainID")
		require.Equal(t, id1, IDFromContext[ComponentID](ctx1), "lookup ID")
		// now overlay another different kind of ID on top
		id2 := NewSuperchainID("foobar")
		ctx2 := ContextWithID(ctx1, id2)
		require.Equal(t, KindSuperchain, KindFromContext(ctx2), "lookup kind")
		require.Equal(t, chainA, ChainIDFromContext(ctx2), "chainID still preserved")
		require.Equal(t, id2, IDFromContext[ComponentID](ctx2), "lookup ID - now shows superchain")
		// With type aliases, IDFromContext returns the stored ComponentID regardless of "type"
		// The Kind() method can be used to check the actual kind of ID
		require.Equal(t, KindSuperchain, IDFromContext[ComponentID](ctx2).Kind(), "id kind check")
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
		fn := KindSelector(KindL2Batcher).Mute()
		require.Equal(t, tri.Undefined, fn(ctx, log.LevelDebug), "regular context should be false")
		require.Equal(t, tri.False, fn(ContextWithKind(ctx, KindL2Batcher), log.LevelDebug), "detected kind should be muted")
		require.Equal(t, tri.Undefined, fn(ContextWithKind(ctx, KindL2Proposer), log.LevelDebug), "different kind should be shown")
	})
	t.Run("id", func(t *testing.T) {
		id1 := NewL2BatcherID("batcherA", chainA)
		fn := IDSelector(id1).Mute()
		require.Equal(t, tri.Undefined, fn(ctx, log.LevelDebug), "regular context should be false")
		require.Equal(t, tri.False, fn(ContextWithID(ctx, id1), log.LevelDebug), "detected id should be muted")
		id2 := NewSuperchainID("foobar")
		require.Equal(t, tri.Undefined, fn(ContextWithID(ctx, id2), log.LevelDebug), "different id should be shown")
	})
}
