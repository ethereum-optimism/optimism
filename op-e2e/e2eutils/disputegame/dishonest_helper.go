package disputegame

import (
	"context"
	"errors"
	"time"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	"github.com/ethereum/go-ethereum/common"
)

type DishonestHelper struct {
	*OutputGameHelper
	*OutputHonestHelper
	defender bool
}

func newDishonestHelper(g *OutputGameHelper, correctTrace *OutputHonestHelper, defender bool) *DishonestHelper {
	return &DishonestHelper{g, correctTrace, defender}
}

// ExhaustDishonestClaims makes all possible significant moves (mod honest challenger's) in a game.
// It is very inefficient and should NOT be used on games with large depths
func (d *DishonestHelper) ExhaustDishonestClaims(ctx context.Context, rootClaim *ClaimHelper) {
	depth := d.MaxDepth(ctx)
	splitDepth := d.SplitDepth(ctx)

	move := func(claimIndex int64, claimData types.Claim) {
		// dishonest level, valid attack
		// dishonest level, invalid attack
		// dishonest level, valid defense
		// dishonest level, invalid defense
		// honest level, invalid attack
		// honest level, invalid defense

		if claimData.Depth() == depth {
			return
		}

		d.LogGameData(ctx)
		d.OutputGameHelper.T.Logf("Dishonest moves against claimIndex %d", claimIndex)
		agreeWithLevel := d.defender == (claimData.Depth()%2 == 0)
		if !agreeWithLevel {
			d.OutputHonestHelper.Attack(ctx, claimIndex, WithIgnoreDuplicates())
			if claimIndex != 0 && claimData.Depth() != splitDepth+1 {
				d.OutputHonestHelper.Defend(ctx, claimIndex, WithIgnoreDuplicates())
			}
		}
		d.OutputGameHelper.Attack(ctx, claimIndex, common.Hash{byte(claimIndex)}, WithIgnoreDuplicates())
		if claimIndex != 0 && claimData.Depth() != splitDepth+1 {
			d.OutputGameHelper.Defend(ctx, claimIndex, common.Hash{byte(claimIndex)}, WithIgnoreDuplicates())
		}
	}

	numClaimsSeen := rootClaim.Index
	for {
		// Use a short timeout since we don't know the challenger will respond,
		// and this is only designed for the alphabet game where the response should be fast.
		newCount, err := d.waitForNewClaim(ctx, numClaimsSeen, 30*time.Second)
		if errors.Is(err, context.DeadlineExceeded) {
			// we assume that the honest challenger has stopped responding
			// There's nothing to respond to.
			break
		}
		d.OutputGameHelper.Require.NoError(err)

		for ; numClaimsSeen < newCount; numClaimsSeen++ {
			claimData := d.getClaim(ctx, numClaimsSeen)
			move(numClaimsSeen, claimData)
		}
	}
}
