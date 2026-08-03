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
		actual := new(big.Int).Set(data.ETHCollateral)
		collateral, ok := result[data.WETHContract]
		if !ok {
			collateral = Collateral{
				Required: big.NewInt(0),
				Actual:   actual,
			}
		} else if collateral.Actual.Cmp(actual) != 0 {
			collateral.BalancesDiffer = true
			if actual.Cmp(collateral.Actual) < 0 {
				collateral.Actual = actual
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

	for _, recipient := range data.RecipientAddresses() {
		required = new(big.Int).Add(required, effectiveCredit(data, recipient))
	}
	return required
}

func effectiveCredit(data *monTypes.BondGameData, recipient common.Address) *big.Int {
	credit := data.Credits[recipient]
	request := data.WithdrawalRequests[recipient]
	if request.Amount.Cmp(credit) > 0 {
		return request.Amount
	}
	return credit
}
