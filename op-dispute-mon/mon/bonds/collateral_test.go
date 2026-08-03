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
	fault := &monTypes.FaultGameData{BondGameData: monTypes.BondGameData{
		Bonds: []monTypes.BondRecord{
			{Amount: big.NewInt(5)},
			{Amount: big.NewInt(100), Resolved: true},
		},
		Credits: map[common.Address]*big.Int{recipient: big.NewInt(3)},
		WithdrawalRequests: map[common.Address]*contracts.WithdrawalRequest{
			recipient: {Amount: big.NewInt(7)},
		},
		WETHContract:  weth,
		ETHCollateral: big.NewInt(100),
	}}
	zk := &monTypes.ZKGameData{BondGameData: monTypes.BondGameData{
		Bonds:              []monTypes.BondRecord{{Amount: big.NewInt(10)}},
		WETHContract:       weth,
		ETHCollateral:      big.NewInt(90),
		Credits:            map[common.Address]*big.Int{},
		WithdrawalRequests: map[common.Address]*contracts.WithdrawalRequest{},
	}}

	actual := CalculateRequiredCollateral([]monTypes.BondedGame{fault, zk})
	require.Equal(t, big.NewInt(22), actual[weth].Required)
	require.Equal(t, big.NewInt(90), actual[weth].Actual)
	require.True(t, actual[weth].BalancesDiffer)
}

func TestCalculateRequiredCollateralUsesLargestPendingRepresentation(t *testing.T) {
	weth := common.Address{0xaa}
	recipient := common.Address{0x11}
	game := &monTypes.ZKGameData{BondGameData: monTypes.BondGameData{
		WETHContract:  weth,
		ETHCollateral: big.NewInt(100),
		Credits:       map[common.Address]*big.Int{recipient: big.NewInt(9)},
		WithdrawalRequests: map[common.Address]*contracts.WithdrawalRequest{
			recipient: {Amount: big.NewInt(7)},
		},
	}}

	actual := CalculateRequiredCollateral([]monTypes.BondedGame{game})
	require.Equal(t, big.NewInt(9), actual[weth].Required)
}
