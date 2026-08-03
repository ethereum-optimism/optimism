package bonds

import (
	"math/big"

	monTypes "github.com/ethereum-optimism/optimism/op-dispute-mon/mon/types"
	"github.com/ethereum/go-ethereum/common"
)

type Collateral struct {
	// Required is the amount of collateral required to pay out bonds.
	Required *big.Int

	// Actual is the amount of collateral actually held by the DelayedWETH contract
	Actual *big.Int

	// BalancesDiffer indicates games sharing this DelayedWETH returned different pinned balances.
	BalancesDiffer bool
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
		} else if collateral.Actual.Cmp(data.ETHCollateral) != 0 {
			collateral.BalancesDiffer = true
			if data.ETHCollateral.Cmp(collateral.Actual) < 0 {
				collateral.Actual = data.ETHCollateral
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

	recipients := make(map[common.Address]bool)
	for recipient := range data.Credits {
		recipients[recipient] = true
	}
	for recipient := range data.WithdrawalRequests {
		recipients[recipient] = true
	}
	for recipient := range recipients {
		required = new(big.Int).Add(required, effectiveCredit(data, recipient))
	}
	return required
}

func effectiveCredit(data *monTypes.BondGameData, recipient common.Address) *big.Int {
	credit := data.Credits[recipient]
	if credit == nil {
		credit = new(big.Int)
	}
	request := data.WithdrawalRequests[recipient]
	if request != nil && request.Amount != nil && request.Amount.Cmp(credit) > 0 {
		return request.Amount
	}
	return credit
}
