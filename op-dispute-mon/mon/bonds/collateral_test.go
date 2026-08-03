package bonds

import (
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts"
	monTypes "github.com/ethereum-optimism/optimism/op-dispute-mon/mon/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestCalculateRequiredCollateralAggregatesFaultAndZKGames(t *testing.T) {
	weth := common.Address{0xaa}
	recipient := common.Address{0x11}
	faultData := bondData(weth, 100, []monTypes.BondRecord{
		{Amount: big.NewInt(5)},
		{Amount: big.NewInt(100), Resolved: true},
	})
	faultData.Credits[recipient] = big.NewInt(3)
	faultData.WithdrawalRequests[recipient] = &contracts.WithdrawalRequest{
		Amount: big.NewInt(7), Timestamp: new(big.Int),
	}
	fault := &monTypes.FaultGameData{BondGameData: faultData}
	zk := &monTypes.ZKGameData{BondGameData: bondData(weth, 90, []monTypes.BondRecord{{Amount: big.NewInt(10)}})}

	actual := CalculateRequiredCollateral([]monTypes.BondedGame{fault, zk})
	require.Equal(t, big.NewInt(22), actual[weth].Required)
	require.Equal(t, big.NewInt(90), actual[weth].Actual)
	require.True(t, actual[weth].BalancesDiffer)

	zk.ETHCollateral.SetInt64(1)
	require.Equal(t, big.NewInt(90), actual[weth].Actual)
}

func TestCalculateRequiredCollateralUsesLargestPendingRepresentation(t *testing.T) {
	weth := common.Address{0xaa}
	recipient := common.Address{0x11}
	data := bondData(weth, 100, nil)
	data.Credits[recipient] = big.NewInt(9)
	data.WithdrawalRequests[recipient] = &contracts.WithdrawalRequest{
		Amount: big.NewInt(7), Timestamp: new(big.Int),
	}
	game := &monTypes.ZKGameData{BondGameData: data}

	actual := CalculateRequiredCollateral([]monTypes.BondedGame{game})
	require.Equal(t, big.NewInt(9), actual[weth].Required)
}

func TestCalculateRequiredCollateralRequiresNormalizedSnapshot(t *testing.T) {
	recipient := common.Address{0x11}
	game := &monTypes.ZKGameData{BondGameData: monTypes.BondGameData{
		Recipients:         map[common.Address]bool{recipient: true},
		Credits:            map[common.Address]*big.Int{recipient: nil},
		ExpectedCredits:    map[common.Address]*big.Int{},
		WithdrawalRequests: map[common.Address]*contracts.WithdrawalRequest{recipient: nil},
		ETHCollateral:      big.NewInt(100),
	}}

	require.Panics(t, func() {
		CalculateRequiredCollateral([]monTypes.BondedGame{game})
	})
}
