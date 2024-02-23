package resolution

import (
	"github.com/ethereum/go-ethereum/common"

	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	monTypes "github.com/ethereum-optimism/optimism/op-dispute-mon/mon/types"
)

// Resolve iterates backwards over the bidirectional tree, iteratively
// checking the leftmost counter of each claim, and updating the claim's counter
// claimant. Once the root claim is reached, the resolution game status is returned.
func Resolve(tree *monTypes.BidirectionalTree) gameTypes.GameStatus {
	for i := len(tree.Claims) - 1; i >= 0; i-- {
		claim := tree.Claims[i]
		counterClaimant := claim.Claim.CounteredBy
		for _, child := range claim.Children {
			if child.Claim.CounteredBy == (common.Address{}) {
				counterClaimant = child.Claim.Claimant
			}
		}
		claim.Claim.CounteredBy = counterClaimant
	}
	if (len(tree.Claims) == 0 || tree.Claims[0].Claim.CounteredBy == common.Address{}) {
		return gameTypes.GameStatusDefenderWon
	} else {
		return gameTypes.GameStatusChallengerWon
	}
}
