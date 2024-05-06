package mon

import (
	"context"
	"errors"
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

var (
	mockRootClaim = common.HexToHash("0x10")
)

func TestDetector_CheckRootAgreement(t *testing.T) {
	t.Parallel()

	t.Run("OutputFetchFails", func(t *testing.T) {
		validator, rollup, metrics := setupOutputValidatorTest(t)
		rollup.outputErr = errors.New("boom")
		agree, fetched, err := validator.CheckRootAgreement(context.Background(), 100, 0, mockRootClaim)
		require.ErrorIs(t, err, rollup.outputErr)
		require.Equal(t, common.Hash{}, fetched)
		require.False(t, agree)
		require.Zero(t, metrics.fetchTime)
	})

	t.Run("OutputMismatch_Safe", func(t *testing.T) {
		validator, _, metrics := setupOutputValidatorTest(t)
		agree, fetched, err := validator.CheckRootAgreement(context.Background(), 100, 0, common.Hash{})
		require.NoError(t, err)
		require.Equal(t, mockRootClaim, fetched)
		require.False(t, agree)
		require.NotZero(t, metrics.fetchTime)
	})

	t.Run("OutputMatches_Safe", func(t *testing.T) {
		validator, _, metrics := setupOutputValidatorTest(t)
		agree, fetched, err := validator.CheckRootAgreement(context.Background(), 200, 0, mockRootClaim)
		require.NoError(t, err)
		require.Equal(t, mockRootClaim, fetched)
		require.True(t, agree)
		require.NotZero(t, metrics.fetchTime)
	})

	t.Run("OutputMismatch_NotSafe", func(t *testing.T) {
		validator, client, metrics := setupOutputValidatorTest(t)
		client.safeHeadNum = 99
		agree, fetched, err := validator.CheckRootAgreement(context.Background(), 100, 0, common.Hash{})
		require.NoError(t, err)
		require.Equal(t, mockRootClaim, fetched)
		require.False(t, agree)
		require.NotZero(t, metrics.fetchTime)
	})

	t.Run("OutputMatches_SafeHeadError", func(t *testing.T) {
		validator, client, metrics := setupOutputValidatorTest(t)
		client.safeHeadErr = errors.New("boom")
		agree, fetched, err := validator.CheckRootAgreement(context.Background(), 200, 0, mockRootClaim)
		require.NoError(t, err)
		require.Equal(t, mockRootClaim, fetched)
		require.True(t, agree) // Assume safe if we can't retrieve the safe head so monitoring isn't dependent on safe head db
		require.NotZero(t, metrics.fetchTime)
	})

	t.Run("OutputMismatch_SafeHeadError", func(t *testing.T) {
		validator, client, metrics := setupOutputValidatorTest(t)
		client.safeHeadErr = errors.New("boom")
		agree, fetched, err := validator.CheckRootAgreement(context.Background(), 100, 0, common.Hash{})
		require.NoError(t, err)
		require.Equal(t, mockRootClaim, fetched)
		require.False(t, agree) // Not agreed because the root doesn't match
		require.NotZero(t, metrics.fetchTime)
	})

	t.Run("OutputMatches_NotSafe", func(t *testing.T) {
		validator, client, metrics := setupOutputValidatorTest(t)
		client.safeHeadNum = 99
		agree, fetched, err := validator.CheckRootAgreement(context.Background(), 200, 100, mockRootClaim)
		require.NoError(t, err)
		require.Equal(t, mockRootClaim, fetched)
		require.False(t, agree)
		require.NotZero(t, metrics.fetchTime)
	})

	t.Run("OutputNotFound", func(t *testing.T) {
		validator, rollup, metrics := setupOutputValidatorTest(t)
		// This crazy error is what we actually get back from the API
		rollup.outputErr = errors.New("failed to get L2 block ref with sync status: failed to determine L2BlockRef of height 42984924, could not get payload: not found")
		agree, fetched, err := validator.CheckRootAgreement(context.Background(), 100, 42984924, mockRootClaim)
		require.NoError(t, err)
		require.Equal(t, common.Hash{}, fetched)
		require.False(t, agree)
		require.Zero(t, metrics.fetchTime)
	})
}

func setupOutputValidatorTest(t *testing.T) (*outputValidator, *stubRollupClient, *stubOutputMetrics) {
	logger := testlog.Logger(t, log.LvlInfo)
	client := &stubRollupClient{safeHeadNum: 99999999999}
	metrics := &stubOutputMetrics{}
	validator := newOutputValidator(logger, metrics, client)
	return validator, client, metrics
}

type stubOutputMetrics struct {
	fetchTime float64
}

func (s *stubOutputMetrics) RecordOutputFetchTime(fetchTime float64) {
	s.fetchTime = fetchTime
}

type stubRollupClient struct {
	blockNum    uint64
	outputErr   error
	safeHeadErr error
	safeHeadNum uint64
}

func (s *stubRollupClient) OutputAtBlock(ctx context.Context, blockNum uint64) (*eth.OutputResponse, error) {
	s.blockNum = blockNum
	return &eth.OutputResponse{OutputRoot: eth.Bytes32(mockRootClaim)}, s.outputErr
}

func (s *stubRollupClient) SafeHeadAtL1Block(_ context.Context, _ uint64) (*eth.SafeHeadResponse, error) {
	if s.safeHeadErr != nil {
		return nil, s.safeHeadErr
	}
	return &eth.SafeHeadResponse{
		SafeHead: eth.BlockID{
			Number: s.safeHeadNum,
		},
	}, nil
}
