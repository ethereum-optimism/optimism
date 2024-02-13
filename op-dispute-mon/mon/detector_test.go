package mon

import (
	"context"
	"testing"

	"github.com/ethereum-optimism/optimism/op-challenger/game/types"
	monTypes "github.com/ethereum-optimism/optimism/op-dispute-mon/mon/types"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

func TestDetector_Detect(t *testing.T) {
	t.Parallel()

	t.Run("NoGames", func(t *testing.T) {
		detector, metrics, _ := setupDetectorTest(t)
		detector.Detect(context.Background(), []monTypes.EnrichedGameData{})
		metrics.Equals(t, 0, 0, 0)
		metrics.Mapped(t, map[string]int{})
	})

	t.Run("SingleGame", func(t *testing.T) {
		detector, metrics, _ := setupDetectorTest(t)
		detector.Detect(context.Background(), []monTypes.EnrichedGameData{{}})
		metrics.Equals(t, 1, 0, 0)
		metrics.Mapped(t, map[string]int{"in_progress": 1})
	})

	t.Run("MultipleGames", func(t *testing.T) {
		detector, metrics, _ := setupDetectorTest(t)
		detector.Detect(context.Background(), []monTypes.EnrichedGameData{{}, {}, {}})
		metrics.Equals(t, 3, 0, 0)
		metrics.Mapped(t, map[string]int{"in_progress": 3})
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
			expect: func(t *testing.T, metrics *mockDetectorMetricer) {
				require.Equal(t, 1, metrics.gameAgreement["in_progress"])
			},
		},
		{
			name:  "agree_defender_wins",
			batch: monTypes.DetectionBatch{AgreeDefenderWins: 1},
			expect: func(t *testing.T, metrics *mockDetectorMetricer) {
				require.Equal(t, 1, metrics.gameAgreement["agree_defender_wins"])
			},
		},
		{
			name:  "disagree_defender_wins",
			batch: monTypes.DetectionBatch{DisagreeDefenderWins: 1},
			expect: func(t *testing.T, metrics *mockDetectorMetricer) {
				require.Equal(t, 1, metrics.gameAgreement["disagree_defender_wins"])
			},
		},
		{
			name:  "agree_challenger_wins",
			batch: monTypes.DetectionBatch{AgreeChallengerWins: 1},
			expect: func(t *testing.T, metrics *mockDetectorMetricer) {
				require.Equal(t, 1, metrics.gameAgreement["agree_challenger_wins"])
			},
		},
		{
			name:  "disagree_challenger_wins",
			batch: monTypes.DetectionBatch{DisagreeChallengerWins: 1},
			expect: func(t *testing.T, metrics *mockDetectorMetricer) {
				require.Equal(t, 1, metrics.gameAgreement["disagree_challenger_wins"])
			},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			monitor, metrics, _ := setupDetectorTest(t)
			monitor.recordBatch(test.batch)
			test.expect(t, metrics)
		})
	}
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
			detector, _, logs := setupDetectorTest(t)
			game := monTypes.EnrichedGameData{Status: test.status, RootClaim: test.rootClaim}
			batch, err := detector.checkAgreement(context.Background(), game)
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

var mockRootClaim = common.Hash{0xaa}

func setupDetectorTest(t *testing.T) (*detector, *mockDetectorMetricer, *testlog.CapturingHandler) {
	logger, capturedLogs := testlog.CaptureLogger(t, log.LvlDebug)
	metrics := &mockDetectorMetricer{}
	detector := newDetector(logger, metrics)
	return detector, metrics, capturedLogs
}

type mockDetectorMetricer struct {
	inProgress    int
	defenderWon   int
	challengerWon int
	gameAgreement map[string]int
}

func (m *mockDetectorMetricer) Equals(t *testing.T, inProgress, defenderWon, challengerWon int) {
	require.Equal(t, inProgress, m.inProgress)
	require.Equal(t, defenderWon, m.defenderWon)
	require.Equal(t, challengerWon, m.challengerWon)
}

func (m *mockDetectorMetricer) Mapped(t *testing.T, expected map[string]int) {
	for k, v := range m.gameAgreement {
		require.Equal(t, expected[k], v)
	}
}

func (m *mockDetectorMetricer) RecordGamesStatus(inProgress, defenderWon, challengerWon int) {
	m.inProgress = inProgress
	m.defenderWon = defenderWon
	m.challengerWon = challengerWon
}

func (m *mockDetectorMetricer) RecordGameAgreement(status string, count int) {
	if m.gameAgreement == nil {
		m.gameAgreement = make(map[string]int)
	}
	m.gameAgreement[status] += count
}
