package mon

import (
	"context"
	"errors"
	"testing"

	"github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

func TestDetector_Detect(t *testing.T) {
	t.Parallel()

	t.Run("NoGames", func(t *testing.T) {
		detector, metrics, _, _, _ := setupDetectorTest(t)
		detector.Detect(context.Background(), []types.GameMetadata{})
		metrics.Equals(t, 0, 0, 0)
		metrics.Mapped(t, map[string]int{})
	})

	t.Run("MetadataFetchFails", func(t *testing.T) {
		detector, metrics, creator, _, _ := setupDetectorTest(t)
		creator.err = errors.New("boom")
		detector.Detect(context.Background(), []types.GameMetadata{{}})
		metrics.Equals(t, 0, 0, 0)
		metrics.Mapped(t, map[string]int{})
	})

	t.Run("CheckAgreementFails", func(t *testing.T) {
		detector, metrics, creator, rollup, _ := setupDetectorTest(t)
		rollup.err = errors.New("boom")
		creator.loader = &mockMetadataLoader{status: types.GameStatusInProgress}
		detector.Detect(context.Background(), []types.GameMetadata{{}})
		metrics.Equals(t, 1, 0, 0) // Status should still be metriced here!
		metrics.Mapped(t, map[string]int{})
	})

	t.Run("SingleGame", func(t *testing.T) {
		detector, metrics, creator, _, _ := setupDetectorTest(t)
		loader := &mockMetadataLoader{status: types.GameStatusInProgress}
		creator.loader = loader
		detector.Detect(context.Background(), []types.GameMetadata{{}})
		metrics.Equals(t, 1, 0, 0)
		metrics.Mapped(t, map[string]int{"in_progress": 1})
	})

	t.Run("MultipleGames", func(t *testing.T) {
		detector, metrics, creator, _, _ := setupDetectorTest(t)
		loader := &mockMetadataLoader{status: types.GameStatusInProgress}
		creator.loader = loader
		detector.Detect(context.Background(), []types.GameMetadata{{}, {}, {}})
		metrics.Equals(t, 3, 0, 0)
		metrics.Mapped(t, map[string]int{"in_progress": 3})
	})
}

func TestDetector_RecordBatch(t *testing.T) {
	tests := []struct {
		name   string
		batch  detectionBatch
		expect func(*testing.T, *mockDetectorMetricer)
	}{
		{
			name:   "no games",
			batch:  detectionBatch{},
			expect: func(t *testing.T, metrics *mockDetectorMetricer) {},
		},
		{
			name:  "in_progress",
			batch: detectionBatch{inProgress: 1},
			expect: func(t *testing.T, metrics *mockDetectorMetricer) {
				require.Equal(t, 1, metrics.gameAgreement["in_progress"])
			},
		},
		{
			name:  "agree_defender_wins",
			batch: detectionBatch{agreeDefenderWins: 1},
			expect: func(t *testing.T, metrics *mockDetectorMetricer) {
				require.Equal(t, 1, metrics.gameAgreement["agree_defender_wins"])
			},
		},
		{
			name:  "disagree_defender_wins",
			batch: detectionBatch{disagreeDefenderWins: 1},
			expect: func(t *testing.T, metrics *mockDetectorMetricer) {
				require.Equal(t, 1, metrics.gameAgreement["disagree_defender_wins"])
			},
		},
		{
			name:  "agree_challenger_wins",
			batch: detectionBatch{agreeChallengerWins: 1},
			expect: func(t *testing.T, metrics *mockDetectorMetricer) {
				require.Equal(t, 1, metrics.gameAgreement["agree_challenger_wins"])
			},
		},
		{
			name:  "disagree_challenger_wins",
			batch: detectionBatch{disagreeChallengerWins: 1},
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

func TestDetector_FetchGameMetadata(t *testing.T) {
	t.Parallel()

	t.Run("CreateContractFails", func(t *testing.T) {
		detector, _, creator, _, _ := setupDetectorTest(t)
		creator.err = errors.New("boom")
		_, _, _, err := detector.fetchGameMetadata(context.Background(), types.GameMetadata{})
		require.ErrorIs(t, err, creator.err)
	})

	t.Run("GetGameMetadataFails", func(t *testing.T) {
		detector, _, creator, _, _ := setupDetectorTest(t)
		loader := &mockMetadataLoader{err: errors.New("boom")}
		creator.loader = loader
		_, _, _, err := detector.fetchGameMetadata(context.Background(), types.GameMetadata{})
		require.Error(t, err)
	})

	t.Run("Success", func(t *testing.T) {
		detector, _, creator, _, _ := setupDetectorTest(t)
		loader := &mockMetadataLoader{status: types.GameStatusInProgress}
		creator.loader = loader
		_, _, status, err := detector.fetchGameMetadata(context.Background(), types.GameMetadata{})
		require.NoError(t, err)
		require.Equal(t, types.GameStatusInProgress, status)
	})
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
		expectBatch    func(*detectionBatch)
		expectErrorLog bool
		expectStatus   types.GameStatus
		err            error
	}{
		{
			name: "in_progress",
			expectBatch: func(batch *detectionBatch) {
				require.Equal(t, 1, batch.inProgress)
			},
		},
		{
			name:         "agree_defender_wins",
			rootClaim:    mockRootClaim,
			status:       types.GameStatusDefenderWon,
			expectStatus: types.GameStatusDefenderWon,
			expectBatch: func(batch *detectionBatch) {
				require.Equal(t, 1, batch.agreeDefenderWins)
			},
		},
		{
			name:         "disagree_defender_wins",
			status:       types.GameStatusDefenderWon,
			expectStatus: types.GameStatusChallengerWon,
			expectBatch: func(batch *detectionBatch) {
				require.Equal(t, 1, batch.disagreeDefenderWins)
			},
			expectErrorLog: true,
		},
		{
			name:         "agree_challenger_wins",
			rootClaim:    mockRootClaim,
			status:       types.GameStatusChallengerWon,
			expectStatus: types.GameStatusDefenderWon,
			expectBatch: func(batch *detectionBatch) {
				require.Equal(t, 1, batch.agreeChallengerWins)
			},
			expectErrorLog: true,
		},
		{
			name:         "disagree_challenger_wins",
			status:       types.GameStatusChallengerWon,
			expectStatus: types.GameStatusChallengerWon,
			expectBatch: func(batch *detectionBatch) {
				require.Equal(t, 1, batch.disagreeChallengerWins)
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

			if !test.expectErrorLog {
				require.Empty(t, logs.FindLogsWithLevel(log.LevelError), "Should not log an error`")
			} else {
				l := logs.FindLog(log.LevelError, "Unexpected game result")
				require.NotNil(t, l, "Should have logged an error")
				expectedResult := l.AttrValue("expectedResult")
				require.Equal(t, test.expectStatus, expectedResult)
				actualResult := l.AttrValue("actualResult")
				require.Equal(t, test.status, actualResult)
			}
		})
	}
}

func setupDetectorTest(t *testing.T) (*detector, *mockDetectorMetricer, *mockMetadataCreator, *stubOutputValidator, *testlog.CapturingHandler) {
	logger, capturedLogs := testlog.CaptureLogger(t, log.LvlDebug)
	metrics := &mockDetectorMetricer{}
	loader := &mockMetadataLoader{}
	creator := &mockMetadataCreator{loader: loader}
	validator := &stubOutputValidator{}
	detector := newDetector(logger, metrics, creator, validator)
	return detector, metrics, creator, validator, capturedLogs
}

type stubOutputValidator struct {
	err error
}

func (s *stubOutputValidator) CheckRootAgreement(ctx context.Context, blockNum uint64, rootClaim common.Hash) (bool, common.Hash, error) {
	if s.err != nil {
		return false, common.Hash{}, s.err
	}
	return rootClaim == mockRootClaim, mockRootClaim, nil
}

type mockMetadataCreator struct {
	calls  int
	err    error
	loader *mockMetadataLoader
}

func (m *mockMetadataCreator) CreateContract(game types.GameMetadata) (MetadataLoader, error) {
	m.calls++
	if m.err != nil {
		return nil, m.err
	}
	return m.loader, nil
}

type mockMetadataLoader struct {
	calls  int
	status types.GameStatus
	err    error
}

func (m *mockMetadataLoader) GetGameMetadata(ctx context.Context) (uint64, common.Hash, types.GameStatus, error) {
	m.calls++
	if m.err != nil {
		return 0, common.Hash{}, m.status, m.err
	}
	return 0, common.Hash{}, m.status, nil
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
