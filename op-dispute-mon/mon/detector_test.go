package mon

import (
	"context"
	"errors"
	"testing"

	"github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-dispute-mon/metrics"
	monTypes "github.com/ethereum-optimism/optimism/op-dispute-mon/mon/types"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

func TestDetector_Detect(t *testing.T) {
	t.Parallel()

	t.Run("NoGames", func(t *testing.T) {
		detector, m, _, _ := setupDetectorTest(t)
		detector.Detect(context.Background(), []*monTypes.EnrichedGameData{})
		m.Equals(t, 0, 0, 0)
		m.Mapped(t, map[metrics.GameAgreementStatus]int{})
	})

	t.Run("CheckAgreementFails", func(t *testing.T) {
		detector, m, rollup, _ := setupDetectorTest(t)
		rollup.err = errors.New("boom")
		detector.Detect(context.Background(), []*monTypes.EnrichedGameData{{}})
		m.Equals(t, 1, 0, 0) // Status should still be metriced here!
		m.Mapped(t, map[metrics.GameAgreementStatus]int{})
	})

	t.Run("SingleGame", func(t *testing.T) {
		detector, m, _, _ := setupDetectorTest(t)
		detector.Detect(context.Background(), []*monTypes.EnrichedGameData{{Status: types.GameStatusChallengerWon}})
		m.Equals(t, 0, 0, 1)
		m.Mapped(t, map[metrics.GameAgreementStatus]int{metrics.DisagreeChallengerWins: 1})
	})

	t.Run("MultipleGames", func(t *testing.T) {
		detector, m, _, _ := setupDetectorTest(t)
		detector.Detect(context.Background(), []*monTypes.EnrichedGameData{
			{Status: types.GameStatusChallengerWon},
			{Status: types.GameStatusChallengerWon},
			{Status: types.GameStatusChallengerWon},
		})
		m.Equals(t, 0, 0, 3)
		m.Mapped(t, map[metrics.GameAgreementStatus]int{metrics.DisagreeChallengerWins: 3})
	})
}

func TestDetector_RecordBatch(t *testing.T) {
	tests := []struct {
		name   string
		batch  monTypes.DetectionBatch
		expect func(*testing.T, *mockDetectorMetricer)
	}{
		{
			name:   "no games",
			batch:  monTypes.DetectionBatch{},
			expect: func(t *testing.T, metrics *mockDetectorMetricer) {},
		},
		{
			name:  "in_progress",
			batch: monTypes.DetectionBatch{InProgress: 1},
			expect: func(t *testing.T, m *mockDetectorMetricer) {
				for status, count := range m.gameAgreement {
					require.Zerof(t, count, "incorrectly reported in progress game as %v", status)
				}
			},
		},
		{
			name:  "agree_defender_wins",
			batch: monTypes.DetectionBatch{AgreeDefenderWins: 1},
			expect: func(t *testing.T, m *mockDetectorMetricer) {
				require.Equal(t, 1, m.gameAgreement[metrics.AgreeDefenderWins])
			},
		},
		{
			name:  "disagree_defender_wins",
			batch: monTypes.DetectionBatch{DisagreeDefenderWins: 1},
			expect: func(t *testing.T, m *mockDetectorMetricer) {
				require.Equal(t, 1, m.gameAgreement[metrics.DisagreeDefenderWins])
			},
		},
		{
			name:  "agree_challenger_wins",
			batch: monTypes.DetectionBatch{AgreeChallengerWins: 1},
			expect: func(t *testing.T, m *mockDetectorMetricer) {
				require.Equal(t, 1, m.gameAgreement[metrics.AgreeChallengerWins])
			},
		},
		{
			name:  "disagree_challenger_wins",
			batch: monTypes.DetectionBatch{DisagreeChallengerWins: 1},
			expect: func(t *testing.T, m *mockDetectorMetricer) {
				require.Equal(t, 1, m.gameAgreement[metrics.DisagreeChallengerWins])
			},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			monitor, metrics, _, _ := setupDetectorTest(t)
			monitor.recordBatch(test.batch)
			test.expect(t, metrics)
		})
	}
}

func TestDetector_CheckAgreement_Fails(t *testing.T) {
	detector, _, rollup, _ := setupDetectorTest(t)
	rollup.err = errors.New("boom")
	_, err := detector.checkAgreement(context.Background(), common.Address{}, 0, common.Hash{}, types.GameStatusInProgress)
	require.ErrorIs(t, err, rollup.err)
}

func TestDetector_CheckAgreement_Succeeds(t *testing.T) {
	tests := []struct {
		name           string
		rootClaim      common.Hash
		status         types.GameStatus
		expectBatch    func(*monTypes.DetectionBatch)
		expectErrorLog bool
		expectStatus   types.GameStatus
		err            error
	}{
		{
			name: "in_progress",
			expectBatch: func(batch *monTypes.DetectionBatch) {
				require.Equal(t, 1, batch.InProgress)
			},
		},
		{
			name:         "agree_defender_wins",
			rootClaim:    mockRootClaim,
			status:       types.GameStatusDefenderWon,
			expectStatus: types.GameStatusDefenderWon,
			expectBatch: func(batch *monTypes.DetectionBatch) {
				require.Equal(t, 1, batch.AgreeDefenderWins)
			},
		},
		{
			name:         "disagree_defender_wins",
			status:       types.GameStatusDefenderWon,
			expectStatus: types.GameStatusChallengerWon,
			expectBatch: func(batch *monTypes.DetectionBatch) {
				require.Equal(t, 1, batch.DisagreeDefenderWins)
			},
			expectErrorLog: true,
		},
		{
			name:         "agree_challenger_wins",
			rootClaim:    mockRootClaim,
			status:       types.GameStatusChallengerWon,
			expectStatus: types.GameStatusDefenderWon,
			expectBatch: func(batch *monTypes.DetectionBatch) {
				require.Equal(t, 1, batch.AgreeChallengerWins)
			},
			expectErrorLog: true,
		},
		{
			name:         "disagree_challenger_wins",
			status:       types.GameStatusChallengerWon,
			expectStatus: types.GameStatusChallengerWon,
			expectBatch: func(batch *monTypes.DetectionBatch) {
				require.Equal(t, 1, batch.DisagreeChallengerWins)
			},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			detector, _, _, logs := setupDetectorTest(t)
			batch, err := detector.checkAgreement(context.Background(), common.Address{}, 0, test.rootClaim, test.status)
			require.NoError(t, err)
			test.expectBatch(&batch)

			levelFilter := testlog.NewLevelFilter(log.LevelError)
			if !test.expectErrorLog {
				require.Empty(t, logs.FindLogs(levelFilter), "Should not log an error")
			} else {
				msgFilter := testlog.NewMessageFilter("Unexpected game result")
				l := logs.FindLog(levelFilter, msgFilter)
				require.NotNil(t, l, "Should have logged an error")
				expectedResult := l.AttrValue("expectedResult")
				require.Equal(t, test.expectStatus, expectedResult)
				actualResult := l.AttrValue("actualResult")
				require.Equal(t, test.status, actualResult)
			}
		})
	}
}

func setupDetectorTest(t *testing.T) (*detector, *mockDetectorMetricer, *stubOutputValidator, *testlog.CapturingHandler) {
	logger, capturedLogs := testlog.CaptureLogger(t, log.LvlDebug)
	metrics := &mockDetectorMetricer{}
	validator := &stubOutputValidator{}
	detector := newDetector(logger, metrics, validator)
	return detector, metrics, validator, capturedLogs
}

type stubOutputValidator struct {
	calls int
	err   error
}

func (s *stubOutputValidator) CheckRootAgreement(ctx context.Context, blockNum uint64, rootClaim common.Hash) (bool, common.Hash, error) {
	s.calls++
	if s.err != nil {
		return false, common.Hash{}, s.err
	}
	return rootClaim == mockRootClaim, mockRootClaim, nil
}

type mockDetectorMetricer struct {
	inProgress    int
	defenderWon   int
	challengerWon int
	gameAgreement map[metrics.GameAgreementStatus]int
}

func (m *mockDetectorMetricer) Equals(t *testing.T, inProgress, defenderWon, challengerWon int) {
	require.Equal(t, inProgress, m.inProgress)
	require.Equal(t, defenderWon, m.defenderWon)
	require.Equal(t, challengerWon, m.challengerWon)
}

func (m *mockDetectorMetricer) Mapped(t *testing.T, expected map[metrics.GameAgreementStatus]int) {
	for k, v := range m.gameAgreement {
		require.Equal(t, expected[k], v)
	}
}

func (m *mockDetectorMetricer) RecordGamesStatus(inProgress, defenderWon, challengerWon int) {
	m.inProgress = inProgress
	m.defenderWon = defenderWon
	m.challengerWon = challengerWon
}

func (m *mockDetectorMetricer) RecordGameAgreement(status metrics.GameAgreementStatus, count int) {
	if m.gameAgreement == nil {
		m.gameAgreement = make(map[metrics.GameAgreementStatus]int)
	}
	m.gameAgreement[status] += count
}
