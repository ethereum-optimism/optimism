package loader

import (
	"context"
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

var (
	gameCountErr = errors.New("game count error")
	gameIndexErr = errors.New("game index error")
)

// TestGameLoader_FetchAllGames tests that the game loader correctly fetches all games.
func TestGameLoader_FetchAllGames(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		caller      *mockMinimalDisputeGameFactoryCaller
		earliest    uint64
		blockHash   common.Hash
		expectedErr error
		expectedLen int
	}{
		{
			name:        "success",
			caller:      newMockMinimalDisputeGameFactoryCaller(10, false, false),
			blockHash:   common.Hash{0x01},
			expectedLen: 10,
		},
		{
			name:        "expired game ignored",
			caller:      newMockMinimalDisputeGameFactoryCaller(10, false, false),
			earliest:    500,
			blockHash:   common.Hash{0x01},
			expectedLen: 5,
		},
		{
			name:        "game count error",
			caller:      newMockMinimalDisputeGameFactoryCaller(10, true, false),
			blockHash:   common.Hash{0x01},
			expectedErr: gameCountErr,
		},
		{
			name:        "game index error",
			caller:      newMockMinimalDisputeGameFactoryCaller(10, false, true),
			blockHash:   common.Hash{0x01},
			expectedErr: gameIndexErr,
		},
		{
			name:      "no games",
			caller:    newMockMinimalDisputeGameFactoryCaller(0, false, false),
			blockHash: common.Hash{0x01},
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			loader := NewGameLoader(test.caller)
			games, err := loader.FetchAllGamesAtBlock(context.Background(), test.earliest, test.blockHash)
			require.ErrorIs(t, err, test.expectedErr)
			require.Len(t, games, test.expectedLen)
			expectedGames := test.caller.games
			expectedGames = expectedGames[len(expectedGames)-test.expectedLen:]
			if test.expectedErr != nil {
				expectedGames = make([]types.GameMetadata, 0)
			}
			require.ElementsMatch(t, expectedGames, translateGames(games))
		})
	}
}

func generateMockGames(count uint64) []types.GameMetadata {
	games := make([]types.GameMetadata, count)

	for i := uint64(0); i < count; i++ {
		games[i] = types.GameMetadata{
			Proxy:     common.BigToAddress(big.NewInt(int64(i))),
			Timestamp: i * 100,
		}
	}

	return games
}

func translateGames(games []types.GameMetadata) []types.GameMetadata {
	translated := make([]types.GameMetadata, len(games))

	for i, game := range games {
		translated[i] = translateFaultDisputeGame(game)
	}

	return translated
}

func translateFaultDisputeGame(game types.GameMetadata) types.GameMetadata {
	return types.GameMetadata{
		Proxy:     game.Proxy,
		Timestamp: game.Timestamp,
	}
}

func generateMockGameErrors(count uint64, injectErrors bool) []bool {
	errors := make([]bool, count)

	if injectErrors {
		for i := uint64(0); i < count; i++ {
			errors[i] = true
		}
	}

	return errors
}

type mockMinimalDisputeGameFactoryCaller struct {
	gameCountErr bool
	indexErrors  []bool
	gameCount    uint64
	games        []types.GameMetadata
}

func newMockMinimalDisputeGameFactoryCaller(count uint64, gameCountErr bool, indexErrors bool) *mockMinimalDisputeGameFactoryCaller {
	return &mockMinimalDisputeGameFactoryCaller{
		indexErrors:  generateMockGameErrors(count, indexErrors),
		gameCountErr: gameCountErr,
		gameCount:    count,
		games:        generateMockGames(count),
	}
}

func (m *mockMinimalDisputeGameFactoryCaller) GetGameCount(_ context.Context, _ common.Hash) (uint64, error) {
	if m.gameCountErr {
		return 0, gameCountErr
	}

	return m.gameCount, nil
}

func (m *mockMinimalDisputeGameFactoryCaller) GetGame(_ context.Context, index uint64, _ common.Hash) (types.GameMetadata, error) {
	if m.indexErrors[index] {
		return struct {
			GameType  uint8
			Timestamp uint64
			Proxy     common.Address
		}{}, gameIndexErr
	}

	return m.games[index], nil
}
