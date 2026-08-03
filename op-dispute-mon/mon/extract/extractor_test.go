package extract

import (
	"context"
	"errors"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts"
	monTypes "github.com/ethereum-optimism/optimism/op-dispute-mon/mon/types"
	"github.com/ethereum-optimism/optimism/op-service/clock"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching/rpcblock"
	"github.com/stretchr/testify/require"

	faultTypes "github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

var (
	mockRootClaim = common.HexToHash("0x1234")
	ignoredGames  = []common.Address{common.HexToAddress("0xdeadbeef")}
)

func TestNewExtractorRequiresDependencies(t *testing.T) {
	cl := clock.NewDeterministicClock(time.Unix(1, 0))
	creator := func(context.Context, gameTypes.GameMetadata) (GameCaller, error) { return nil, nil }
	gameFetcher := func(context.Context, common.Hash, uint64) ([]gameTypes.GameMetadata, error) { return nil, nil }
	parentFetcher := func(context.Context, uint64, rpcblock.Block) (gameTypes.GameStatus, error) {
		return gameTypes.GameStatusDefenderWon, nil
	}
	newExtractor := func(
		cl clock.Clock,
		creator CreateGameCaller,
		games FactoryGameFetcher,
		parent ParentGameStatusFetcher,
		bond BondEnricher,
		agreement ZKEnricher,
		maxConcurrency uint,
	) error {
		_, err := NewExtractor(
			testlog.Logger(t, log.LvlDebug),
			cl,
			creator,
			games,
			parent,
			nil,
			maxConcurrency,
			nil,
			nil,
			bond,
			agreement,
		)
		return err
	}
	bond := &recordingBondEnricher{}
	agreement := &recordingZKEnricher{}
	require.ErrorContains(t, newExtractor(nil, creator, gameFetcher, parentFetcher, bond, agreement, 1), "clock is required")
	require.ErrorContains(t, newExtractor(cl, nil, gameFetcher, parentFetcher, bond, agreement, 1), "game caller creator is required")
	require.ErrorContains(t, newExtractor(cl, creator, nil, parentFetcher, bond, agreement, 1), "game fetcher is required")
	require.ErrorContains(t, newExtractor(cl, creator, gameFetcher, nil, bond, agreement, 1), "parent game status fetcher is required")
	require.ErrorContains(t, newExtractor(cl, creator, gameFetcher, parentFetcher, bond, nil, 1), "ZK agreement enricher is required")
	require.ErrorContains(t, newExtractor(cl, creator, gameFetcher, parentFetcher, nil, agreement, 1), "bond enricher is required")
	require.ErrorContains(t, newExtractor(cl, creator, gameFetcher, parentFetcher, bond, agreement, 0), "max concurrency must be greater than zero")
	require.NoError(t, newExtractor(cl, creator, gameFetcher, parentFetcher, bond, agreement, 1))
}

func TestExtractor_Extract(t *testing.T) {
	t.Run("FetchGamesError", func(t *testing.T) {
		extractor, _, games, _, _ := setupExtractorTest(t)
		games.err = errors.New("boom")
		_, _, _, err := extractor.Extract(context.Background(), common.Hash{}, 0)
		require.ErrorIs(t, err, games.err)
		require.Equal(t, 1, games.calls)
	})

	t.Run("CreateGameErrorLog", func(t *testing.T) {
		extractor, creator, games, logs, _ := setupExtractorTest(t)
		games.games = []gameTypes.GameMetadata{{}}
		creator.err = errors.New("boom")
		enriched, ignored, failed, err := extractor.Extract(context.Background(), common.Hash{}, 0)
		require.NoError(t, err)
		require.Equal(t, 1, failed)
		require.Zero(t, ignored)
		require.Len(t, enriched, 0)
		require.Equal(t, 1, games.calls)
		require.Equal(t, 1, creator.calls)
		require.Equal(t, 0, creator.caller.metadataCalls)
		require.Equal(t, 0, creator.caller.claimsCalls)
		verifyLogs(t, logs, 1, 0, 0, 0)
	})

	t.Run("MetadataFetchErrorLog", func(t *testing.T) {
		extractor, creator, games, logs, _ := setupExtractorTest(t)
		games.games = []gameTypes.GameMetadata{{}}
		creator.caller.metadataErr = errors.New("boom")
		enriched, ignored, failed, err := extractor.Extract(context.Background(), common.Hash{}, 0)
		require.NoError(t, err)
		require.Zero(t, ignored)
		require.Equal(t, 1, failed)
		require.Len(t, enriched, 0)
		require.Equal(t, 1, games.calls)
		require.Equal(t, 1, creator.calls)
		require.Equal(t, 1, creator.caller.metadataCalls)
		require.Equal(t, 0, creator.caller.claimsCalls)
		verifyLogs(t, logs, 0, 1, 0, 0)
	})

	t.Run("ClaimsFetchErrorLog", func(t *testing.T) {
		extractor, creator, games, logs, _ := setupExtractorTest(t)
		games.games = []gameTypes.GameMetadata{{}}
		creator.caller.claimsErr = errors.New("boom")
		enriched, ignored, failed, err := extractor.Extract(context.Background(), common.Hash{}, 0)
		require.NoError(t, err)
		require.Zero(t, ignored)
		require.Equal(t, 1, failed)
		require.Len(t, enriched, 0)
		require.Equal(t, 1, games.calls)
		require.Equal(t, 1, creator.calls)
		require.Equal(t, 1, creator.caller.metadataCalls)
		require.Equal(t, 1, creator.caller.claimsCalls)
		verifyLogs(t, logs, 0, 0, 1, 0)
	})

	t.Run("Success", func(t *testing.T) {
		extractor, creator, games, _, _ := setupExtractorTest(t)
		games.games = []gameTypes.GameMetadata{{}}
		enriched, ignored, failed, err := extractor.Extract(context.Background(), common.Hash{}, 0)
		require.NoError(t, err)
		require.Zero(t, ignored)
		require.Zero(t, failed)
		require.Len(t, enriched, 1)
		require.Equal(t, 1, games.calls)
		require.Equal(t, 1, creator.calls)
		require.Equal(t, 1, creator.caller.metadataCalls)
		require.Equal(t, 1, creator.caller.claimsCalls)
	})

	t.Run("EnricherFails", func(t *testing.T) {
		enricher := &mockEnricher{err: errors.New("whoops")}
		extractor, _, games, logs, _ := setupExtractorTest(t, enricher)
		games.games = []gameTypes.GameMetadata{{}}
		enriched, ignored, failed, err := extractor.Extract(context.Background(), common.Hash{}, 0)
		require.NoError(t, err)
		require.Zero(t, ignored)
		require.Equal(t, 1, failed)
		l := logs.FindLogs(testlog.NewAttributesContainsFilter("err", "failed to enrich game"))
		require.Len(t, l, 1, "Should have logged error")
		require.Len(t, enriched, 0, "Should not return games that failed to enrich")
	})

	t.Run("EnricherSuccess", func(t *testing.T) {
		enricher := &mockEnricher{}
		extractor, _, games, _, _ := setupExtractorTest(t, enricher)
		games.games = []gameTypes.GameMetadata{{}}
		enriched, ignored, failed, err := extractor.Extract(context.Background(), common.Hash{}, 0)
		require.NoError(t, err)
		require.Zero(t, ignored)
		require.Zero(t, failed)
		require.Len(t, enriched, 1)
		require.Equal(t, 1, enricher.calls)
	})

	t.Run("MultipleEnrichersMultipleGames", func(t *testing.T) {
		enricher1 := &mockEnricher{}
		enricher2 := &mockEnricher{}
		extractor, _, games, _, _ := setupExtractorTest(t, enricher1, enricher2)
		games.games = []gameTypes.GameMetadata{{Proxy: common.Address{0xaa}}, {Proxy: common.Address{0xbb}}}
		enriched, ignored, failed, err := extractor.Extract(context.Background(), common.Hash{}, 0)
		require.NoError(t, err)
		require.Zero(t, ignored)
		require.Zero(t, failed)
		require.Len(t, enriched, 2)
		require.Equal(t, 2, enricher1.calls)
		require.Equal(t, 2, enricher2.calls)
	})

	t.Run("IgnoreGames", func(t *testing.T) {
		enricher1 := &mockEnricher{}
		extractor, _, games, logs, _ := setupExtractorTest(t, enricher1)
		// Two games, one of which is ignored
		games.games = []gameTypes.GameMetadata{{Proxy: ignoredGames[0]}, {Proxy: common.Address{0xaa}}}
		enriched, ignored, failed, err := extractor.Extract(context.Background(), common.Hash{}, 0)
		require.NoError(t, err)
		// Should ignore one and enrich the other
		require.Equal(t, 1, ignored)
		require.Zero(t, failed)
		require.Len(t, enriched, 1)
		require.Equal(t, 1, enricher1.calls)
		require.Equal(t, enriched[0].Common().Proxy, common.Address{0xaa})
		require.NotNil(t, logs.FindLog(
			testlog.NewLevelFilter(log.LevelWarn),
			testlog.NewMessageFilter("Ignoring game"),
			testlog.NewAttributesFilter("game", ignoredGames[0].Hex())))
	})

	t.Run("UseCachedValueOnFailure", func(t *testing.T) {
		enricher := &mockEnricher{
			action: func(game *monTypes.CommonGameData) error {
				game.Status = gameTypes.GameStatusDefenderWon
				return nil
			},
		}
		extractor, _, games, _, cl := setupExtractorTest(t, enricher)
		gameA := common.Address{0xaa}
		gameB := common.Address{0xbb}
		games.games = []gameTypes.GameMetadata{{Proxy: gameA}, {Proxy: gameB}}

		// First fetch succeeds and the results should be cached
		enriched, ignored, failed, err := extractor.Extract(context.Background(), common.Hash{}, 0)
		require.NoError(t, err)
		require.Zero(t, ignored)
		require.Zero(t, failed)
		require.Len(t, enriched, 2)
		require.Equal(t, 2, enricher.calls)
		firstUpdateTime := cl.Now()
		// All results should have current LastUpdateTime
		for _, data := range enriched {
			require.Equal(t, firstUpdateTime, data.Common().LastUpdateTime)
		}

		cl.AdvanceTime(2 * time.Minute)
		secondUpdateTime := cl.Now()
		enricher.action = func(game *monTypes.CommonGameData) error {
			if game.Proxy == gameA {
				return errors.New("boom")
			}
			// Updated games will have a different status
			game.Status = gameTypes.GameStatusChallengerWon
			return nil
		}
		// Second fetch fails for one of the two games, it's cached value should be used.
		enriched, ignored, failed, err = extractor.Extract(context.Background(), common.Hash{}, 0)
		require.NoError(t, err)
		require.Zero(t, ignored)
		require.Equal(t, 1, failed)
		require.Len(t, enriched, 2)
		require.Equal(t, 4, enricher.calls)
		// The returned games are not in a fixed order, create a map to look up the game we need to assert
		actual := make(map[common.Address]*monTypes.CommonGameData)
		for _, data := range enriched {
			actual[data.Common().Proxy] = data.Common()
		}
		require.Contains(t, actual, gameA)
		require.Contains(t, actual, gameB)
		require.Equal(t, actual[gameA].Status, gameTypes.GameStatusDefenderWon)   // Uses cached value from game A
		require.Equal(t, actual[gameB].Status, gameTypes.GameStatusChallengerWon) // Updates game B
		require.Equal(t, firstUpdateTime, actual[gameA].LastUpdateTime)
		require.Equal(t, secondUpdateTime, actual[gameB].LastUpdateTime)
	})
}

func TestExtractorRejectsCallerMissingVariantCapability(t *testing.T) {
	tests := []struct {
		name     string
		gameType gameTypes.GameType
		wantErr  string
	}{
		{name: "fault", gameType: gameTypes.CannonGameType, wantErr: "does not support fault game extraction"},
		{name: "ZK", gameType: gameTypes.ZKDisputeGameType, wantErr: "does not support ZK game extraction"},
		{name: "SuperPermissioned", gameType: gameTypes.SuperPermissionedGameType, wantErr: "does not support common game extraction"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			extractor, _, _, _, _ := setupExtractorTest(t)
			extractor.createContract = func(context.Context, gameTypes.GameMetadata) (GameCaller, error) {
				return anchorOnlyGameCaller{}, nil
			}
			_, err := extractor.enrichGame(context.Background(), common.Hash{}, gameTypes.GameMetadata{GameType: uint32(test.gameType)})
			require.ErrorContains(t, err, test.wantErr)
		})
	}
}

func TestExtractorRejectsUnsupportedGameTypeBeforeVariantDispatch(t *testing.T) {
	extractor, _, _, _, _ := setupExtractorTest(t)

	_, err := extractor.enrichGame(context.Background(), common.Hash{}, gameTypes.GameMetadata{GameType: 99})
	require.EqualError(t, err, "unsupported game type: 99")
}

type anchorOnlyGameCaller struct{}

func (anchorOnlyGameCaller) GetAnchorStateRegistry(context.Context, rpcblock.Block) (common.Address, error) {
	return common.Address{}, nil
}

func verifyLogs(t *testing.T, logs *testlog.CapturingHandler, createErr, metadataErr, claimsErr, durationErr int) {
	errorLevelFilter := testlog.NewLevelFilter(log.LevelError)
	createMessageFilter := testlog.NewAttributesContainsFilter("err", "failed to create contracts")
	l := logs.FindLogs(errorLevelFilter, createMessageFilter)
	require.Len(t, l, createErr)
	fetchMessageFilter := testlog.NewAttributesContainsFilter("err", "failed to fetch game metadata")
	l = logs.FindLogs(errorLevelFilter, fetchMessageFilter)
	require.Len(t, l, metadataErr)
	claimsMessageFilter := testlog.NewAttributesContainsFilter("err", "failed to fetch game claims")
	l = logs.FindLogs(errorLevelFilter, claimsMessageFilter)
	require.Len(t, l, claimsErr)
	durationMessageFilter := testlog.NewAttributesContainsFilter("err", "failed to fetch game duration")
	l = logs.FindLogs(errorLevelFilter, durationMessageFilter)
	require.Len(t, l, durationErr)
}

func setupExtractorTest(t *testing.T, enrichers ...CommonEnricher) (*Extractor, *mockGameCallerCreator, *mockGameFetcher, *testlog.CapturingHandler, *clock.DeterministicClock) {
	logger, capturedLogs := testlog.CaptureLogger(t, log.LvlDebug)
	games := &mockGameFetcher{}
	caller := &mockGameCaller{rootClaim: mockRootClaim}
	creator := &mockGameCallerCreator{caller: caller}
	cl := clock.NewDeterministicClock(time.Unix(48294294, 58))
	extractor, err := NewExtractor(
		logger,
		cl,
		creator.CreateGameCaller,
		games.FetchGames,
		func(context.Context, uint64, rpcblock.Block) (gameTypes.GameStatus, error) {
			return gameTypes.GameStatusDefenderWon, nil
		},
		ignoredGames,
		5,
		enrichers,
		nil,
		&recordingBondEnricher{},
		&recordingZKEnricher{},
	)
	require.NoError(t, err)
	return extractor, creator, games, capturedLogs, cl
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

func (m *mockGameCallerCreator) CreateGameCaller(_ context.Context, _ gameTypes.GameMetadata) (GameCaller, error) {
	m.calls++
	if m.err != nil {
		return nil, m.err
	}
	return m.caller, nil
}

type mockGameCaller struct {
	metadataCalls        int
	metadataErr          error
	claimsCalls          int
	claimsErr            error
	rootClaim            common.Hash
	claims               []faultTypes.Claim
	requestedCredits     []common.Address
	creditsErr           error
	credits              map[common.Address]*big.Int
	extraCredit          []*big.Int
	bondDistributionMode faultTypes.BondDistributionMode
	balanceErr           error
	balance              *big.Int
	delayDuration        time.Duration
	balanceAddr          common.Address
	withdrawalsCalls     int
	withdrawalsErr       error
	withdrawals          []*contracts.WithdrawalRequest
	resolvedCalls        int
	resolvedErr          error
	resolved             map[int]bool
	anchorStateRegistry  common.Address
	anchorStateRegErr    error
}

func (m *mockGameCaller) GetWithdrawals(_ context.Context, _ rpcblock.Block, _ ...common.Address) ([]*contracts.WithdrawalRequest, error) {
	m.withdrawalsCalls++
	if m.withdrawalsErr != nil {
		return nil, m.withdrawalsErr
	}
	if m.withdrawals != nil {
		return m.withdrawals, nil
	}
	return []*contracts.WithdrawalRequest{
		{
			Timestamp: big.NewInt(1),
			Amount:    big.NewInt(2),
		},
		{
			Timestamp: big.NewInt(3),
			Amount:    big.NewInt(4),
		},
	}, nil
}

func (m *mockGameCaller) GetExtendedMetadata(_ context.Context, _ rpcblock.Block) (contracts.GameMetadata, error) {
	m.metadataCalls++
	if m.metadataErr != nil {
		return contracts.GameMetadata{}, m.metadataErr
	}
	return contracts.GameMetadata{
		L1Head:    common.Hash{0xaa},
		RootClaim: mockRootClaim,
	}, nil
}

func (m *mockGameCaller) GetAnchorStateRegistry(_ context.Context, _ rpcblock.Block) (common.Address, error) {
	if m.anchorStateRegErr != nil {
		return common.Address{}, m.anchorStateRegErr
	}
	return m.anchorStateRegistry, nil
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

func (m *mockGameCaller) GetBondDistributionMode(_ context.Context, _ rpcblock.Block) (faultTypes.BondDistributionMode, error) {
	return m.bondDistributionMode, nil
}

func (m *mockGameCaller) GetBalanceAndDelay(_ context.Context, _ rpcblock.Block) (*big.Int, time.Duration, common.Address, error) {
	if m.balanceErr != nil {
		return nil, 0, common.Address{}, m.balanceErr
	}
	return m.balance, m.delayDuration, m.balanceAddr, nil
}

func (m *mockGameCaller) IsResolved(_ context.Context, _ rpcblock.Block, claims ...faultTypes.Claim) ([]bool, error) {
	m.resolvedCalls++
	if m.resolvedErr != nil {
		return nil, m.resolvedErr
	}
	resolved := make([]bool, len(claims))
	for i, claim := range claims {
		resolved[i] = m.resolved[claim.ContractIndex]
	}
	return resolved, nil
}

func TestExtractor_EnrichGameInitializesRollupEndpointErrorCount(t *testing.T) {
	extractor, _, games, _, _ := setupExtractorTest(t)
	games.games = []gameTypes.GameMetadata{{}}
	enriched, ignored, failed, err := extractor.Extract(context.Background(), common.Hash{}, 0)
	require.NoError(t, err)
	require.Zero(t, ignored)
	require.Zero(t, failed)
	require.Len(t, enriched, 1)
	require.Equal(t, 0, enriched[0].Common().NodeEndpointErrorCount, "NodeEndpointErrorCount should be initialized to 0")
}

func TestExtractor_EnrichGameInitializesRollupEndpointOutOfSyncCount(t *testing.T) {
	extractor, _, games, _, _ := setupExtractorTest(t)
	games.games = []gameTypes.GameMetadata{{}}
	enriched, ignored, failed, err := extractor.Extract(context.Background(), common.Hash{}, 0)
	require.NoError(t, err)
	require.Zero(t, ignored)
	require.Zero(t, failed)
	require.Len(t, enriched, 1)
	require.Equal(t, 0, enriched[0].Common().NodeEndpointOutOfSyncCount, "NodeEndpointOutOfSyncCount should be initialized to 0")
}

type mockEnricher struct {
	err    error
	calls  int
	action func(game *monTypes.CommonGameData) error
}

func (m *mockEnricher) Enrich(_ context.Context, _ rpcblock.Block, _ GameCaller, game *monTypes.CommonGameData) error {
	m.calls++
	if m.action != nil {
		return m.action(game)
	}
	return m.err
}
