package bonds

import (
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts"
	monTypes "github.com/ethereum-optimism/optimism/op-dispute-mon/mon/types"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestCalculateRequiredCollateral(t *testing.T) {
	weth1 := common.Address{0x1a}
	weth1Balance := big.NewInt(4200)
	weth2 := common.Address{0x2b}
	weth2Balance := big.NewInt(6000)
	game1 := &monTypes.FaultGameData{BondGameData: monTypes.BondGameData{
		Bonds: []monTypes.BondRecord{
			{Amount: big.NewInt(17), Resolved: true},
			{Amount: big.NewInt(5)},
			{Amount: big.NewInt(7)},
		},
		Credits: map[common.Address]*big.Int{
			common.Address{0x01}: big.NewInt(2),
			common.Address{0x04}: big.NewInt(3),
		},
		WETHContract:  weth1,
		ETHCollateral: weth1Balance,
	}}
	game2 := &monTypes.FaultGameData{BondGameData: monTypes.BondGameData{
		Bonds: []monTypes.BondRecord{
			{Amount: big.NewInt(10), Resolved: true},
			{Amount: big.NewInt(6)},
			{Amount: big.NewInt(9)},
		},
		Credits: map[common.Address]*big.Int{
			common.Address{0x01}: big.NewInt(4),
			common.Address{0x04}: big.NewInt(1),
		},
		WETHContract:  weth1,
		ETHCollateral: weth1Balance,
	}}
	game3 := &monTypes.FaultGameData{BondGameData: monTypes.BondGameData{
		Bonds: []monTypes.BondRecord{{Amount: big.NewInt(23)}},
		Credits: map[common.Address]*big.Int{
			common.Address{0x01}: big.NewInt(46),
		},
		WETHContract:  weth2,
		ETHCollateral: weth2Balance,
	}}

	actual := CalculateRequiredCollateral([]monTypes.BondedGame{game1, game2, game3})
	require.Len(t, actual, 2)
	require.Contains(t, actual, weth1)
	require.Contains(t, actual, weth2)
	require.Equal(t, uint64(5+7+2+3+6+9+4+1), bigs.Uint64Strict(actual[weth1].Required))
	require.Equal(t, bigs.Uint64Strict(weth1Balance), bigs.Uint64Strict(actual[weth1].Actual))
	require.Equal(t, uint64(23+46), bigs.Uint64Strict(actual[weth2].Required))
	require.Equal(t, bigs.Uint64Strict(weth2Balance), bigs.Uint64Strict(actual[weth2].Actual))
}

func TestCalculateRequiredCollateralAggregatesFaultAndZK(t *testing.T) {
	weth := common.Address{0xaa}
	recipient := common.Address{0x01}
	fault := &monTypes.FaultGameData{BondGameData: monTypes.BondGameData{
		Bonds:         []monTypes.BondRecord{{Amount: big.NewInt(5)}},
		Credits:       map[common.Address]*big.Int{recipient: big.NewInt(2)},
		WETHContract:  weth,
		ETHCollateral: big.NewInt(100),
	}}
	zk := &monTypes.ZKGameData{BondGameData: monTypes.BondGameData{
		Bonds:   []monTypes.BondRecord{{Amount: big.NewInt(11)}},
		Credits: map[common.Address]*big.Int{recipient: big.NewInt(3)},
		WithdrawalRequests: map[common.Address]*contracts.WithdrawalRequest{
			recipient: {Amount: big.NewInt(7), Timestamp: big.NewInt(1)},
		},
		WETHContract:  weth,
		ETHCollateral: big.NewInt(100),
	}}

	actual := CalculateRequiredCollateral([]monTypes.BondedGame{fault, zk})
	require.Equal(t, big.NewInt(25), actual[weth].Required)
	require.Equal(t, big.NewInt(100), actual[weth].Actual)
}
