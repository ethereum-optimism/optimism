package extract

import (
	"context"
	"errors"
	"math/big"
	"testing"

	faultTypes "github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	monTypes "github.com/ethereum-optimism/optimism/op-dispute-mon/mon/types"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching/rpcblock"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestBondEnricher(t *testing.T) {
	makeGame := func() *monTypes.EnrichedGameData {
		return &monTypes.EnrichedGameData{
			Claims: []faultTypes.Claim{
				{
					ClaimData: faultTypes.ClaimData{
						Bond: monTypes.ResolvedBondAmount,
					},
					Claimant:    common.Address{0x01},
					CounteredBy: common.Address{0x02},
				},
				{
					ClaimData: faultTypes.ClaimData{
						Bond: big.NewInt(5),
					},
					Claimant:    common.Address{0x03},
					CounteredBy: common.Address{},
				},
				{
					ClaimData: faultTypes.ClaimData{
						Bond: big.NewInt(7),
					},
					Claimant:    common.Address{0x03},
					CounteredBy: common.Address{},
				},
			},
		}
	}

	t.Run("GetCreditsFails", func(t *testing.T) {
		enricher := NewBondEnricher()
		caller := &mockGameCaller{creditsErr: errors.New("nope")}
		game := makeGame()
		err := enricher.Enrich(context.Background(), rpcblock.Latest, caller, game)
		require.ErrorIs(t, err, caller.creditsErr)
	})

	t.Run("GetCreditsWrongNumberOfResults", func(t *testing.T) {
		enricher := NewBondEnricher()
		caller := &mockGameCaller{extraCredit: []*big.Int{big.NewInt(4)}}
		game := makeGame()
		err := enricher.Enrich(context.Background(), rpcblock.Latest, caller, game)
		require.ErrorIs(t, err, ErrIncorrectCreditCount)
	})

	t.Run("GetCreditsSuccess", func(t *testing.T) {
		game := makeGame()
		expectedRecipients := []common.Address{
			game.Claims[0].Claimant,
			game.Claims[0].CounteredBy,
			game.Claims[1].Claimant,
			// Claim 1 CounteredBy is unset
			// Claim 2 Claimant is same as claim 1 Claimant
			// Claim 2 CounteredBy is unset
		}
		enricher := NewBondEnricher()
		expectedCredits := map[common.Address]*big.Int{
			expectedRecipients[0]: big.NewInt(10),
			expectedRecipients[1]: big.NewInt(20),
			expectedRecipients[2]: big.NewInt(30),
		}
		caller := &mockGameCaller{credits: expectedCredits}
		err := enricher.Enrich(context.Background(), rpcblock.Latest, caller, game)
		require.NoError(t, err)

		require.Equal(t, len(expectedRecipients), len(caller.requestedCredits))
		for _, recipient := range expectedRecipients {
			require.Contains(t, caller.requestedCredits, recipient)
		}
		require.Equal(t, expectedCredits, game.Credits)
	})
}
