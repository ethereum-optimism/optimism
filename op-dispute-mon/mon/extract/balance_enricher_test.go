package extract

import (
	"context"
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-dispute-mon/mon/types"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching/rpcblock"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestBalanceEnricher(t *testing.T) {
	t.Run("GetBalanceError", func(t *testing.T) {
		enricher := NewBalanceEnricher()
		caller := &mockGameCaller{balanceErr: errors.New("nope")}
		game := &types.EnrichedGameData{}
		err := enricher.Enrich(context.Background(), rpcblock.Latest, caller, game)
		require.ErrorIs(t, err, caller.balanceErr)
	})

	t.Run("GetBalanceSuccess", func(t *testing.T) {
		enricher := NewBalanceEnricher()
		caller := &mockGameCaller{balance: big.NewInt(84242), balanceAddr: common.Address{0xdd}}
		game := &types.EnrichedGameData{}
		err := enricher.Enrich(context.Background(), rpcblock.Latest, caller, game)
		require.NoError(t, err)
		require.Equal(t, game.WETHContract, caller.balanceAddr)
		require.Equal(t, game.ETHCollateral, caller.balance)
	})
}
