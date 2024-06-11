package solver

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-dispute-mon/mon"
	"github.com/ethereum-optimism/optimism/op-dispute-mon/mon/transform"
	disputeTypes "github.com/ethereum-optimism/optimism/op-dispute-mon/mon/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func verifyGameRules(t *testing.T, game types.Game, rootClaimCorrect bool) {
	actualResult, claimTree, resolvedGame := gameResult(game)

	verifyExpectedGameResult(t, rootClaimCorrect, actualResult)

	verifyNoChallengerClaimsWereSuccessfullyCountered(t, resolvedGame)
	verifyChallengerAlwaysWinsParentBond(t, resolvedGame)
	verifyChallengerNeverCountersAClaimTwice(t, claimTree)
}

// verifyExpectedGameResult verifies that valid output roots are successfully defended and invalid roots are challenged
// Rationale: Ensures the game always and only allows valid output roots to be finalized.
func verifyExpectedGameResult(t *testing.T, rootClaimCorrect bool, actualResult gameTypes.GameStatus) {
	expectedResult := gameTypes.GameStatusChallengerWon
	if rootClaimCorrect {
		expectedResult = gameTypes.GameStatusDefenderWon
	}
	require.Equalf(t, expectedResult, actualResult, "Game should resolve correctly expected %v but was %v", expectedResult, actualResult)
}

// verifyNoChallengerClaimsWereSuccessfullyCountered verifies the challenger didn't lose any of its bonds
// Note that this also forbids the challenger losing a bond to itself since it shouldn't challenge its own claims
// Rationale: If honest actors lose their bond, it indicates that incentive compatibility is broken because honest actors
// lose money.
func verifyNoChallengerClaimsWereSuccessfullyCountered(t *testing.T, resolvedGame types.Game) {
	for _, claim := range resolvedGame.Claims() {
		if claim.Claimant != challengerAddr {
			continue
		}
		if claim.CounteredBy != (common.Address{}) {
			t.Fatalf("Challenger posted claim %v but it was countered by someone else:\n%v", claim.ContractIndex, printClaim(claim, resolvedGame))
		}
	}
}

// verifyChallengerAlwaysWinsParentBond verifies that the challenger is always allocated the bond of any parent claim it
// counters.
// Rationale: If an honest action does not win the bond for countering a claim, incentive compatibility is broken because
// honest actors are not being paid to perform their job (or the challenger is posting unnecessary claims)
func verifyChallengerAlwaysWinsParentBond(t *testing.T, resolvedGame types.Game) {
	for _, claim := range resolvedGame.Claims() {
		if claim.Claimant != challengerAddr {
			continue
		}
		parent, err := resolvedGame.GetParent(claim)
		require.NoErrorf(t, err, "Failed to get parent of claim %v", claim.ContractIndex)
		require.Equal(t, challengerAddr, parent.CounteredBy,
			"Expected claim %v to have challenger as its claimant because of counter claim %v", parent.ContractIndex, claim.ContractIndex)
	}
}

// verifyChallengerNeverCountersAClaimTwice verifies that the challenger never posts more than one counter to a claim
// Rationale: The parent claim bond is only intended to cover costs of a single counter claim so incentive compatibility
// is broken if the challenger needs to post multiple claims. Or if the claim wasn't required, the challenger is just
// wasting money posting unnecessary claims.
func verifyChallengerNeverCountersAClaimTwice(t *testing.T, tree *disputeTypes.BidirectionalTree) {
	for _, claim := range tree.Claims {
		challengerCounterCount := 0
		for _, child := range claim.Children {
			if child.Claim.Claimant != challengerAddr {
				continue
			}
			challengerCounterCount++
		}
		require.LessOrEqualf(t, challengerCounterCount, 1, "Found multiple honest counters to claim %v", claim.Claim.ContractIndex)
	}
}

func enrichClaims(claims []types.Claim) []disputeTypes.EnrichedClaim {
	enriched := make([]disputeTypes.EnrichedClaim, len(claims))
	for i, claim := range claims {
		enriched[i] = disputeTypes.EnrichedClaim{Claim: claim}
	}
	return enriched
}

func gameResult(game types.Game) (gameTypes.GameStatus, *disputeTypes.BidirectionalTree, types.Game) {
	tree := transform.CreateBidirectionalTree(enrichClaims(game.Claims()))
	result := mon.Resolve(tree)
	resolvedClaims := make([]types.Claim, 0, len(tree.Claims))
	for _, claim := range tree.Claims {
		resolvedClaims = append(resolvedClaims, *claim.Claim)
	}
	return result, tree, types.NewGameState(resolvedClaims, game.MaxDepth())
}
