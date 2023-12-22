package disputegame

import (
	"context"
	"errors"
	"time"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	"github.com/ethereum/go-ethereum/common"
)

type dishonestOutputClaim struct {
	ParentIndex int64
	IsAttack    bool
	Valid       bool
}

type OutputDishonestHelper struct {
	*OutputGameHelper
	*OutputHonestHelper
	claims   map[dishonestOutputClaim]bool
	defender bool
}

func newOutputDishonestHelper(g *OutputGameHelper, correctTrace *OutputHonestHelper, defender bool) *OutputDishonestHelper {
	return &OutputDishonestHelper{g, correctTrace, make(map[dishonestOutputClaim]bool), defender}
}

func (t *OutputDishonestHelper) Attack(ctx context.Context, claimIndex int64) {
	c := dishonestOutputClaim{claimIndex, true, false}
	if t.claims[c] {
		return
	}
	t.claims[c] = true
	t.OutputGameHelper.Attack(ctx, claimIndex, common.Hash{byte(claimIndex)})
}

func (t *OutputDishonestHelper) Defend(ctx context.Context, claimIndex int64) {
	c := dishonestOutputClaim{claimIndex, false, false}
	if t.claims[c] {
		return
	}
	t.claims[c] = true
	t.OutputGameHelper.Defend(ctx, claimIndex, common.Hash{byte(claimIndex)})
}

func (t *OutputDishonestHelper) AttackCorrect(ctx context.Context, claimIndex int64) {
	c := dishonestOutputClaim{claimIndex, true, true}
	if t.claims[c] {
		return
	}
	t.claims[c] = true
	t.OutputHonestHelper.Attack(ctx, claimIndex)
}

func (t *OutputDishonestHelper) DefendCorrect(ctx context.Context, claimIndex int64) {
	c := dishonestOutputClaim{claimIndex, false, true}
	if t.claims[c] {
		return
	}
	t.claims[c] = true
	t.OutputHonestHelper.Defend(ctx, claimIndex)
}

// ExhaustDishonestClaims makes all possible significant moves (mod honest challenger's) in a game.
// It is very inefficient and should NOT be used on games with large depths
func (d *OutputDishonestHelper) ExhaustDishonestClaims(ctx context.Context) {
	depth := d.OutputGameHelper.MaxDepth(ctx)

	move := func(claimIndex int64, claimData ContractClaim) {
		// dishonest level, valid attack
		// dishonest level, invalid attack
		// dishonest level, valid defense
		// dishonest level, invalid defense
		// honest level, invalid attack
		// honest level, invalid defense

		pos := types.NewPositionFromGIndex(claimData.Position)
		if int64(pos.Depth()) == depth {
			return
		}

		d.OutputGameHelper.LogGameData(ctx)
		d.OutputGameHelper.t.Logf("Dishonest moves against claimIndex %d", claimIndex)
		agreeWithLevel := d.defender == (pos.Depth()%2 == 0)
		if !agreeWithLevel {
			d.AttackCorrect(ctx, claimIndex)
			if claimIndex != 0 {
				d.DefendCorrect(ctx, claimIndex)
			}
		}
		d.Attack(ctx, claimIndex)
		if claimIndex != 0 {
			d.Defend(ctx, claimIndex)
		}
	}

	var numClaimsSeen int64
	for {
		// Use a short timeout since we don't know the challenger will respond,
		// and this is only designed for the alphabet game where the response should be fast.
		newCount, err := d.OutputGameHelper.waitForNewClaim(ctx, numClaimsSeen, 30*time.Second)
		if errors.Is(err, context.DeadlineExceeded) {
			// we assume that the honest challenger has stopped responding
			// There's nothing to respond to.
			break
		}
		d.OutputGameHelper.require.NoError(err)

		for i := numClaimsSeen; i < newCount; i++ {
			claimData := d.OutputGameHelper.getClaim(ctx, numClaimsSeen)
			move(numClaimsSeen, claimData)
			numClaimsSeen++
		}
	}
}
