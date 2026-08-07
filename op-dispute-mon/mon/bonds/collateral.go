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
func CalculateRequiredCollateral(games []monTypes.BondedGame) map[common.Address]Collateral {
	result := make(map[common.Address]Collateral)
	for _, game := range games {
		data := game.BondData()
		collateral, ok := result[data.WETHContract]
		if !ok {
			collateral = Collateral{
				Required: big.NewInt(0),
				Actual:   data.ETHCollateral,
			}
		}
		gameRequired := requiredCollateralForGame(game)
		collateral.Required = new(big.Int).Add(collateral.Required, gameRequired)
		result[data.WETHContract] = collateral
	}
	return result
}

func requiredCollateralForGame(game monTypes.BondedGame) *big.Int {
	data := game.BondData()
	required := big.NewInt(0)
	for _, bond := range data.Bonds {
		if !bond.Resolved {
			required = new(big.Int).Add(required, bond.Amount)
		}
	}

	for _, unclaimedCredit := range data.Credits {
		required = new(big.Int).Add(required, unclaimedCredit)
	}
	return required
}
