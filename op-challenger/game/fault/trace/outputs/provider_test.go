package outputs

import (
	"context"
	"fmt"
	"testing"

	"github.com/ethereum-optimism/optimism/op-node/testlog"
	"github.com/ethereum-optimism/optimism/op-service/eth"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

var (
	prestateBlock      = uint64(100)
	prestateOutputRoot = common.HexToHash("0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	firstOutputRoot    = common.HexToHash("0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")
)

func TestGet(t *testing.T) {
	t.Run("TraceIndexBeforePrestate", func(t *testing.T) {
		provider, _ := setupWithTestData(t, prestateBlock)
		_, err := provider.Get(context.Background(), 0)
		require.ErrorIs(t, err, PreStateRequestErr)
	})

	t.Run("MissingOutputAtBlock", func(t *testing.T) {
		provider, _ := setupWithTestData(t, prestateBlock)
		traceIndex := 101
		_, err := provider.Get(context.Background(), uint64(traceIndex))
		require.ErrorAs(t, fmt.Errorf("no output at block %d", uint64(traceIndex+1)), &err)
	})

	t.Run("Success", func(t *testing.T) {
		provider, _ := setupWithTestData(t, prestateBlock)
		value, err := provider.Get(context.Background(), 100)
		require.NoError(t, err)
		require.Equal(t, value, firstOutputRoot)
	})
}

func TestAbsolutePreStateCommitment(t *testing.T) {
	t.Run("FailedToFetchOutput", func(t *testing.T) {
		provider, rollupClient := setupWithTestData(t, prestateBlock)
		rollupClient.errorsOnPrestateFetch = true
		_, err := provider.AbsolutePreStateCommitment(context.Background())
		require.ErrorAs(t, fmt.Errorf("no output at block %d", prestateBlock), &err)
	})

	t.Run("Success", func(t *testing.T) {
		provider, _ := setupWithTestData(t, prestateBlock)
		value, err := provider.AbsolutePreStateCommitment(context.Background())
		require.NoError(t, err)
		require.Equal(t, value, prestateOutputRoot)
	})
}

func TestGetStepData(t *testing.T) {
	provider, _ := setupWithTestData(t, prestateBlock)
	_, _, _, err := provider.GetStepData(context.Background(), 0)
	require.ErrorIs(t, err, GetStepDataErr)
}

func TestAbsolutePreState(t *testing.T) {
	provider, _ := setupWithTestData(t, prestateBlock)
	_, err := provider.AbsolutePreState(context.Background())
	require.ErrorIs(t, err, AbsolutePreStateErr)
}

func setupWithTestData(t *testing.T, prestateBlock uint64) (*OutputTraceProvider, *stubRollupClient) {
	rollupClient := stubRollupClient{
		outputs: map[uint64]*eth.OutputResponse{
			100: {
				OutputRoot: eth.Bytes32(prestateOutputRoot),
			},
			101: {
				OutputRoot: eth.Bytes32(firstOutputRoot),
			},
		},
	}
	return &OutputTraceProvider{
		logger:        testlog.Logger(t, log.LvlInfo),
		rollupClient:  &rollupClient,
		prestateBlock: prestateBlock,
	}, &rollupClient
}

type stubRollupClient struct {
	errorsOnPrestateFetch bool
	outputs               map[uint64]*eth.OutputResponse
}

func (s *stubRollupClient) OutputAtBlock(ctx context.Context, blockNum uint64) (*eth.OutputResponse, error) {
	output, ok := s.outputs[blockNum]
	if !ok || s.errorsOnPrestateFetch {
		return nil, fmt.Errorf("no output at block %d", blockNum)
	}
	return output, nil
}
