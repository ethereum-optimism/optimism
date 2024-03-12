package bonds

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	monTypes "github.com/ethereum-optimism/optimism/op-dispute-mon/mon/types"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching/rpcblock"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestCalculateRequiredCollateral(t *testing.T) {
	claims := []types.Claim{
		{
			ClaimData: types.ClaimData{
				Bond: monTypes.ResolvedBondAmount,
			},
			Claimant:    common.Address{0x01},
			CounteredBy: common.Address{0x02},
		},
		{
			ClaimData: types.ClaimData{
				Bond: big.NewInt(5),
			},
			Claimant:    common.Address{0x03},
			CounteredBy: common.Address{},
		},
		{
			ClaimData: types.ClaimData{
				Bond: big.NewInt(7),
			},
			Claimant:    common.Address{0x03},
			CounteredBy: common.Address{},
		},
	}
	contract := &stubBondContract{
		credits: map[common.Address]*big.Int{
			{0x01}: big.NewInt(3),
			{0x03}: big.NewInt(8),
		},
	}
	collateral, err := CalculateRequiredCollateral(context.Background(), contract, common.Hash{0xab}, claims)
	require.NoError(t, err)
	require.Equal(t, collateral.Int64(), int64(5+7+3+8))
}

type stubBondContract struct {
	credits map[common.Address]*big.Int
}

func (s *stubBondContract) GetCredits(_ context.Context, _ rpcblock.Block, recipients ...common.Address) ([]*big.Int, error) {
	results := make([]*big.Int, len(recipients))
	for i, recipient := range recipients {
		credit, ok := s.credits[recipient]
		if !ok {
			credit = big.NewInt(0)
		}
		results[i] = credit
	}
	return results, nil
}
