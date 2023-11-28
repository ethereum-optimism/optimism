package sources

import (
	"context"
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockEthClient is a mock of EthClient used for testing.
type MockEthClient struct {
	mock.Mock
}

func (m *MockEthClient) InfoByNumber(ctx context.Context, number uint64) (eth.BlockInfo, error) {
	args := m.Called(ctx, number)
	return args.Get(0).(eth.BlockInfo), args.Error(1)
}

// // MockRPC is a mock of the RPC client, similar to mockRPC but tailored for PrefetchingEthClient.
// // Add any additional methods that might be required for testing.

// // ... [MockRPC implementation] ...

// var _ client.RPC = (*MockRPC)(nil)

// TestPrefetchingEthClient_InfoByNumber tests the basic functionality of InfoByNumber.
func TestPrefetchingEthClient_InfoByNumber(t *testing.T) {
	ctx := context.Background()
	mockRPC := new(MockRPC)
	mockEthClient := new(MockEthClient)

	// Assume the prefetch range is set to 3 for this test
	prefetchRange := 3

	// Initialize PrefetchingEthClient
	client, err := NewPrefetchingEthClient(mockRPC, nil, nil, testEthClientConfig)
	require.NoError(t, err)

	// The block number for which InfoByNumber will be called
	requestedBlockNumber := uint64(1234)

	// Generate a random rpcHeader for the requested block
	_, requestedRpcHeader := randHeader()
	requestedRpcHeader.Number = hexutil.Uint64(requestedBlockNumber) // Adjust the block number on rpcHeader
	requestedBlockInfo, _ := requestedRpcHeader.Info(true, false)    // Convert to BlockInfo

	// Mock the response for the requested block
	mockEthClient.On("InfoByNumber", ctx, requestedBlockNumber).Return(requestedBlockInfo, nil).Once()

	// Mock responses for subsequent prefetched blocks
	for i := uint64(1); i <= uint64(prefetchRange); i++ {
		_, prefetchedRpcHeader := randHeader()
		prefetchedRpcHeader.Number = hexutil.Uint64(requestedBlockNumber + i) // Adjust the block number on rpcHeader
		prefetchedBlockInfo, _ := prefetchedRpcHeader.Info(true, false)       // Convert to BlockInfo
		mockEthClient.On("InfoByNumber", ctx, requestedBlockNumber+i).Return(prefetchedBlockInfo, nil).Once()
	}

	// Call to InfoByNumber for the requested block
	info, err := client.InfoByNumber(ctx, requestedBlockNumber)
	require.NoError(t, err)
	require.Equal(t, requestedBlockInfo, info)

	// Verify that the internal EthClient was called for the requested block
	mockEthClient.AssertCalled(t, "InfoByNumber", ctx, requestedBlockNumber)

	// Verify that the internal EthClient was also called for the subsequent prefetched blocks
	for i := uint64(1); i <= uint64(prefetchRange); i++ {
		mockEthClient.AssertCalled(t, "InfoByNumber", ctx, requestedBlockNumber+i)
	}

	// Clean up any mock expectations
	mockEthClient.AssertExpectations(t)
}
