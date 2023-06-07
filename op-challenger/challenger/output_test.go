package challenger

import (
	"context"
	"errors"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-challenger/metrics"
	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/testlog"
)

func TestParseOutputLog_Succeeds(t *testing.T) {
	challenger := newTestChallenger(t, eth.OutputResponse{}, true)
	expectedBlockNumber := big.NewInt(0x04)
	expectedOutputRoot := eth.Bytes32{0x02}
	logTopic := challenger.l2ooABI.Events["OutputProposed"].ID
	log := types.Log{
		Topics: []common.Hash{logTopic, common.Hash(expectedOutputRoot), {0x03}, common.BigToHash(expectedBlockNumber)},
	}
	parsedBlockNumber, parsedOutputRoot, err := challenger.ParseOutputLog(&log)
	require.NoError(t, err)
	require.Equal(t, expectedBlockNumber, parsedBlockNumber)
	require.Equal(t, expectedOutputRoot, parsedOutputRoot)
}

func TestParseOutputLog_WrongLogTopic_Errors(t *testing.T) {
	challenger := newTestChallenger(t, eth.OutputResponse{}, true)
	_, _, err := challenger.ParseOutputLog(&types.Log{
		Topics: []common.Hash{{0x01}, {0x02}, {0x03}, {0x04}},
	})
	require.ErrorIs(t, err, ErrInvalidOutputLogTopic)
}

func TestParseOutputLog_BadLog_Panics(t *testing.T) {
	challenger := newTestChallenger(t, eth.OutputResponse{}, true)
	logTopic := challenger.l2ooABI.Events["OutputProposed"].ID
	require.Panics(t, func() {
		log := types.Log{
			Topics: []common.Hash{logTopic, {0x02}, {0x03}},
		}
		_, _, _ = challenger.ParseOutputLog(&log)
	})
}

func TestChallenger_ValidateOutput_RollupClientErrors(t *testing.T) {
	output := eth.OutputResponse{
		Version:    supportedL2OutputVersion,
		OutputRoot: eth.Bytes32{},
		BlockRef:   eth.L2BlockRef{},
	}

	challenger := newTestChallenger(t, output, true)

	valid, received, err := challenger.ValidateOutput(context.Background(), big.NewInt(0), output.OutputRoot)
	require.False(t, valid)
	require.Nil(t, received)
	require.ErrorIs(t, err, mockOutputApiError)
}

func TestChallenger_ValidateOutput_ErrorsWithWrongVersion(t *testing.T) {
	output := eth.OutputResponse{
		Version:    eth.Bytes32{0x01},
		OutputRoot: eth.Bytes32{0x01},
		BlockRef:   eth.L2BlockRef{},
	}

	challenger := newTestChallenger(t, output, false)

	valid, received, err := challenger.ValidateOutput(context.Background(), big.NewInt(0), eth.Bytes32{})
	require.False(t, valid)
	require.Nil(t, received)
	require.ErrorIs(t, err, ErrUnsupportedL2OOVersion)
}

func TestChallenger_ValidateOutput_ErrorsInvalidBlockNumber(t *testing.T) {
	output := eth.OutputResponse{
		Version:    supportedL2OutputVersion,
		OutputRoot: eth.Bytes32{0x01},
		BlockRef:   eth.L2BlockRef{},
	}

	challenger := newTestChallenger(t, output, false)

	valid, received, err := challenger.ValidateOutput(context.Background(), big.NewInt(1), output.OutputRoot)
	require.False(t, valid)
	require.Nil(t, received)
	require.ErrorIs(t, err, ErrInvalidBlockNumber)
}

func TestOutput_ValidateOutput(t *testing.T) {
	output := eth.OutputResponse{
		Version:    eth.Bytes32{},
		OutputRoot: eth.Bytes32{},
		BlockRef:   eth.L2BlockRef{},
	}

	challenger := newTestChallenger(t, output, false)

	valid, expected, err := challenger.ValidateOutput(context.Background(), big.NewInt(0), output.OutputRoot)
	require.Equal(t, *expected, output.OutputRoot)
	require.True(t, valid)
	require.NoError(t, err)
}

func TestChallenger_CompareOutputRoots_ErrorsWithDifferentRoots(t *testing.T) {
	output := eth.OutputResponse{
		Version:    eth.Bytes32{0xFF, 0xFF, 0xFF, 0xFF},
		OutputRoot: eth.Bytes32{},
		BlockRef:   eth.L2BlockRef{},
	}

	challenger := newTestChallenger(t, output, false)

	valid, err := challenger.compareOutputRoots(&output, output.OutputRoot, big.NewInt(0))
	require.False(t, valid)
	require.ErrorIs(t, err, ErrUnsupportedL2OOVersion)
}

func TestChallenger_CompareOutputRoots_ErrInvalidBlockNumber(t *testing.T) {
	output := eth.OutputResponse{
		Version:    supportedL2OutputVersion,
		OutputRoot: eth.Bytes32{},
		BlockRef:   eth.L2BlockRef{},
	}

	challenger := newTestChallenger(t, output, false)

	valid, err := challenger.compareOutputRoots(&output, output.OutputRoot, big.NewInt(1))
	require.False(t, valid)
	require.ErrorIs(t, err, ErrInvalidBlockNumber)
}

func TestChallenger_CompareOutputRoots_Succeeds(t *testing.T) {
	output := eth.OutputResponse{
		Version:    supportedL2OutputVersion,
		OutputRoot: eth.Bytes32{},
		BlockRef:   eth.L2BlockRef{},
	}

	challenger := newTestChallenger(t, output, false)

	valid, err := challenger.compareOutputRoots(&output, output.OutputRoot, big.NewInt(0))
	require.True(t, valid)
	require.NoError(t, err)

	valid, err = challenger.compareOutputRoots(&output, eth.Bytes32{0x01}, big.NewInt(0))
	require.False(t, valid)
	require.NoError(t, err)
}

func newTestChallenger(t *testing.T, output eth.OutputResponse, errors bool) *Challenger {
	outputApi := newMockOutputApi(output, errors)
	log := testlog.Logger(t, log.LvlError)
	metr := metrics.NewMetrics("test")
	parsedL2oo, err := bindings.L2OutputOracleMetaData.GetAbi()
	require.NoError(t, err)
	challenger := Challenger{
		rollupClient:   outputApi,
		log:            log,
		metr:           metr,
		networkTimeout: time.Duration(5) * time.Second,
		l2ooABI:        parsedL2oo,
	}
	return &challenger
}

var mockOutputApiError = errors.New("mock output api error")

type mockOutputApi struct {
	mock.Mock
	expected eth.OutputResponse
	errors   bool
}

func newMockOutputApi(output eth.OutputResponse, errors bool) *mockOutputApi {
	return &mockOutputApi{
		expected: output,
		errors:   errors,
	}
}

func (m *mockOutputApi) OutputAtBlock(ctx context.Context, blockNumber uint64) (*eth.OutputResponse, error) {
	if m.errors {
		return nil, mockOutputApiError
	}
	return &m.expected, nil
}
