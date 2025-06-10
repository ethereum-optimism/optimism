package monitor

import (
	"context"
	"testing"

	"github.com/ethereum/go-ethereum"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
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
}

// TestRPCFinder_ProcessBlock tests the ProcessBlock method of the RPCFinder
// confirming that it calls the BlockReceipts method of the caller and returns the expected jobs
// from the provided receiptsToJobs function
func TestRPCFinder_ProcessBlock(t *testing.T) {

}
