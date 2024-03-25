package bonds

import (
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	monTypes "github.com/ethereum-optimism/optimism/op-dispute-mon/mon/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestCalculateRequiredCollateral(t *testing.T) {
	weth1 := common.Address{0x1a}
	weth1Balance := big.NewInt(4200)
	weth2 := common.Address{0x2b}
	weth2Balance := big.NewInt(6000)
	game1 := &monTypes.EnrichedGameData{
		Claims: []monTypes.EnrichedClaim{
			{
				Claim: types.Claim{
					ClaimData: types.ClaimData{
						Bond: big.NewInt(17),
					},
					Claimant:    common.Address{0x01},
					CounteredBy: common.Address{0x02},
				},
				Resolved: true,
			},
			{
				Claim: types.Claim{
					ClaimData: types.ClaimData{
						Bond: big.NewInt(5),
					},
					Claimant:    common.Address{0x03},
					CounteredBy: common.Address{},
				},
			},
			{
				Claim: types.Claim{
					ClaimData: types.ClaimData{
						Bond: big.NewInt(7),
					},
					Claimant:    common.Address{0x03},
					CounteredBy: common.Address{},
				},
			},
		},
		Credits: map[common.Address]*big.Int{
			common.Address{0x01}: big.NewInt(2),
			common.Address{0x04}: big.NewInt(3),
		},
		WETHContract:  weth1,
		ETHCollateral: weth1Balance,
	}
	game2 := &monTypes.EnrichedGameData{
		Claims: []monTypes.EnrichedClaim{
			{
				Claim: types.Claim{
					ClaimData: types.ClaimData{
						Bond: big.NewInt(10),
					},
					Claimant:    common.Address{0x01},
					CounteredBy: common.Address{0x02},
				},
				Resolved: true,
			},
			{
				Claim: types.Claim{
					ClaimData: types.ClaimData{
						Bond: big.NewInt(6),
					},
					Claimant:    common.Address{0x03},
					CounteredBy: common.Address{},
				},
			},
			{
				Claim: types.Claim{
					ClaimData: types.ClaimData{
						Bond: big.NewInt(9),
					},
					Claimant:    common.Address{0x03},
					CounteredBy: common.Address{},
				},
			},
		},
		Credits: map[common.Address]*big.Int{
			common.Address{0x01}: big.NewInt(4),
			common.Address{0x04}: big.NewInt(1),
		},
		WETHContract:  weth1,
		ETHCollateral: weth1Balance,
	}
	game3 := &monTypes.EnrichedGameData{
		Claims: []monTypes.EnrichedClaim{
			{
				Claim: types.Claim{
					ClaimData: types.ClaimData{
						Bond: big.NewInt(23),
					},
					Claimant:    common.Address{0x03},
					CounteredBy: common.Address{},
				},
			},
		},
		Credits: map[common.Address]*big.Int{
			common.Address{0x01}: big.NewInt(46),
		},
		WETHContract:  weth2,
		ETHCollateral: weth2Balance,
	}
	actual := CalculateRequiredCollateral([]*monTypes.EnrichedGameData{game1, game2, game3})
	require.Len(t, actual, 2)
	require.Contains(t, actual, weth1)
	require.Contains(t, actual, weth2)
	require.Equal(t, actual[weth1].Required.Uint64(), uint64(5+7+2+3+6+9+4+1))
	require.Equal(t, actual[weth1].Actual.Uint64(), weth1Balance.Uint64())
	require.Equal(t, actual[weth2].Required.Uint64(), uint64(23+46))
	require.Equal(t, actual[weth2].Actual.Uint64(), weth2Balance.Uint64())
}
