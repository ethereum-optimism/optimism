package extract

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	faultTypes "github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	"github.com/ethereum-optimism/optimism/op-challenger/game/types"
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

var mockRootClaim = common.HexToHash("0x1234")

func TestExtractor_Extract(t *testing.T) {
	t.Run("FetchGamesError", func(t *testing.T) {
		extractor, _, games, _ := setupExtractorTest(t)
		games.err = errors.New("boom")
		_, err := extractor.Extract(context.Background(), common.Hash{}, 0)
		require.ErrorIs(t, err, games.err)
		require.Equal(t, 1, games.calls)
	})

	t.Run("CreateGameErrorLog", func(t *testing.T) {
		extractor, creator, games, logs := setupExtractorTest(t)
		games.games = []gameTypes.GameMetadata{{}}
		creator.err = errors.New("boom")
		enriched, err := extractor.Extract(context.Background(), common.Hash{}, 0)
		require.NoError(t, err)
		require.Len(t, enriched, 0)
		require.Equal(t, 1, games.calls)
		require.Equal(t, 1, creator.calls)
		verifyLogs(t, logs, 1, 0)
	})

	t.Run("MetadataFetchErrorLog", func(t *testing.T) {
		extractor, creator, games, logs := setupExtractorTest(t)
		games.games = []gameTypes.GameMetadata{{}}
		creator.caller.err = errors.New("boom")
		enriched, err := extractor.Extract(context.Background(), common.Hash{}, 0)
		require.NoError(t, err)
		require.Len(t, enriched, 0)
		require.Equal(t, 1, games.calls)
		require.Equal(t, 1, creator.calls)
		require.Equal(t, 1, creator.caller.calls)
		verifyLogs(t, logs, 0, 1)
	})

	t.Run("Success", func(t *testing.T) {
		extractor, creator, games, _ := setupExtractorTest(t)
		games.games = []gameTypes.GameMetadata{{}}
		enriched, err := extractor.Extract(context.Background(), common.Hash{}, 0)
		require.NoError(t, err)
		require.Len(t, enriched, 1)
		require.Equal(t, 1, games.calls)
		require.Equal(t, 1, creator.calls)
		require.Equal(t, 1, creator.caller.calls)
	})
}

func verifyLogs(t *testing.T, logs *testlog.CapturingHandler, createErr int, metadataErr int) {
	errorLevelFilter := testlog.NewLevelFilter(log.LevelError)
	createMessageFilter := testlog.NewMessageFilter("failed to create game caller")
	l := logs.FindLogs(errorLevelFilter, createMessageFilter)
	require.Len(t, l, createErr)
	fetchMessageFilter := testlog.NewMessageFilter("failed to fetch game metadata")
	l = logs.FindLogs(errorLevelFilter, fetchMessageFilter)
	require.Len(t, l, metadataErr)
}

func setupExtractorTest(t *testing.T) (*Extractor, *mockGameCallerCreator, *mockGameFetcher, *testlog.CapturingHandler) {
	logger, capturedLogs := testlog.CaptureLogger(t, log.LvlDebug)
	games := &mockGameFetcher{}
	caller := &mockGameCaller{rootClaim: mockRootClaim}
	creator := &mockGameCallerCreator{caller: caller}
	return NewExtractor(
			logger,
			creator.CreateGameCaller,
			games.FetchGames,
		),
		creator,
		games,
		capturedLogs
}

type mockGameFetcher struct {
	calls int
	err   error
	games []gameTypes.GameMetadata
}

func (m *mockGameFetcher) FetchGames(_ context.Context, _ common.Hash, _ uint64) ([]gameTypes.GameMetadata, error) {
	m.calls++
	if m.err != nil {
		return nil, m.err
	}
	return m.games, nil
}

type mockGameCallerCreator struct {
	calls  int
	err    error
	caller *mockGameCaller
}

func (m *mockGameCallerCreator) CreateGameCaller(_ gameTypes.GameMetadata) (GameCaller, error) {
	m.calls++
	if m.err != nil {
		return nil, m.err
	}
	return m.caller, nil
}

type mockGameCaller struct {
	calls     int
	err       error
	rootClaim common.Hash
}

func (m *mockGameCaller) GetGameMetadata(_ context.Context) (uint64, common.Hash, types.GameStatus, error) {
	m.calls++
	if m.err != nil {
		return 0, common.Hash{}, 0, m.err
	}
	return 0, mockRootClaim, 0, nil
}

func (m *mockGameCaller) GetAllClaims(ctx context.Context) ([]faultTypes.Claim, error) {
	m.calls++
	if m.err != nil {
		return nil, m.err
	}
	return []faultTypes.Claim{{}}, nil
}
