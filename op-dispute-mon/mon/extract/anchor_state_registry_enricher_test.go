package extract

import (
	"context"
	"errors"
	"testing"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts"
	"github.com/ethereum-optimism/optimism/op-dispute-mon/mon/types"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching/rpcblock"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

func TestAnchorStateRegistryEnricher(t *testing.T) {
	logger := testlog.Logger(t, log.LevelError)
	expected := common.HexToAddress("0x0123456789abcDEF0123456789abCDef01234567")

	t.Run("Success", func(t *testing.T) {
		enricher := NewAnchorStateRegistryEnricher(logger)
		caller := &mockGameCaller{anchorStateRegistry: expected}
		game := &types.EnrichedGameData{}
		err := enricher.Enrich(context.Background(), rpcblock.Latest, caller, game)
		require.NoError(t, err)
		require.Equal(t, expected, game.AnchorStateRegistry)
	})

	t.Run("NotSupportedSkipsSilently", func(t *testing.T) {
		enricher := NewAnchorStateRegistryEnricher(logger)
		caller := &mockGameCaller{anchorStateRegErr: contracts.ErrAnchorStateRegistryNotSupported}
		game := &types.EnrichedGameData{}
		err := enricher.Enrich(context.Background(), rpcblock.Latest, caller, game)
		require.NoError(t, err)
		require.Equal(t, common.Address{}, game.AnchorStateRegistry)
	})

	t.Run("OtherErrorDoesNotFailGame", func(t *testing.T) {
		enricher := NewAnchorStateRegistryEnricher(logger)
		caller := &mockGameCaller{anchorStateRegErr: errors.New("boom")}
		game := &types.EnrichedGameData{}
		err := enricher.Enrich(context.Background(), rpcblock.Latest, caller, game)
		require.NoError(t, err)
		require.Equal(t, common.Address{}, game.AnchorStateRegistry)
	})
}
