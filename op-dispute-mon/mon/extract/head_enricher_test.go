package extract

import (
	"context"
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-dispute-mon/mon/types"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching/rpcblock"
	"github.com/ethereum/go-ethereum/common"
	gethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func TestL1HeadEnricher(t *testing.T) {
	t.Run("HeaderError", func(t *testing.T) {
		client := &stubBlockFetcher{err: errors.New("boom")}
		enricher := NewL1HeadBlockNumEnricher(client)
		caller := &mockGameCaller{}
		game := &types.EnrichedGameData{}
		err := enricher.Enrich(context.Background(), rpcblock.Latest, caller, game)
		require.ErrorIs(t, err, client.err)
	})

	t.Run("GetBalanceSuccess", func(t *testing.T) {
		client := &stubBlockFetcher{num: 5000}
		enricher := NewL1HeadBlockNumEnricher(client)
		caller := &mockGameCaller{}
		game := &types.EnrichedGameData{}
		err := enricher.Enrich(context.Background(), rpcblock.Latest, caller, game)
		require.NoError(t, err)
		require.Equal(t, client.num, game.L1HeadNum)
	})
}

type stubBlockFetcher struct {
	num uint64
	err error
}

func (s *stubBlockFetcher) HeaderByHash(_ context.Context, _ common.Hash) (*gethTypes.Header, error) {
	if s.err != nil {
		return nil, s.err
	}
	return &gethTypes.Header{
		Number: new(big.Int).SetUint64(s.num),
	}, nil
}
