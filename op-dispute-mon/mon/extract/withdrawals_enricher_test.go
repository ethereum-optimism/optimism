package extract

import (
	"context"
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts"
	faultTypes "github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	monTypes "github.com/ethereum-optimism/optimism/op-dispute-mon/mon/types"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching/rpcblock"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestWithdrawalsEnricher(t *testing.T) {
	makeGame := func() *monTypes.EnrichedGameData {
		return &monTypes.EnrichedGameData{
			Recipients: map[common.Address]bool{
				common.Address{0x02}: true,
				common.Address{0x03}: true,
			},
			Claims: []monTypes.EnrichedClaim{
				{
					Claim: faultTypes.Claim{
						ClaimData: faultTypes.ClaimData{
							Bond: big.NewInt(10),
						},
						Claimant:    common.Address{0x01},
						CounteredBy: common.Address{0x02},
					},
					Resolved: true,
				},
				{
					Claim: faultTypes.Claim{
						ClaimData: faultTypes.ClaimData{
							Bond: big.NewInt(5),
						},
						Claimant:    common.Address{0x03},
						CounteredBy: common.Address{},
					},
				},
				{
					Claim: faultTypes.Claim{
						ClaimData: faultTypes.ClaimData{
							Bond: big.NewInt(7),
						},
						Claimant:    common.Address{0x03},
						CounteredBy: common.Address{},
					},
				},
			},
		}
	}

	t.Run("GetWithdrawalsFails", func(t *testing.T) {
		enricher := NewWithdrawalsEnricher()
		caller := &mockGameCaller{withdrawalsErr: errors.New("nope")}
		game := makeGame()
		err := enricher.Enrich(context.Background(), rpcblock.Latest, caller, game)
		require.ErrorIs(t, err, caller.withdrawalsErr)
	})

	t.Run("GetWithdrawalsWrongNumberOfResults", func(t *testing.T) {
		enricher := NewWithdrawalsEnricher()
		caller := &mockGameCaller{withdrawals: []*contracts.WithdrawalRequest{{}}}
		game := makeGame()
		err := enricher.Enrich(context.Background(), rpcblock.Latest, caller, game)
		require.ErrorIs(t, err, ErrIncorrectWithdrawalsCount)
	})

	t.Run("GetWithdrawalsSuccess", func(t *testing.T) {
		game := makeGame()
		enricher := NewWithdrawalsEnricher()
		caller := &mockGameCaller{}
		err := enricher.Enrich(context.Background(), rpcblock.Latest, caller, game)
		require.NoError(t, err)
		require.Equal(t, 2, len(game.WithdrawalRequests))
	})
}
