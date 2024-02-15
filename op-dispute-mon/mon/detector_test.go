package mon

import (
	"context"
	"errors"
	"testing"

	faultTypes "github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	"github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-dispute-mon/mon/extract"
	monTypes "github.com/ethereum-optimism/optimism/op-dispute-mon/mon/types"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

func TestDetector_Detect(t *testing.T) {
	t.Parallel()

	t.Run("NoGames", func(t *testing.T) {
		detector, metrics, _, _, _ := setupDetectorTest(t)
		detector.Detect(context.Background(), []monTypes.EnrichedGameData{})
		metrics.Equals(t, 0, 0, 0)
		metrics.Mapped(t, map[string]int{})
	})

	t.Run("CheckAgreementFails", func(t *testing.T) {
		detector, metrics, creator, rollup, _ := setupDetectorTest(t)
		rollup.err = errors.New("boom")
		creator.caller.status = []types.GameStatus{types.GameStatusInProgress}
		creator.caller.rootClaim = []common.Hash{{}}
		detector.Detect(context.Background(), []monTypes.EnrichedGameData{{}})
		metrics.Equals(t, 1, 0, 0) // Status should still be metriced here!
		metrics.Mapped(t, map[string]int{})
	})

	t.Run("SingleGame", func(t *testing.T) {
		detector, metrics, creator, _, _ := setupDetectorTest(t)
		creator.caller.status = []types.GameStatus{types.GameStatusInProgress}
		creator.caller.rootClaim = []common.Hash{{}}
		detector.Detect(context.Background(), []monTypes.EnrichedGameData{{}})
		metrics.Equals(t, 1, 0, 0)
		metrics.Mapped(t, map[string]int{"in_progress": 1})
	})

	t.Run("MultipleGames", func(t *testing.T) {
		detector, metrics, creator, _, _ := setupDetectorTest(t)
		creator.caller.status = []types.GameStatus{
			types.GameStatusInProgress,
			types.GameStatusInProgress,
			types.GameStatusInProgress,
		}
		creator.caller.rootClaim = []common.Hash{{}, {}, {}}
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
			monitor, metrics, _, _, _ := setupDetectorTest(t)
			monitor.recordBatch(test.batch)
			test.expect(t, metrics)
		})
	}
}

func TestDetector_CheckAgreement_Fails(t *testing.T) {
	detector, _, _, rollup, _ := setupDetectorTest(t)
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
			detector, _, _, _, logs := setupDetectorTest(t)
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

func setupDetectorTest(t *testing.T) (*detector, *mockDetectorMetricer, *mockGameCallerCreator, *stubOutputValidator, *testlog.CapturingHandler) {
	logger, capturedLogs := testlog.CaptureLogger(t, log.LvlDebug)
	metrics := &mockDetectorMetricer{}
	caller := &mockGameCaller{}
	creator := &mockGameCallerCreator{caller: caller}
	validator := &stubOutputValidator{}
	detector := newDetector(logger, metrics, creator, validator)
	return detector, metrics, creator, validator, capturedLogs
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

type mockGameCallerCreator struct {
	calls  int
	err    error
	caller *mockGameCaller
}

func (m *mockGameCallerCreator) CreateContract(game types.GameMetadata) (extract.GameCaller, error) {
	m.calls++
	if m.err != nil {
		return nil, m.err
	}
	return m.caller, nil
}

type mockGameCaller struct {
	calls       int
	claimsCalls int
	claims      [][]faultTypes.Claim
	status      []types.GameStatus
	rootClaim   []common.Hash
	err         error
	claimsErr   error
}

func (m *mockGameCaller) GetGameMetadata(ctx context.Context) (uint64, common.Hash, types.GameStatus, error) {
	idx := m.calls
	m.calls++
	if m.err != nil {
		return 0, m.rootClaim[idx], m.status[idx], m.err
	}
	return 0, m.rootClaim[idx], m.status[idx], nil
}

func (m *mockGameCaller) GetAllClaims(ctx context.Context) ([]faultTypes.Claim, error) {
	idx := m.claimsCalls
	m.claimsCalls++
	if m.claimsErr != nil {
		return nil, m.claimsErr
	}
	return m.claims[idx], nil
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
