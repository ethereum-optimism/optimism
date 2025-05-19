package monitor

import (
	"context"
	"log/slog"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"
)

// mockFinderClient implements FinderClient interface for testing
type mockFinderClient struct {
	blockReceiptsFunc    func(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) ([]*ethTypes.Receipt, error)
	subscribeNewHeadFunc func(ctx context.Context, ch chan<- *ethTypes.Header) (ethereum.Subscription, error)
	err                  error
}

func (m *mockFinderClient) BlockReceipts(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) ([]*ethTypes.Receipt, error) {
	if m.blockReceiptsFunc != nil {
		return m.blockReceiptsFunc(ctx, blockNrOrHash)
	}
	return nil, m.err
}

func (m *mockFinderClient) SubscribeNewHead(ctx context.Context, ch chan<- *ethTypes.Header) (ethereum.Subscription, error) {
	if m.subscribeNewHeadFunc != nil {
		return m.subscribeNewHeadFunc(ctx, ch)
	}
	return nil, m.err
}

func mockReceiptsToCases(receipts []*ethTypes.Receipt) []*Job {
	return nil
}

func mockCallback(job *Job) {
}

func TestRPCFinder_StartStop(t *testing.T) {
	client := &mockFinderClient{}
	logger := testlog.Logger(t, slog.LevelDebug)
	finder := NewFinder(eth.ChainIDFromUInt64(1), client, mockReceiptsToCases, mockCallback, logger)

	require.NoError(t, finder.Start(context.Background()))
	require.NoError(t, finder.Stop())

	require.Eventually(t, func() bool {
		return finder.Stopped()
	}, time.Second, 100*time.Millisecond)
}

// TestRPCFinder_ProcessBlock tests the ProcessBlock method of the RPCFinder
// confirming that it calls the BlockReceipts method of the caller and returns the expected jobs
// from the provided receiptsToJobs function
func TestRPCFinder_ProcessBlock(t *testing.T) {
	client := &mockFinderClient{}
	logger := testlog.Logger(t, slog.LevelDebug)

	expectedReceipts := []*ethTypes.Receipt{
		{
			Type: ethTypes.LegacyTxType,
		},
		{
			Type: ethTypes.AccessListTxType,
		},
	}
	expectedJobs := []*Job{
		{status: []jobStatus{jobStatusUnknown}},
		{status: []jobStatus{jobStatusUnknown}},
	}
	client.blockReceiptsFunc = func(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) ([]*ethTypes.Receipt, error) {
		return expectedReceipts, nil
	}
	called := false
	receiptsToJobs := func(receipts []*ethTypes.Receipt) []*Job {
		require.Equal(t, expectedReceipts, receipts)
		called = true
		return expectedJobs
	}

	finder := NewFinder(eth.ChainIDFromUInt64(1), client, receiptsToJobs, mockCallback, logger)

	jobs, err := finder.ProcessBlock(context.Background(), &ethTypes.Header{Number: big.NewInt(1)})
	require.NoError(t, err)
	require.Equal(t, expectedJobs, jobs)
	require.True(t, called)
}
