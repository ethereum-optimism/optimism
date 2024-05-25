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
	pos := types.NewPositionFromGIndex(claim.Position)
	attackPos := pos.Attack()
	h.t.Logf("Attacking at position %v with g index %v", attackPos, attackPos.ToGIndex())
	value, err := h.correctTrace.Get(ctx, attackPos)
	h.require.NoErrorf(err, "Get correct claim at position %v with g index %v", attackPos, attackPos.ToGIndex())
	h.t.Log("Performing attack")
	h.game.Attack(ctx, claimIdx, value)
	h.t.Log("Attack complete")
}

func (h *HonestHelper) Defend(ctx context.Context, claimIdx int64) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	claim := h.game.getClaim(ctx, claimIdx)
	pos := types.NewPositionFromGIndex(claim.Position)
	defendPos := pos.Defend()
	value, err := h.correctTrace.Get(ctx, defendPos)
	h.game.require.NoErrorf(err, "Get correct claim at position %v with g index %v", defendPos, defendPos.ToGIndex())
	h.game.Defend(ctx, claimIdx, value)
}

func (h *HonestHelper) StepFails(ctx context.Context, claimIdx int64, isAttack bool) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	pos := h.game.getClaimPosition(ctx, claimIdx)
	if !isAttack {
		// If we're defending, then the step will be from the trace to the next one
		pos = pos.MoveRight()
	}
	prestate, proofData, _, err := h.correctTrace.GetStepData(ctx, pos)
	h.require.NoError(err, "Get step data")
	h.game.StepFails(claimIdx, isAttack, prestate, proofData)
}
