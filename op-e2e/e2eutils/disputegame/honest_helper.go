package disputegame

import (
	"context"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	"github.com/stretchr/testify/require"
)

type HonestHelper struct {
	t            *testing.T
	require      *require.Assertions
	game         *FaultGameHelper
	correctTrace types.TraceProvider
}

func (h *HonestHelper) Attack(ctx context.Context, claimIdx int64) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	claim := h.game.getClaim(ctx, claimIdx)
	pos := types.NewPositionFromGIndex(claim.Position.Uint64())
	attackPos := pos.Attack()
	traceIdx := attackPos.TraceIndex(int(h.game.MaxDepth(ctx)))
	h.t.Logf("Attacking at position %v using correct trace from index %v", attackPos.ToGIndex(), traceIdx)
	value, err := h.correctTrace.Get(ctx, traceIdx)
	h.require.NoErrorf(err, "Get correct claim at trace index %v", traceIdx)
	h.t.Log("Performing attack")
	h.game.Attack(ctx, claimIdx, value)
	h.t.Log("Attack complete")
}

func (h *HonestHelper) Defend(ctx context.Context, claimIdx int64) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	claim := h.game.getClaim(ctx, claimIdx)
	pos := types.NewPositionFromGIndex(claim.Position.Uint64())
	defendPos := pos.Defend()
	traceIdx := defendPos.TraceIndex(int(h.game.MaxDepth(ctx)))
	value, err := h.correctTrace.Get(ctx, traceIdx)
	h.game.require.NoErrorf(err, "Get correct claim at trace index %v", traceIdx)
	h.game.Defend(ctx, claimIdx, value)
}
