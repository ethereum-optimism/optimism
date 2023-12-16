package disputegame

import (
	"context"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	"github.com/stretchr/testify/require"
)

type OutputHonestHelper struct {
	t            *testing.T
	require      *require.Assertions
	game         *OutputGameHelper
	contract     *contracts.OutputBisectionGameContract
	correctTrace types.TraceAccessor
}

func (h *OutputHonestHelper) Attack(ctx context.Context, claimIdx int64) {
	// Ensure the claim exists
	h.game.WaitForClaimCount(ctx, claimIdx+1)

	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	game, claim := h.loadState(ctx, claimIdx)
	attackPos := claim.Position.Attack()
	h.t.Logf("Attacking claim %v at position %v with g index %v", claimIdx, attackPos, attackPos.ToGIndex())
	value, err := h.correctTrace.Get(ctx, game, claim, attackPos)
	h.require.NoErrorf(err, "Get correct claim at position %v with g index %v", attackPos, attackPos.ToGIndex())
	h.t.Log("Performing attack")
	h.game.Attack(ctx, claimIdx, value)
	h.t.Log("Attack complete")
}

func (h *OutputHonestHelper) Defend(ctx context.Context, claimIdx int64) {
	// Ensure the claim exists
	h.game.WaitForClaimCount(ctx, claimIdx+1)

	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	game, claim := h.loadState(ctx, claimIdx)
	defendPos := claim.Position.Defend()
	value, err := h.correctTrace.Get(ctx, game, claim, defendPos)
	h.game.require.NoErrorf(err, "Get correct claim at position %v with g index %v", defendPos, defendPos.ToGIndex())
	h.game.Defend(ctx, claimIdx, value)
}

func (h *OutputHonestHelper) StepFails(ctx context.Context, claimIdx int64, isAttack bool) {
	// Ensure the claim exists
	h.game.WaitForClaimCount(ctx, claimIdx+1)

	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	game, claim := h.loadState(ctx, claimIdx)
	pos := claim.Position
	if !isAttack {
		// If we're defending, then the step will be from the trace to the next one
		pos = pos.MoveRight()
	}
	prestate, proofData, _, err := h.correctTrace.GetStepData(ctx, game, claim, pos)
	h.require.NoError(err, "Get step data")
	h.game.StepFails(claimIdx, isAttack, prestate, proofData)
}

func (h *OutputHonestHelper) loadState(ctx context.Context, claimIdx int64) (types.Game, types.Claim) {
	claims, err := h.contract.GetAllClaims(ctx)
	h.require.NoError(err, "Failed to load claims from game")
	game := types.NewGameState(claims, uint64(h.game.MaxDepth(ctx)))

	claim := game.Claims()[claimIdx]
	return game, claim
}
