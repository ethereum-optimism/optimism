package extract

import (
	"context"
	"errors"
	"math/big"
	"testing"

	monTypes "github.com/ethereum-optimism/optimism/op-dispute-mon/mon/types"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching/rpcblock"
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
		require.Equal(t, 0, creator.caller.metadataCalls)
		require.Equal(t, 0, creator.caller.claimsCalls)
		verifyLogs(t, logs, 1, 0, 0, 0)
	})

	t.Run("MetadataFetchErrorLog", func(t *testing.T) {
		extractor, creator, games, logs := setupExtractorTest(t)
		games.games = []gameTypes.GameMetadata{{}}
		creator.caller.metadataErr = errors.New("boom")
		enriched, err := extractor.Extract(context.Background(), common.Hash{}, 0)
		require.NoError(t, err)
		require.Len(t, enriched, 0)
		require.Equal(t, 1, games.calls)
		require.Equal(t, 1, creator.calls)
		require.Equal(t, 1, creator.caller.metadataCalls)
		require.Equal(t, 0, creator.caller.claimsCalls)
		verifyLogs(t, logs, 0, 1, 0, 0)
	})

	t.Run("ClaimsFetchErrorLog", func(t *testing.T) {
		extractor, creator, games, logs := setupExtractorTest(t)
		games.games = []gameTypes.GameMetadata{{}}
		creator.caller.claimsErr = errors.New("boom")
		enriched, err := extractor.Extract(context.Background(), common.Hash{}, 0)
		require.NoError(t, err)
		require.Len(t, enriched, 0)
		require.Equal(t, 1, games.calls)
		require.Equal(t, 1, creator.calls)
		require.Equal(t, 1, creator.caller.metadataCalls)
		require.Equal(t, 1, creator.caller.claimsCalls)
		verifyLogs(t, logs, 0, 0, 1, 0)
	})

	t.Run("Success", func(t *testing.T) {
		extractor, creator, games, _ := setupExtractorTest(t)
		games.games = []gameTypes.GameMetadata{{}}
		enriched, err := extractor.Extract(context.Background(), common.Hash{}, 0)
		require.NoError(t, err)
		require.Len(t, enriched, 1)
		require.Equal(t, 1, games.calls)
		require.Equal(t, 1, creator.calls)
		require.Equal(t, 1, creator.caller.metadataCalls)
		require.Equal(t, 1, creator.caller.claimsCalls)
	})

	t.Run("EnricherFails", func(t *testing.T) {
		enricher := &mockEnricher{err: errors.New("whoops")}
		extractor, _, games, logs := setupExtractorTest(t, enricher)
		games.games = []gameTypes.GameMetadata{{}}
		enriched, err := extractor.Extract(context.Background(), common.Hash{}, 0)
		require.NoError(t, err)
		l := logs.FindLogs(testlog.NewMessageFilter("Failed to enrich game"))
		require.Len(t, l, 1, "Should have logged error")
		require.Len(t, enriched, 0, "Should not return games that failed to enrich")
	})

	t.Run("EnricherSuccess", func(t *testing.T) {
		enricher := &mockEnricher{}
		extractor, _, games, _ := setupExtractorTest(t, enricher)
		games.games = []gameTypes.GameMetadata{{}}
		enriched, err := extractor.Extract(context.Background(), common.Hash{}, 0)
		require.NoError(t, err)
		require.Len(t, enriched, 1)
		require.Equal(t, 1, enricher.calls)
	})

	t.Run("MultipleEnrichersMultipleGames", func(t *testing.T) {
		enricher1 := &mockEnricher{}
		enricher2 := &mockEnricher{}
		extractor, _, games, _ := setupExtractorTest(t, enricher1, enricher2)
		games.games = []gameTypes.GameMetadata{{}, {}}
		enriched, err := extractor.Extract(context.Background(), common.Hash{}, 0)
		require.NoError(t, err)
		require.Len(t, enriched, 2)
		require.Equal(t, 2, enricher1.calls)
		require.Equal(t, 2, enricher2.calls)
	})
}

func verifyLogs(t *testing.T, logs *testlog.CapturingHandler, createErr int, metadataErr int, claimsErr int, durationErr int) {
	errorLevelFilter := testlog.NewLevelFilter(log.LevelError)
	createMessageFilter := testlog.NewMessageFilter("Failed to create game caller")
	l := logs.FindLogs(errorLevelFilter, createMessageFilter)
	require.Len(t, l, createErr)
	fetchMessageFilter := testlog.NewMessageFilter("Failed to fetch game metadata")
	l = logs.FindLogs(errorLevelFilter, fetchMessageFilter)
	require.Len(t, l, metadataErr)
	claimsMessageFilter := testlog.NewMessageFilter("Failed to fetch game claims")
	l = logs.FindLogs(errorLevelFilter, claimsMessageFilter)
	require.Len(t, l, claimsErr)
	durationMessageFilter := testlog.NewMessageFilter("Failed to fetch game duration")
	l = logs.FindLogs(errorLevelFilter, durationMessageFilter)
	require.Len(t, l, durationErr)
}

func setupExtractorTest(t *testing.T, enrichers ...Enricher) (*Extractor, *mockGameCallerCreator, *mockGameFetcher, *testlog.CapturingHandler) {
	logger, capturedLogs := testlog.CaptureLogger(t, log.LvlDebug)
	games := &mockGameFetcher{}
	caller := &mockGameCaller{rootClaim: mockRootClaim}
	creator := &mockGameCallerCreator{caller: caller}
	extractor := NewExtractor(
		logger,
		creator.CreateGameCaller,
		games.FetchGames,
		enrichers...,
	)
	return extractor, creator, games, capturedLogs
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
	metadataCalls    int
	metadataErr      error
	claimsCalls      int
	claimsErr        error
	rootClaim        common.Hash
	claims           []faultTypes.Claim
	requestedCredits []common.Address
	creditsErr       error
	credits          map[common.Address]*big.Int
	extraCredit      []*big.Int
	balanceErr       error
	balance          *big.Int
	balanceAddr      common.Address
}

func (m *mockGameCaller) GetGameMetadata(_ context.Context, _ rpcblock.Block) (common.Hash, uint64, common.Hash, types.GameStatus, uint64, error) {
	m.metadataCalls++
	if m.metadataErr != nil {
		return common.Hash{}, 0, common.Hash{}, 0, 0, m.metadataErr
	}
	return common.Hash{0xaa}, 0, mockRootClaim, 0, 0, nil
}

func (m *mockGameCaller) GetAllClaims(_ context.Context, _ rpcblock.Block) ([]faultTypes.Claim, error) {
	m.claimsCalls++
	if m.claimsErr != nil {
		return nil, m.claimsErr
	}
	return m.claims, nil
}

func (m *mockGameCaller) GetCredits(_ context.Context, _ rpcblock.Block, recipients ...common.Address) ([]*big.Int, error) {
	m.requestedCredits = recipients
	if m.creditsErr != nil {
		return nil, m.creditsErr
	}
	response := make([]*big.Int, 0, len(recipients))
	for _, recipient := range recipients {
		credit, ok := m.credits[recipient]
		if !ok {
			credit = big.NewInt(0)
		}
		response = append(response, credit)
	}
	response = append(response, m.extraCredit...)
	return response, nil
}

func (m *mockGameCaller) GetBalance(_ context.Context, _ rpcblock.Block) (*big.Int, common.Address, error) {
	if m.balanceErr != nil {
		return nil, common.Address{}, m.balanceErr
	}
	return m.balance, m.balanceAddr, nil
}

type mockEnricher struct {
	err   error
	calls int
}

func (m *mockEnricher) Enrich(_ context.Context, _ rpcblock.Block, _ GameCaller, _ *monTypes.EnrichedGameData) error {
	m.calls++
	return m.err
}
