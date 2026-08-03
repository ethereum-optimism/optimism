package mon

import (
	"math"
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts"
	faultTypes "github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-dispute-mon/metrics"
	monTypes "github.com/ethereum-optimism/optimism/op-dispute-mon/mon/types"
	"github.com/ethereum-optimism/optimism/op-service/ptr"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

var mockRootClaim = common.Hash{0x11}

func TestAgreementStatusCoversCanonicalSeries(t *testing.T) {
	actual := make(map[metrics.GameAgreementStatus]bool)
	for _, agree := range []bool{false, true} {
		for _, result := range []gameTypes.GameStatus{gameTypes.GameStatusDefenderWon, gameTypes.GameStatusChallengerWon} {
			for _, inProgress := range []bool{false, true} {
				actual[agreementStatus(agree, result, inProgress)] = true
			}
		}
	}

	require.Equal(t, map[metrics.GameAgreementStatus]bool{
		metrics.AgreeChallengerAhead:    true,
		metrics.DisagreeChallengerAhead: true,
		metrics.AgreeDefenderAhead:      true,
		metrics.DisagreeDefenderAhead:   true,
		metrics.AgreeDefenderWins:       true,
		metrics.DisagreeDefenderWins:    true,
		metrics.AgreeChallengerWins:     true,
		metrics.DisagreeChallengerWins:  true,
	}, actual)
}

func TestForecastFaultGamesUnchanged(t *testing.T) {
	tests := []struct {
		name   string
		game   *monTypes.FaultGameData
		status metrics.GameAgreementStatus
	}{
		{
			name:   "agree challenger won",
			game:   faultGame(gameTypes.GameStatusChallengerWon, true),
			status: metrics.AgreeChallengerWins,
		},
		{
			name:   "disagree challenger won",
			game:   faultGame(gameTypes.GameStatusChallengerWon, false),
			status: metrics.DisagreeChallengerWins,
		},
		{
			name:   "agree defender won",
			game:   faultGame(gameTypes.GameStatusDefenderWon, true),
			status: metrics.AgreeDefenderWins,
		},
		{
			name:   "disagree defender won",
			game:   faultGame(gameTypes.GameStatusDefenderWon, false),
			status: metrics.DisagreeDefenderWins,
		},
		{
			name: "block number challenged",
			game: func() *monTypes.FaultGameData {
				game := faultGame(gameTypes.GameStatusInProgress, true)
				game.BlockNumberChallenged = true
				return game
			}(),
			status: metrics.AgreeChallengerAhead,
		},
		{
			name: "fault tree defender ahead",
			game: func() *monTypes.FaultGameData {
				game := faultGame(gameTypes.GameStatusInProgress, true)
				game.Claims = createDeepClaimList()[:1]
				return game
			}(),
			status: metrics.AgreeDefenderAhead,
		},
		{
			name: "fault tree challenger ahead",
			game: func() *monTypes.FaultGameData {
				game := faultGame(gameTypes.GameStatusInProgress, false)
				game.Claims = createDeepClaimList()[:2]
				return game
			}(),
			status: metrics.DisagreeChallengerAhead,
		},
		{
			name: "fault tree defender ahead while disagreeing",
			game: func() *monTypes.FaultGameData {
				game := faultGame(gameTypes.GameStatusInProgress, false)
				game.Claims = createDeepClaimList()[:1]
				return game
			}(),
			status: metrics.DisagreeDefenderAhead,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			forecast, metricer := setupForecastTest(t)
			forecast.Forecast([]monTypes.EnrichedGame{test.game}, 0, 0)
			require.Equal(t, map[metrics.GameAgreementStatus]int{test.status: 1}, metricer.gameAgreements)
		})
	}
}

func TestForecastSuperPermissionedGamesUnchanged(t *testing.T) {
	tests := []struct {
		name    string
		status  gameTypes.GameStatus
		agree   bool
		outcome metrics.GameAgreementStatus
	}{
		{"agree terminal", gameTypes.GameStatusDefenderWon, true, metrics.AgreeDefenderWins},
		{"disagree terminal", gameTypes.GameStatusDefenderWon, false, metrics.DisagreeDefenderWins},
		{"agree in progress", gameTypes.GameStatusInProgress, true, metrics.AgreeChallengerAhead},
		{"disagree in progress", gameTypes.GameStatusInProgress, false, metrics.DisagreeDefenderAhead},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			forecast, metricer := setupForecastTest(t)
			game := &monTypes.SuperPermissionedGameData{CommonGameData: commonGame(
				gameTypes.SuperPermissionedGameType, test.status, test.agree,
			)}
			forecast.Forecast([]monTypes.EnrichedGame{game}, 0, 0)
			require.Equal(t, map[metrics.GameAgreementStatus]int{test.outcome: 1}, metricer.gameAgreements)
		})
	}
}

func TestForecastZKInProgressDecisionTable(t *testing.T) {
	expectedBuckets := map[bool]map[gameTypes.GameStatus]metrics.GameAgreementStatus{
		true: {
			gameTypes.GameStatusDefenderWon:   metrics.AgreeDefenderAhead,
			gameTypes.GameStatusChallengerWon: metrics.AgreeChallengerAhead,
		},
		false: {
			gameTypes.GameStatusDefenderWon:   metrics.DisagreeDefenderAhead,
			gameTypes.GameStatusChallengerWon: metrics.DisagreeChallengerAhead,
		},
	}
	proposalResults := map[contracts.ProposalStatus]gameTypes.GameStatus{
		contracts.ProposalStatusUnchallenged:                      gameTypes.GameStatusDefenderWon,
		contracts.ProposalStatusChallenged:                        gameTypes.GameStatusChallengerWon,
		contracts.ProposalStatusUnchallengedAndValidProofProvided: gameTypes.GameStatusDefenderWon,
		contracts.ProposalStatusChallengedAndValidProofProvided:   gameTypes.GameStatusDefenderWon,
	}
	parentStates := []struct {
		name   string
		status *gameTypes.GameStatus
	}{
		{"root", nil},
		{"parent in progress", ptr.New(gameTypes.GameStatusInProgress)},
		{"parent defender won", ptr.New(gameTypes.GameStatusDefenderWon)},
		{"parent challenger won", ptr.New(gameTypes.GameStatusChallengerWon)},
	}

	for proposal, proposalResult := range proposalResults {
		for _, parent := range parentStates {
			for _, agree := range []bool{false, true} {
				name := proposal.String() + "/" + parent.name
				if agree {
					name += "/agree"
				} else {
					name += "/disagree"
				}
				t.Run(name, func(t *testing.T) {
					forecast, metricer := setupForecastTest(t)
					game := &monTypes.ZKGameData{
						CommonGameData: commonGame(gameTypes.ZKDisputeGameType, gameTypes.GameStatusInProgress, agree),
						ParentStatus:   parent.status,
						ProposalStatus: proposal,
					}
					actual := proposalResult
					if parent.status != nil && *parent.status == gameTypes.GameStatusChallengerWon {
						actual = gameTypes.GameStatusChallengerWon
					}
					forecast.Forecast([]monTypes.EnrichedGame{game}, 0, 0)
					status := expectedBuckets[agree][actual]
					require.Equal(t, map[metrics.GameAgreementStatus]int{status: 1}, metricer.gameAgreements)
				})
			}
		}
	}
}

func TestForecastZKTerminalUsesActualResult(t *testing.T) {
	expectedBuckets := map[bool]map[gameTypes.GameStatus]metrics.GameAgreementStatus{
		true: {
			gameTypes.GameStatusDefenderWon:   metrics.AgreeDefenderWins,
			gameTypes.GameStatusChallengerWon: metrics.AgreeChallengerWins,
		},
		false: {
			gameTypes.GameStatusDefenderWon:   metrics.DisagreeDefenderWins,
			gameTypes.GameStatusChallengerWon: metrics.DisagreeChallengerWins,
		},
	}
	parentStates := []struct {
		name   string
		status *gameTypes.GameStatus
	}{
		{"root", nil},
		{"parent in progress", ptr.New(gameTypes.GameStatusInProgress)},
		{"parent defender won", ptr.New(gameTypes.GameStatusDefenderWon)},
		{"parent challenger won", ptr.New(gameTypes.GameStatusChallengerWon)},
	}
	for _, status := range []gameTypes.GameStatus{gameTypes.GameStatusDefenderWon, gameTypes.GameStatusChallengerWon} {
		for _, parent := range parentStates {
			for _, agree := range []bool{false, true} {
				name := status.String() + "/" + parent.name
				if agree {
					name += "/agree"
				} else {
					name += "/disagree"
				}
				t.Run(name, func(t *testing.T) {
					forecast, metricer := setupForecastTest(t)
					game := &monTypes.ZKGameData{
						CommonGameData: commonGame(gameTypes.ZKDisputeGameType, status, agree),
						ParentStatus:   parent.status,
						ProposalStatus: contracts.ProposalStatusResolved,
					}
					forecast.Forecast([]monTypes.EnrichedGame{game}, 0, 0)
					agreement := expectedBuckets[agree][status]
					require.Equal(t, map[metrics.GameAgreementStatus]int{agreement: 1}, metricer.gameAgreements)
				})
			}
		}
	}
}

func TestForecastZKParentResolutionTransition(t *testing.T) {
	forecast, metricer := setupForecastTest(t)
	game := &monTypes.ZKGameData{
		CommonGameData: commonGame(gameTypes.ZKDisputeGameType, gameTypes.GameStatusInProgress, true),
		ParentStatus:   ptr.New(gameTypes.GameStatusInProgress),
		ProposalStatus: contracts.ProposalStatusUnchallenged,
	}

	forecast.Forecast([]monTypes.EnrichedGame{game}, 0, 0)
	require.Equal(t, map[metrics.GameAgreementStatus]int{
		metrics.AgreeDefenderAhead: 1,
	}, metricer.gameAgreements)

	game.ParentStatus = ptr.New(gameTypes.GameStatusChallengerWon)
	forecast.Forecast([]monTypes.EnrichedGame{game}, 0, 0)
	require.Equal(t, map[metrics.GameAgreementStatus]int{
		metrics.AgreeChallengerAhead: 1,
	}, metricer.gameAgreements)

	game.Status = gameTypes.GameStatusChallengerWon
	game.ProposalStatus = contracts.ProposalStatusResolved
	forecast.Forecast([]monTypes.EnrichedGame{game}, 0, 0)
	require.Equal(t, map[metrics.GameAgreementStatus]int{
		metrics.AgreeChallengerWins: 1,
	}, metricer.gameAgreements)
}

func TestForecastZKInvalidParentOverridesCanonicalChild(t *testing.T) {
	forecast, metricer := setupForecastTest(t)
	game := &monTypes.ZKGameData{
		CommonGameData: commonGame(gameTypes.ZKDisputeGameType, gameTypes.GameStatusInProgress, true),
		ParentStatus:   ptr.New(gameTypes.GameStatusChallengerWon),
		ProposalStatus: contracts.ProposalStatusUnchallenged,
	}
	forecast.Forecast([]monTypes.EnrichedGame{game}, 0, 0)
	require.Equal(t, map[metrics.GameAgreementStatus]int{
		metrics.AgreeChallengerAhead: 1,
	}, metricer.gameAgreements)
}

func TestForecastLatestProposalMetrics(t *testing.T) {
	forecast, metricer := setupForecastTest(t)
	fault := faultGame(gameTypes.GameStatusInProgress, true)
	fault.Timestamp = 3
	fault.L2SequenceNumber = 100
	zk := &monTypes.ZKGameData{
		CommonGameData: commonGame(gameTypes.ZKDisputeGameType, gameTypes.GameStatusInProgress, true),
		ProposalStatus: contracts.ProposalStatusUnchallenged,
	}
	zk.Timestamp = 5
	zk.L2SequenceNumber = 999
	superPermissioned := &monTypes.SuperPermissionedGameData{
		CommonGameData: commonGame(gameTypes.SuperPermissionedGameType, gameTypes.GameStatusDefenderWon, true),
	}
	superPermissioned.Timestamp = 6
	superPermissioned.L2SequenceNumber = 1_700_000_000
	superCannon := faultGame(gameTypes.GameStatusDefenderWon, true)
	superCannon.GameType = uint32(gameTypes.SuperCannonKonaGameType)
	superCannon.Timestamp = 8
	superCannon.L2SequenceNumber = 1_800_000_000
	invalidFault := faultGame(gameTypes.GameStatusInProgress, false)
	invalidFault.Timestamp = 4
	invalidZK := &monTypes.ZKGameData{
		CommonGameData: commonGame(gameTypes.ZKDisputeGameType, gameTypes.GameStatusInProgress, false),
		ProposalStatus: contracts.ProposalStatusChallenged,
	}
	invalidZK.Timestamp = 7

	forecast.Forecast([]monTypes.EnrichedGame{fault, zk, superPermissioned, superCannon, invalidFault, invalidZK}, 6, 7)
	require.EqualValues(t, 100, metricer.latestValidProposalL2Block)
	require.EqualValues(t, 8, metricer.latestValidProposal)
	require.EqualValues(t, 7, metricer.latestInvalidProposal)
	require.Equal(t, 6, metricer.ignoredGames)
	require.Equal(t, 7, metricer.failedGames)

	forecast.Forecast([]monTypes.EnrichedGame{zk}, 0, 0)
	require.Zero(t, metricer.latestValidProposalL2Block)
}

func TestForecastInvalidZKStateDoesNotPanic(t *testing.T) {
	forecast, metricer := setupForecastTest(t)
	game := &monTypes.ZKGameData{
		CommonGameData: commonGame(gameTypes.ZKDisputeGameType, gameTypes.GameStatusInProgress, true),
		ProposalStatus: contracts.ProposalStatusResolved,
	}
	require.NotPanics(t, func() {
		forecast.Forecast([]monTypes.EnrichedGame{game}, 0, 0)
	})
	require.Equal(t, map[metrics.GameAgreementStatus]int{metrics.AgreeChallengerAhead: 1}, metricer.gameAgreements)
}

func TestForecastAggregatesMultipleGames(t *testing.T) {
	forecast, metricer := setupForecastTest(t)
	forecast.Forecast([]monTypes.EnrichedGame{
		faultGame(gameTypes.GameStatusDefenderWon, true),
		&monTypes.ZKGameData{
			CommonGameData: commonGame(gameTypes.ZKDisputeGameType, gameTypes.GameStatusDefenderWon, true),
			ProposalStatus: contracts.ProposalStatusResolved,
		},
		&monTypes.SuperPermissionedGameData{
			CommonGameData: commonGame(gameTypes.SuperPermissionedGameType, gameTypes.GameStatusDefenderWon, true),
		},
		faultGame(gameTypes.GameStatusChallengerWon, false),
	}, 3, 4)
	require.Equal(t, map[metrics.GameAgreementStatus]int{
		metrics.AgreeDefenderWins:      3,
		metrics.DisagreeChallengerWins: 1,
	}, metricer.gameAgreements)
	require.Equal(t, 3, metricer.ignoredGames)
	require.Equal(t, 4, metricer.failedGames)
}

func TestForecastLogsExpectedRootAndUnexpectedResults(t *testing.T) {
	expectedRoot := common.Hash{0x22}

	t.Run("in progress", func(t *testing.T) {
		logger, logs := testlog.CaptureLogger(t, log.LvlDebug)
		metricer := &mockForecastMetrics{}
		forecast := NewForecast(logger, metricer)
		game := faultGame(gameTypes.GameStatusInProgress, true)
		game.Claims = createDeepClaimList()[:2]
		game.ExpectedRootClaim = expectedRoot

		forecast.Forecast([]monTypes.EnrichedGame{game}, 0, 0)

		entry := logs.FindLog(
			testlog.NewLevelFilter(log.LevelWarn),
			testlog.NewMessageFilter("Forecasting unexpected game result"),
		)
		require.NotNil(t, entry)
		require.Equal(t, game.RootClaim, entry.AttrValue("rootClaim"))
		require.Equal(t, expectedRoot, entry.AttrValue("expected"))
	})

	t.Run("terminal", func(t *testing.T) {
		logger, logs := testlog.CaptureLogger(t, log.LvlDebug)
		metricer := &mockForecastMetrics{}
		forecast := NewForecast(logger, metricer)
		game := faultGame(gameTypes.GameStatusChallengerWon, true)
		game.ExpectedRootClaim = expectedRoot

		forecast.Forecast([]monTypes.EnrichedGame{game}, 0, 0)

		entry := logs.FindLog(
			testlog.NewLevelFilter(log.LevelError),
			testlog.NewMessageFilter("Unexpected game result"),
		)
		require.NotNil(t, entry)
		require.Equal(t, game.RootClaim, entry.AttrValue("rootClaim"))
		require.Equal(t, expectedRoot, entry.AttrValue("correctClaim"))
	})

	t.Run("expected in progress", func(t *testing.T) {
		logger, logs := testlog.CaptureLogger(t, log.LvlDebug)
		metricer := &mockForecastMetrics{}
		forecast := NewForecast(logger, metricer)
		game := faultGame(gameTypes.GameStatusInProgress, true)
		game.Claims = createDeepClaimList()[:1]
		game.ExpectedRootClaim = expectedRoot

		forecast.Forecast([]monTypes.EnrichedGame{game}, 0, 0)

		entry := logs.FindLog(
			testlog.NewLevelFilter(log.LevelDebug),
			testlog.NewMessageFilter("Forecasting expected game result"),
		)
		require.NotNil(t, entry)
		require.Equal(t, game.RootClaim, entry.AttrValue("rootClaim"))
		require.Equal(t, expectedRoot, entry.AttrValue("expected"))
	})

	t.Run("super permissioned in progress", func(t *testing.T) {
		logger, logs := testlog.CaptureLogger(t, log.LvlDebug)
		metricer := &mockForecastMetrics{}
		forecast := NewForecast(logger, metricer)
		game := &monTypes.SuperPermissionedGameData{CommonGameData: commonGame(
			gameTypes.SuperPermissionedGameType, gameTypes.GameStatusInProgress, true,
		)}

		forecast.Forecast([]monTypes.EnrichedGame{game}, 0, 0)

		entry := logs.FindLog(
			testlog.NewLevelFilter(log.LevelError),
			testlog.NewMessageFilter("Found super permissioned game still in progress, this should be impossible, check game configuration"),
		)
		require.NotNil(t, entry)
		require.Equal(t, game.Proxy, entry.AttrValue("game"))
	})
}

func commonGame(gameType gameTypes.GameType, status gameTypes.GameStatus, agree bool) monTypes.CommonGameData {
	return monTypes.CommonGameData{
		GameMetadata:      gameTypes.GameMetadata{GameType: uint32(gameType)},
		Status:            status,
		RootClaim:         mockRootClaim,
		AgreeWithClaim:    agree,
		ExpectedRootClaim: mockRootClaim,
	}
}

func faultGame(status gameTypes.GameStatus, agree bool) *monTypes.FaultGameData {
	return &monTypes.FaultGameData{
		CommonGameData: commonGame(gameTypes.CannonGameType, status, agree),
	}
}

func setupForecastTest(t *testing.T) (*Forecast, *mockForecastMetrics) {
	logger := testlog.Logger(t, log.LvlDebug)
	metricer := &mockForecastMetrics{}
	return NewForecast(logger, metricer), metricer
}

type mockForecastMetrics struct {
	gameAgreements             map[metrics.GameAgreementStatus]int
	ignoredGames               int
	latestValidProposalL2Block uint64
	latestInvalidProposal      uint64
	latestValidProposal        uint64
	failedGames                int
}

func (m *mockForecastMetrics) RecordFailedGames(count int) {
	m.failedGames = count
}

func (m *mockForecastMetrics) RecordGameAgreements(counts map[metrics.GameAgreementStatus]int) {
	m.gameAgreements = counts
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
	}
}
