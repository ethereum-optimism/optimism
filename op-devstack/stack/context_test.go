package stack

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/logfilter"
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
		require.Equal(t, UnknownKind, KindFromContext(ctx), "none")
		require.Equal(t, L2BatcherKind, KindFromContext(ContextWithKind(ctx, L2BatcherKind)), "lookup")
		require.Equal(t, L2ProposerKind, KindFromContext(ContextWithKind(ContextWithKind(ctx, L2BatcherKind), L2ProposerKind)), "priority")
	})
	t.Run("id", func(t *testing.T) {
		require.Equal(t, L2BatcherID{}, IDFromContext[L2BatcherID](ctx), "none")
		require.Equal(t, SuperchainID(""), IDFromContext[SuperchainID](ctx), "none")
		id1 := L2BatcherID{
			key:     "batcherA",
			chainID: chainA,
		}
		ctx1 := ContextWithID(ctx, id1)
		require.Equal(t, L2BatcherKind, KindFromContext(ctx1), "lookup kind")
		require.Equal(t, chainA, ChainIDFromContext(ctx1), "lookup chainID")
		require.Equal(t, id1, IDFromContext[L2BatcherID](ctx1), "lookup ID")
		// now overlay another different kind of ID on top
		id2 := SuperchainID("foobar")
		ctx2 := ContextWithID(ctx1, id2)
		require.Equal(t, SuperchainKind, KindFromContext(ctx2), "lookup kind")
		require.Equal(t, chainA, ChainIDFromContext(ctx2), "chainID still preserved")
		require.Equal(t, id2, IDFromContext[SuperchainID](ctx2), "lookup ID")
		require.Equal(t, L2BatcherID{}, IDFromContext[L2BatcherID](ctx2), "batcher ID not available")
	})
}

func TestLogFilter(t *testing.T) {
	ctx := context.Background()
	chainA := eth.ChainIDFromUInt64(900)
	chainB := eth.ChainIDFromUInt64(901)
	t.Run("chainID", func(t *testing.T) {
		fn := ChainIDLogFilter(chainA, logfilter.As(log.LevelCrit))
		require.Equal(t, log.LevelDebug, fn(ctx, log.LevelDebug), "regular level")
		require.Equal(t, log.LevelCrit, fn(ContextWithChainID(ctx, chainA), log.LevelDebug), "detected chain")
		require.Equal(t, log.LevelDebug, fn(ContextWithChainID(ctx, chainB), log.LevelDebug), "different chain")
	})
	t.Run("kind", func(t *testing.T) {
		fn := KindLogFilter(L2BatcherKind, logfilter.As(log.LevelCrit))
		require.Equal(t, log.LevelDebug, fn(ctx, log.LevelDebug), "regular level")
		require.Equal(t, log.LevelCrit, fn(ContextWithKind(ctx, L2BatcherKind), log.LevelDebug), "detected kind")
		require.Equal(t, log.LevelDebug, fn(ContextWithKind(ctx, L2ProposerKind), log.LevelDebug), "different kind")
	})
	t.Run("id", func(t *testing.T) {
		id1 := L2BatcherID{
			key:     "batcherA",
			chainID: chainA,
		}
		fn := IDLogFilter(id1, logfilter.As(log.LevelCrit))
		require.Equal(t, log.LevelDebug, fn(ctx, log.LevelDebug), "regular level")
		require.Equal(t, log.LevelCrit, fn(ContextWithID(ctx, id1), log.LevelDebug), "detected id")
		id2 := SuperchainID("foobar")
		require.Equal(t, log.LevelDebug, fn(ContextWithID(ctx, id2), log.LevelDebug), "different id")
	})
}
