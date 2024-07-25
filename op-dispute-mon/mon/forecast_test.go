package mon

import (
	"math"
	"math/big"
	"testing"

	faultTypes "github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	"github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-dispute-mon/metrics"
	monTypes "github.com/ethereum-optimism/optimism/op-dispute-mon/mon/types"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

var (
	mockRootClaim       = common.Hash{0x11}
	failedForecastLog   = "Failed to forecast game"
	lostGameLog         = "Unexpected game result"
	unexpectedResultLog = "Forecasting unexpected game result"
	expectedResultLog   = "Forecasting expected game result"
)

func TestForecast_Forecast_BasicTests(t *testing.T) {
	t.Parallel()

	t.Run("NoGames", func(t *testing.T) {
		forecast, _, logs := setupForecastTest(t)
		forecast.Forecast([]*monTypes.EnrichedGameData{}, 0, 0)
		levelFilter := testlog.NewLevelFilter(log.LevelError)
		messageFilter := testlog.NewMessageFilter(failedForecastLog)
		require.Nil(t, logs.FindLog(levelFilter, messageFilter))
	})

	t.Run("ChallengerWonGame_Agree", func(t *testing.T) {
		forecast, m, logs := setupForecastTest(t)
		expectedGame := monTypes.EnrichedGameData{Status: types.GameStatusChallengerWon, RootClaim: mockRootClaim, AgreeWithClaim: true}
		forecast.Forecast([]*monTypes.EnrichedGameData{&expectedGame}, 0, 0)
		l := logs.FindLog(testlog.NewLevelFilter(log.LevelError), testlog.NewMessageFilter(lostGameLog))
		require.NotNil(t, l)
		require.Equal(t, expectedGame.Proxy, l.AttrValue("game"))
		require.Equal(t, types.GameStatusDefenderWon, l.AttrValue("expectedResult"))
		require.Equal(t, types.GameStatusChallengerWon, l.AttrValue("actualResult"))

		expectedMetrics := zeroGameAgreement()
		expectedMetrics[metrics.AgreeChallengerWins] = 1
		require.Equal(t, expectedMetrics, m.gameAgreement)
	})

	t.Run("ChallengerWonGame_Disagree", func(t *testing.T) {
		forecast, m, logs := setupForecastTest(t)
		expectedGame := monTypes.EnrichedGameData{Status: types.GameStatusChallengerWon, RootClaim: common.Hash{0xbb}, AgreeWithClaim: false}
		forecast.Forecast([]*monTypes.EnrichedGameData{&expectedGame}, 0, 0)
		l := logs.FindLog(testlog.NewLevelFilter(log.LevelError), testlog.NewMessageFilter(lostGameLog))
		require.Nil(t, l)

		expectedMetrics := zeroGameAgreement()
		expectedMetrics[metrics.DisagreeChallengerWins] = 1
		require.Equal(t, expectedMetrics, m.gameAgreement)
	})

	t.Run("DefenderWonGame_Agree", func(t *testing.T) {
		forecast, m, logs := setupForecastTest(t)
		expectedGame := monTypes.EnrichedGameData{Status: types.GameStatusDefenderWon, RootClaim: mockRootClaim, AgreeWithClaim: true}
		forecast.Forecast([]*monTypes.EnrichedGameData{&expectedGame}, 0, 0)
		l := logs.FindLog(testlog.NewLevelFilter(log.LevelError), testlog.NewMessageFilter(lostGameLog))
		require.Nil(t, l)

		expectedMetrics := zeroGameAgreement()
		expectedMetrics[metrics.AgreeDefenderWins] = 1
		require.Equal(t, expectedMetrics, m.gameAgreement)
	})

	t.Run("DefenderWonGame_Disagree", func(t *testing.T) {
		forecast, m, logs := setupForecastTest(t)
		expectedGame := monTypes.EnrichedGameData{Status: types.GameStatusDefenderWon, RootClaim: common.Hash{0xbb}, AgreeWithClaim: false}
		forecast.Forecast([]*monTypes.EnrichedGameData{&expectedGame}, 0, 0)
		l := logs.FindLog(testlog.NewLevelFilter(log.LevelError), testlog.NewMessageFilter(lostGameLog))
		require.NotNil(t, l)
		require.Equal(t, expectedGame.Proxy, l.AttrValue("game"))
		require.Equal(t, types.GameStatusChallengerWon, l.AttrValue("expectedResult"))
		require.Equal(t, types.GameStatusDefenderWon, l.AttrValue("actualResult"))

		expectedMetrics := zeroGameAgreement()
		expectedMetrics[metrics.DisagreeDefenderWins] = 1
		require.Equal(t, expectedMetrics, m.gameAgreement)
	})

	t.Run("SingleGame", func(t *testing.T) {
		forecast, _, logs := setupForecastTest(t)
		forecast.Forecast([]*monTypes.EnrichedGameData{{}}, 0, 0)
		require.Nil(t, logs.FindLog(testlog.NewLevelFilter(log.LevelError), testlog.NewMessageFilter(failedForecastLog)))
	})

	t.Run("MultipleGames", func(t *testing.T) {
		forecast, _, logs := setupForecastTest(t)
		forecast.Forecast([]*monTypes.EnrichedGameData{{}, {}, {}}, 0, 0)
		require.Nil(t, logs.FindLog(testlog.NewLevelFilter(log.LevelError), testlog.NewMessageFilter(failedForecastLog)))
	})
}

func TestForecast_Forecast_EndLogs(t *testing.T) {
	t.Parallel()

	t.Run("BlockNumberChallenged_AgreeWithChallenge", func(t *testing.T) {
		forecast, m, logs := setupForecastTest(t)
		expectedGame := monTypes.EnrichedGameData{
			Status:                types.GameStatusInProgress,
			BlockNumberChallenged: true,
			L2BlockNumber:         6,
			AgreeWithClaim:        false,
		}
		forecast.Forecast([]*monTypes.EnrichedGameData{&expectedGame}, 0, 0)
		l := logs.FindLog(testlog.NewLevelFilter(log.LevelDebug), testlog.NewMessageFilter("Found game with challenged block number"))
		require.NotNil(t, l)
		require.Equal(t, expectedGame.Proxy, l.AttrValue("game"))
		require.Equal(t, expectedGame.L2BlockNumber, l.AttrValue("blockNum"))
		require.Equal(t, false, l.AttrValue("agreement"))

		expectedMetrics := zeroGameAgreement()
		// We disagree with the root claim and the challenger is ahead
		expectedMetrics[metrics.DisagreeChallengerAhead] = 1
		require.Equal(t, expectedMetrics, m.gameAgreement)
	})

	t.Run("BlockNumberChallenged_DisagreeWithChallenge", func(t *testing.T) {
		forecast, m, logs := setupForecastTest(t)
		expectedGame := monTypes.EnrichedGameData{
			Status:                types.GameStatusInProgress,
			BlockNumberChallenged: true,
			L2BlockNumber:         6,
			AgreeWithClaim:        true,
		}
		forecast.Forecast([]*monTypes.EnrichedGameData{&expectedGame}, 0, 0)
		l := logs.FindLog(testlog.NewLevelFilter(log.LevelDebug), testlog.NewMessageFilter("Found game with challenged block number"))
		require.NotNil(t, l)
		require.Equal(t, expectedGame.Proxy, l.AttrValue("game"))
		require.Equal(t, expectedGame.L2BlockNumber, l.AttrValue("blockNum"))
		require.Equal(t, true, l.AttrValue("agreement"))

		expectedMetrics := zeroGameAgreement()
		// We agree with the root claim and the challenger is ahead
		expectedMetrics[metrics.AgreeChallengerAhead] = 1
		require.Equal(t, expectedMetrics, m.gameAgreement)
	})

	t.Run("AgreeDefenderWins", func(t *testing.T) {
		forecast, _, logs := setupForecastTest(t)
		games := []*monTypes.EnrichedGameData{{
			Status:            types.GameStatusInProgress,
			RootClaim:         mockRootClaim,
			Claims:            createDeepClaimList()[:1],
			AgreeWithClaim:    true,
			ExpectedRootClaim: mockRootClaim,
		}}
		forecast.Forecast(games, 0, 0)
		levelFilter := testlog.NewLevelFilter(log.LevelError)
		messageFilter := testlog.NewMessageFilter(failedForecastLog)
		require.Nil(t, logs.FindLog(levelFilter, messageFilter))
		levelFilter = testlog.NewLevelFilter(log.LevelDebug)
		messageFilter = testlog.NewMessageFilter(expectedResultLog)
		l := logs.FindLog(levelFilter, messageFilter)
		require.NotNil(t, l)
		require.Equal(t, mockRootClaim, l.AttrValue("rootClaim"))
		require.Equal(t, mockRootClaim, l.AttrValue("expected"))
		require.Equal(t, types.GameStatusDefenderWon, l.AttrValue("status"))
	})

	t.Run("AgreeChallengerWins", func(t *testing.T) {
		forecast, _, logs := setupForecastTest(t)
		games := []*monTypes.EnrichedGameData{{
			Status:            types.GameStatusInProgress,
			RootClaim:         mockRootClaim,
			Claims:            createDeepClaimList()[:2],
			AgreeWithClaim:    true,
			ExpectedRootClaim: mockRootClaim,
		}}
		forecast.Forecast(games, 0, 0)
		levelFilter := testlog.NewLevelFilter(log.LevelError)
		messageFilter := testlog.NewMessageFilter(failedForecastLog)
		require.Nil(t, logs.FindLog(levelFilter, messageFilter))
		levelFilter = testlog.NewLevelFilter(log.LevelWarn)
		messageFilter = testlog.NewMessageFilter(unexpectedResultLog)
		l := logs.FindLog(levelFilter, messageFilter)
		require.NotNil(t, l)
		require.Equal(t, mockRootClaim, l.AttrValue("rootClaim"))
		require.Equal(t, mockRootClaim, l.AttrValue("expected"))
		require.Equal(t, types.GameStatusChallengerWon, l.AttrValue("status"))
	})

	t.Run("DisagreeChallengerWins", func(t *testing.T) {
		forecast, _, logs := setupForecastTest(t)
		forecast.Forecast([]*monTypes.EnrichedGameData{{
			Status:            types.GameStatusInProgress,
			Claims:            createDeepClaimList()[:2],
			AgreeWithClaim:    false,
			ExpectedRootClaim: mockRootClaim,
		}}, 0, 0)
		levelFilter := testlog.NewLevelFilter(log.LevelError)
		messageFilter := testlog.NewMessageFilter(failedForecastLog)
		require.Nil(t, logs.FindLog(levelFilter, messageFilter))
		levelFilter = testlog.NewLevelFilter(log.LevelDebug)
		messageFilter = testlog.NewMessageFilter(expectedResultLog)
		l := logs.FindLog(levelFilter, messageFilter)
		require.NotNil(t, l)
		require.Equal(t, common.Hash{}, l.AttrValue("rootClaim"))
		require.Equal(t, mockRootClaim, l.AttrValue("expected"))
		require.Equal(t, types.GameStatusChallengerWon, l.AttrValue("status"))
	})

	t.Run("DisagreeDefenderWins", func(t *testing.T) {
		forecast, _, logs := setupForecastTest(t)
		forecast.Forecast([]*monTypes.EnrichedGameData{{
			Status:            types.GameStatusInProgress,
			Claims:            createDeepClaimList()[:1],
			AgreeWithClaim:    false,
			ExpectedRootClaim: mockRootClaim,
		}}, 0, 0)
		levelFilter := testlog.NewLevelFilter(log.LevelError)
		messageFilter := testlog.NewMessageFilter(failedForecastLog)
		require.Nil(t, logs.FindLog(levelFilter, messageFilter))
		levelFilter = testlog.NewLevelFilter(log.LevelWarn)
		messageFilter = testlog.NewMessageFilter(unexpectedResultLog)
		l := logs.FindLog(levelFilter, messageFilter)
		require.NotNil(t, l)
		require.Equal(t, common.Hash{}, l.AttrValue("rootClaim"))
		require.Equal(t, mockRootClaim, l.AttrValue("expected"))
		require.Equal(t, types.GameStatusDefenderWon, l.AttrValue("status"))
	})
}

func TestForecast_Forecast_MultipleGames(t *testing.T) {
	forecast, m, logs := setupForecastTest(t)
	gameStatus := []types.GameStatus{
		types.GameStatusChallengerWon,
		types.GameStatusInProgress,
		types.GameStatusInProgress,
		types.GameStatusDefenderWon,
		types.GameStatusInProgress,
		types.GameStatusInProgress,
		types.GameStatusDefenderWon,
		types.GameStatusChallengerWon,
		types.GameStatusChallengerWon,
	}
	claims := [][]monTypes.EnrichedClaim{
		createDeepClaimList()[:1],
		createDeepClaimList()[:2],
		createDeepClaimList()[:2],
		createDeepClaimList()[:1],
		createDeepClaimList()[:1],
		createDeepClaimList()[:1],
		createDeepClaimList()[:1],
		createDeepClaimList()[:1],
		createDeepClaimList()[:1],
	}
	rootClaims := []common.Hash{
		{},
		{},
		mockRootClaim,
		{},
		{},
		mockRootClaim,
		{},
		{},            // Expected latest invalid proposal (will have timestamp 7)
		mockRootClaim, // Expected latest valid proposal (will have timestamp 8)
	}
	games := make([]*monTypes.EnrichedGameData, 9)
	for i := range games {
		games[i] = &monTypes.EnrichedGameData{
			Status:        gameStatus[i],
			Claims:        claims[i],
			RootClaim:     rootClaims[i],
			L2BlockNumber: uint64(i),
			GameMetadata: types.GameMetadata{
				Timestamp: uint64(i),
			},
			AgreeWithClaim:    rootClaims[i] == mockRootClaim,
			ExpectedRootClaim: mockRootClaim,
		}
	}
	forecast.Forecast(games, 3, 4)
	require.Nil(t, logs.FindLog(testlog.NewLevelFilter(log.LevelError), testlog.NewMessageFilter(failedForecastLog)))
	expectedMetrics := zeroGameAgreement()
	expectedMetrics[metrics.AgreeChallengerAhead] = 1
	expectedMetrics[metrics.DisagreeChallengerAhead] = 1
	expectedMetrics[metrics.AgreeDefenderAhead] = 1
	expectedMetrics[metrics.DisagreeDefenderAhead] = 1
	expectedMetrics[metrics.AgreeChallengerWins] = 1
	expectedMetrics[metrics.DisagreeDefenderWins] = 2
	expectedMetrics[metrics.DisagreeChallengerWins] = 2
	require.Equal(t, expectedMetrics, m.gameAgreement)
	require.Equal(t, 3, m.ignoredGames)
	require.Equal(t, 4, m.contractCreationFails)
	require.EqualValues(t, 8, m.latestValidProposalL2Block)
	require.EqualValues(t, 7, m.latestInvalidProposal)
	require.EqualValues(t, 8, m.latestValidProposal)
}

func setupForecastTest(t *testing.T) (*Forecast, *mockForecastMetrics, *testlog.CapturingHandler) {
	logger, capturedLogs := testlog.CaptureLogger(t, log.LvlDebug)
	m := &mockForecastMetrics{
		gameAgreement: zeroGameAgreement(),
	}
	return NewForecast(logger, m), m, capturedLogs
}

func zeroGameAgreement() map[metrics.GameAgreementStatus]int {
	return map[metrics.GameAgreementStatus]int{
		metrics.AgreeChallengerAhead:    0,
		metrics.DisagreeChallengerAhead: 0,
		metrics.AgreeDefenderAhead:      0,
		metrics.DisagreeDefenderAhead:   0,
		metrics.AgreeDefenderWins:       0,
		metrics.DisagreeDefenderWins:    0,
		metrics.AgreeChallengerWins:     0,
		metrics.DisagreeChallengerWins:  0,
	}
}

type mockForecastMetrics struct {
	gameAgreement              map[metrics.GameAgreementStatus]int
	ignoredGames               int
	latestValidProposalL2Block uint64
	latestInvalidProposal      uint64
	latestValidProposal        uint64
	contractCreationFails      int
}

func (m *mockForecastMetrics) RecordFailedGames(count int) {
	m.contractCreationFails = count
}

func (m *mockForecastMetrics) RecordGameAgreement(status metrics.GameAgreementStatus, count int) {
	m.gameAgreement[status] = count
}

func (m *mockForecastMetrics) RecordLatestValidProposalL2Block(valid uint64) {
	m.latestValidProposalL2Block = valid
}

func (m *mockForecastMetrics) RecordLatestProposals(valid, invalid uint64) {
	m.latestValidProposal = valid
	m.latestInvalidProposal = invalid
}

func (m *mockForecastMetrics) RecordIgnoredGames(count int) {
	m.ignoredGames = count
}

func createDeepClaimList() []monTypes.EnrichedClaim {
	return []monTypes.EnrichedClaim{
		{
			Claim: faultTypes.Claim{
				ClaimData: faultTypes.ClaimData{
					Position: faultTypes.NewPosition(0, big.NewInt(0)),
				},
				ContractIndex:       0,
				ParentContractIndex: math.MaxInt64,
				Claimant:            common.HexToAddress("0x111111"),
			},
		},
		{
			Claim: faultTypes.Claim{
				ClaimData: faultTypes.ClaimData{
					Position: faultTypes.NewPosition(1, big.NewInt(0)),
				},
				ContractIndex:       1,
				ParentContractIndex: 0,
				Claimant:            common.HexToAddress("0x222222"),
			},
		},
		{
			Claim: faultTypes.Claim{
				ClaimData: faultTypes.ClaimData{
					Position: faultTypes.NewPosition(2, big.NewInt(0)),
				},
				ContractIndex:       2,
				ParentContractIndex: 1,
				Claimant:            common.HexToAddress("0x111111"),
			},
		},
	}
}
