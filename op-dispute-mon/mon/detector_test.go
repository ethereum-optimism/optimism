package mon

import (
	"context"
	"errors"
	"testing"

	"github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

var (
	mockRootClaim = common.HexToHash("0x10")
)

func TestDetector_RecordBatch(t *testing.T) {
	tests := []struct {
		name   string
		batch  detectionBatch
		expect func(*testing.T, *stubMonitorMetricer)
	}{
		{
			name:   "no games",
			batch:  detectionBatch{},
			expect: func(t *testing.T, metrics *stubMonitorMetricer) {},
		},
		{
			name:  "in_progress",
			batch: detectionBatch{inProgress: 1},
			expect: func(t *testing.T, metrics *stubMonitorMetricer) {
				require.Equal(t, 1, metrics.gameAgreement["in_progress"])
			},
		},
		{
			name:  "agree_defender_wins",
			batch: detectionBatch{agreeDefenderWins: 1},
			expect: func(t *testing.T, metrics *stubMonitorMetricer) {
				require.Equal(t, 1, metrics.gameAgreement["agree_defender_wins"])
			},
		},
		{
			name:  "disagree_defender_wins",
			batch: detectionBatch{disagreeDefenderWins: 1},
			expect: func(t *testing.T, metrics *stubMonitorMetricer) {
				require.Equal(t, 1, metrics.gameAgreement["disagree_defender_wins"])
			},
		},
		{
			name:  "agree_challenger_wins",
			batch: detectionBatch{agreeChallengerWins: 1},
			expect: func(t *testing.T, metrics *stubMonitorMetricer) {
				require.Equal(t, 1, metrics.gameAgreement["agree_challenger_wins"])
			},
		},
		{
			name:  "disagree_challenger_wins",
			batch: detectionBatch{disagreeChallengerWins: 1},
			expect: func(t *testing.T, metrics *stubMonitorMetricer) {
				require.Equal(t, 1, metrics.gameAgreement["disagree_challenger_wins"])
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

func TestDetector_RecordGameStatus(t *testing.T) {
	tests := []struct {
		name   string
		status types.GameStatus
		expect func(*testing.T, *stubMonitorMetricer)
	}{
		{
			name:   "in_progress",
			status: types.GameStatusInProgress,
			expect: func(t *testing.T, metrics *stubMonitorMetricer) {
				require.Equal(t, 1, metrics.inProgress)
				require.Equal(t, 0, metrics.defenderWon)
				require.Equal(t, 0, metrics.challengerWon)
			},
		},
		{
			name:   "defender_won",
			status: types.GameStatusDefenderWon,
			expect: func(t *testing.T, metrics *stubMonitorMetricer) {
				require.Equal(t, 0, metrics.inProgress)
				require.Equal(t, 1, metrics.defenderWon)
				require.Equal(t, 0, metrics.challengerWon)
			},
		},
		{
			name:   "challenger_won",
			status: types.GameStatusChallengerWon,
			expect: func(t *testing.T, metrics *stubMonitorMetricer) {
				require.Equal(t, 0, metrics.inProgress)
				require.Equal(t, 0, metrics.defenderWon)
				require.Equal(t, 1, metrics.challengerWon)
			},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			detector, metrics, _, _ := setupDetectorTest(t)
			detector.recordGameStatus(context.Background(), test.status)
			test.expect(t, metrics)
		})
	}
}

func TestDetector_CheckRootAgreement(t *testing.T) {
	t.Parallel()

	t.Run("OutputFetchFails", func(t *testing.T) {
		detector, _, _, rollup := setupDetectorTest(t)
		rollup.err = errors.New("boom")
		agree, err := detector.checkRootAgreement(context.Background(), 0, mockRootClaim)
		require.ErrorIs(t, err, rollup.err)
		require.False(t, agree)
	})

	t.Run("OutputMismatch", func(t *testing.T) {
		detector, _, _, _ := setupDetectorTest(t)
		agree, err := detector.checkRootAgreement(context.Background(), 0, common.Hash{})
		require.NoError(t, err)
		require.False(t, agree)
	})

	t.Run("OutputMatches", func(t *testing.T) {
		detector, _, _, _ := setupDetectorTest(t)
		agree, err := detector.checkRootAgreement(context.Background(), 0, mockRootClaim)
		require.NoError(t, err)
		require.True(t, agree)
	})
}

func TestDetector_ProcessGame(t *testing.T) {
	t.Parallel()

	t.Run("GetGameMetadataFails", func(t *testing.T) {
		detector, _, loader, _ := setupDetectorTest(t)
		loader.err = errors.New("boom")
		_, err := detector.processGame(context.Background(), types.GameMetadata{})
		require.ErrorIs(t, err, loader.err)
	})

	t.Run("CheckRootAgreementFails", func(t *testing.T) {
		detector, _, _, rollup := setupDetectorTest(t)
		rollup.err = errors.New("boom")
		_, err := detector.processGame(context.Background(), types.GameMetadata{})
		require.ErrorIs(t, err, rollup.err)
	})

	t.Run("GameStatusInProgress", func(t *testing.T) {
		detector, _, loader, _ := setupDetectorTest(t)
		loader.status = types.GameStatusInProgress
		batch, err := detector.processGame(context.Background(), types.GameMetadata{})
		require.NoError(t, err)
		require.Equal(t, detectionBatch{inProgress: 1}, batch)
	})

	t.Run("GameStatusDefenderWon-Agree", func(t *testing.T) {
		detector, _, loader, _ := setupDetectorTest(t)
		loader.status = types.GameStatusDefenderWon
		loader.rootClaim = mockRootClaim
		batch, err := detector.processGame(context.Background(), types.GameMetadata{})
		require.NoError(t, err)
		require.Equal(t, detectionBatch{agreeDefenderWins: 1}, batch)
	})

	t.Run("GameStatusDefenderWon-Disagree", func(t *testing.T) {
		detector, _, loader, _ := setupDetectorTest(t)
		loader.status = types.GameStatusDefenderWon
		batch, err := detector.processGame(context.Background(), types.GameMetadata{})
		require.NoError(t, err)
		require.Equal(t, detectionBatch{disagreeDefenderWins: 1}, batch)
	})

	t.Run("GameStatusChallengerWon-Agree", func(t *testing.T) {
		detector, _, loader, _ := setupDetectorTest(t)
		loader.status = types.GameStatusChallengerWon
		loader.rootClaim = mockRootClaim
		batch, err := detector.processGame(context.Background(), types.GameMetadata{})
		require.NoError(t, err)
		require.Equal(t, detectionBatch{agreeChallengerWins: 1}, batch)
	})

	t.Run("GameStatusChallengerWon-Disagree", func(t *testing.T) {
		detector, _, loader, _ := setupDetectorTest(t)
		loader.status = types.GameStatusChallengerWon
		batch, err := detector.processGame(context.Background(), types.GameMetadata{})
		require.NoError(t, err)
		require.Equal(t, detectionBatch{disagreeChallengerWins: 1}, batch)
	})
}

func TestDetector_Detect_MergesBatches(t *testing.T) {
	tests := []struct {
		name      string
		status    types.GameStatus
		rootClaim common.Hash
		games     []types.GameMetadata
	}{
		{
			name:   "no_games",
			status: types.GameStatusInProgress,
			games:  []types.GameMetadata{},
		},
		{
			name:   "in_progress",
			status: types.GameStatusInProgress,
			games:  []types.GameMetadata{{}, {}, {}},
		},
		{
			name:      "agree_defender_wins",
			status:    types.GameStatusDefenderWon,
			rootClaim: mockRootClaim,
			games:     []types.GameMetadata{{}, {}, {}},
		},
		{
			name:   "disagree_defender_wins",
			status: types.GameStatusDefenderWon,
			games:  []types.GameMetadata{{}, {}, {}},
		},
		{
			name:      "agree_challenger_wins",
			status:    types.GameStatusChallengerWon,
			rootClaim: mockRootClaim,
			games:     []types.GameMetadata{{}, {}, {}},
		},
		{
			name:   "disagree_challenger_wins",
			status: types.GameStatusChallengerWon,
			games:  []types.GameMetadata{{}, {}, {}},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			detector, metrics, loader, _ := setupDetectorTest(t)
			loader.status = test.status
			loader.rootClaim = test.rootClaim
			detector.Detect(context.Background(), test.games)
			require.Equal(t, len(test.games), loader.calls)
			require.Equal(t, len(test.games), metrics.gameAgreement[test.name])
		})
	}
}

func setupDetectorTest(t *testing.T) (*detector, *stubMonitorMetricer, *stubMetadataLoader, *stubRollupClient) {
	logger := testlog.Logger(t, log.LvlDebug)
	metrics := &stubMonitorMetricer{}
	loader := &stubMetadataLoader{}
	rollupClient := &stubRollupClient{}
	detector := newDetector(logger, metrics, loader, rollupClient)
	return detector, metrics, loader, rollupClient
}

type stubRollupClient struct {
	blockNum uint64
	err      error
}

func (s *stubRollupClient) OutputAtBlock(ctx context.Context, blockNum uint64) (*eth.OutputResponse, error) {
	s.blockNum = blockNum
	return &eth.OutputResponse{OutputRoot: eth.Bytes32(mockRootClaim)}, s.err
}

type stubMetadataLoader struct {
	calls     int
	rootClaim common.Hash
	status    types.GameStatus
	err       error
}

func (m *stubMetadataLoader) GetGameMetadata(ctx context.Context, _ common.Address) (uint64, common.Hash, types.GameStatus, error) {
	m.calls++
	if m.err != nil {
		return 0, common.Hash{}, 0, m.err
	}
	return 0, m.rootClaim, m.status, nil
}
