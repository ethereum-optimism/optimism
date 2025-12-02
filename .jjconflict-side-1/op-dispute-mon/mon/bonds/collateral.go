package bonds

import (
	"math/big"

	monTypes "github.com/ethereum-optimism/optimism/op-dispute-mon/mon/types"
	"github.com/ethereum/go-ethereum/common"
)

type Collateral struct {
	// Required is the amount of collateral required to pay out bonds.
	Required *big.Int

	// Actual is the amount of collateral actually head by the DelayedWETH contract
	Actual *big.Int
}

// CalculateRequiredCollateral determines the minimum balance required for each DelayedWETH contract used by a set
// of dispute games.
// Returns a map of DelayedWETH contract address to collateral data (required and actual amounts)
func CalculateRequiredCollateral(games []*monTypes.EnrichedGameData) map[common.Address]Collateral {
	result := make(map[common.Address]Collateral)
	for _, game := range games {
		collateral, ok := result[game.WETHContract]
		if !ok {
			collateral = Collateral{
				Required: big.NewInt(0),
				Actual:   game.ETHCollateral,
			}
		}
		gameRequired := requiredCollateralForGame(game)
		collateral.Required = new(big.Int).Add(collateral.Required, gameRequired)
		result[game.WETHContract] = collateral
	}
	return result
}

func requiredCollateralForGame(game *monTypes.EnrichedGameData) *big.Int {
	required := big.NewInt(0)
	for _, claim := range game.Claims {
		if !claim.Resolved {
			required = new(big.Int).Add(required, claim.Bond)
		}
	}

	for _, unclaimedCredit := range game.Credits {
		required = new(big.Int).Add(required, unclaimedCredit)
	}
	return required
}
