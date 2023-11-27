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

// TestPrefetchingEthClient_ConsecutiveCalls tests consecutive calls to InfoByNumber.
func TestPrefetchingEthClient_ConsecutiveCalls(t *testing.T) {
	ctx := context.Background()
	mockRPC := new(MockRPC)
	mockEthClient := new(MockEthClient)

	// Assume the prefetch range is set to 3 for this test
	prefetchRange := 3

	// Initialize PrefetchingEthClient
	client, err := NewPrefetchingEthClient(mockRPC, nil, nil, testEthClientConfig)
	require.NoError(t, err)

	firstBlockNumber := uint64(1234)

	// Generate and mock responses for the first block and prefetched blocks
	_, firstRpcHeader := randHeader()
	firstRpcHeader.Number = hexutil.Uint64(firstBlockNumber) // Adjust the block number on rpcHeader
	firstBlockInfo, _ := firstRpcHeader.Info(true, false)    // Convert to BlockInfo
	mockEthClient.On("InfoByNumber", ctx, firstBlockNumber).Return(firstBlockInfo, nil).Once()

	for i := uint64(1); i <= uint64(prefetchRange); i++ {
		_, prefetchedRpcHeader := randHeader()
		prefetchedRpcHeader.Number = hexutil.Uint64(firstBlockNumber + i) // Adjust the block number on rpcHeader
		prefetchedBlockInfo, _ := prefetchedRpcHeader.Info(true, false)   // Convert to BlockInfo
		mockEthClient.On("InfoByNumber", ctx, firstBlockNumber+i).Return(prefetchedBlockInfo, nil).Once()
	}

	// First call to InfoByNumber
	info1, err1 := client.InfoByNumber(ctx, firstBlockNumber)
	require.NoError(t, err1)
	require.Equal(t, firstBlockInfo, info1)

	// Second call to InfoByNumber for a subsequent block, expecting to use prefetched data
	secondBlockNumber := firstBlockNumber + 1
	info2, err2 := client.InfoByNumber(ctx, secondBlockNumber)
	require.NoError(t, err2)

	// Validate that the second info is from the prefetched data
	_, expectedSecondRpcHeader := randHeader()
	expectedSecondRpcHeader.Number = hexutil.Uint64(secondBlockNumber)
	expectedSecondBlockInfo, _ := expectedSecondRpcHeader.Info(true, false)
	require.Equal(t, expectedSecondBlockInfo, info2)

	// Ensure the mock EthClient was called only once for the first block
	mockEthClient.AssertNumberOfCalls(t, "InfoByNumber", 1)
	mockEthClient.AssertCalled(t, "InfoByNumber", ctx, firstBlockNumber)

	// Additional assertions to verify that the prefetched blocks were not redundantly fetched
	for i := uint64(2); i <= uint64(prefetchRange); i++ {
		mockEthClient.AssertNumberOfCalls(t, "InfoByNumber", 1)
	}
}

// TestPrefetchingEthClient_CachingBehavior tests the caching behavior of prefetched blocks.
func TestPrefetchingEthClient_CachingBehavior(t *testing.T) {
	ctx := context.Background()
	mockRPC := new(MockRPC)
	mockEthClient := new(MockEthClient)

	// Assume the prefetch range is set to 3 for this test
	prefetchRange := 3

	// Initialize PrefetchingEthClient
	client, err := NewPrefetchingEthClient(mockRPC, nil, nil, testEthClientConfig)
	require.NoError(t, err)

	blockNumber := uint64(1234)

	// Generate and mock responses for the requested block and prefetched blocks
	_, requestedRpcHeader := randHeader()
	requestedRpcHeader.Number = hexutil.Uint64(blockNumber)       // Adjust the block number on rpcHeader
	requestedBlockInfo, _ := requestedRpcHeader.Info(true, false) // Convert to BlockInfo
	mockEthClient.On("InfoByNumber", ctx, blockNumber).Return(requestedBlockInfo, nil).Once()

	for i := uint64(1); i <= uint64(prefetchRange); i++ {
		_, prefetchedRpcHeader := randHeader()
		prefetchedRpcHeader.Number = hexutil.Uint64(blockNumber + i)    // Adjust the block number on rpcHeader
		prefetchedBlockInfo, _ := prefetchedRpcHeader.Info(true, false) // Convert to BlockInfo
		mockEthClient.On("InfoByNumber", ctx, blockNumber+i).Return(prefetchedBlockInfo, nil).Once()
	}

	// Call to InfoByNumber for the requested block
	_, err = client.InfoByNumber(ctx, blockNumber)
	require.NoError(t, err)

	// Call to InfoByNumber for the next block, which should be served from cache
	nextBlockNumber := blockNumber + 1
	info, err := client.InfoByNumber(ctx, nextBlockNumber)
	require.NoError(t, err)

	// Verify that the mock EthClient was not called again for the next block, implying it was served from cache
	mockEthClient.AssertNotCalled(t, "InfoByNumber", ctx, nextBlockNumber)

	// Assert the information received is for the correct block (next block)
	_, expectedNextRpcHeader := randHeader()
	expectedNextRpcHeader.Number = hexutil.Uint64(nextBlockNumber)
	expectedNextBlockInfo, _ := expectedNextRpcHeader.Info(true, false)
	require.Equal(t, expectedNextBlockInfo, info)

	// Clean up any mock expectations
	mockEthClient.AssertExpectations(t)
}

// TestPrefetchingEthClient_PrefetchRange tests the range of blocks prefetched.
func TestPrefetchingEthClient_PrefetchRange(t *testing.T) {
	ctx := context.Background()
	mockRPC := new(MockRPC)
	mockEthClient := new(MockEthClient)

	// Initialize PrefetchingEthClient with a specific prefetch range
	prefetchRange := 5 // Example range for prefetching
	client, err := NewPrefetchingEthClient(mockRPC, nil, nil, testEthClientConfig)
	require.NoError(t, err)

	blockNumber := uint64(1234)

	// Generate and mock responses for the requested block and prefetched blocks
	_, requestedRpcHeader := randHeader()
	requestedRpcHeader.Number = hexutil.Uint64(blockNumber)       // Adjust the block number on rpcHeader
	requestedBlockInfo, _ := requestedRpcHeader.Info(true, false) // Convert to BlockInfo
	mockEthClient.On("InfoByNumber", ctx, blockNumber).Return(requestedBlockInfo, nil).Once()

	for i := uint64(1); i < uint64(prefetchRange); i++ {
		_, prefetchedRpcHeader := randHeader()
		prefetchedRpcHeader.Number = hexutil.Uint64(blockNumber + i)    // Adjust the block number on rpcHeader
		prefetchedBlockInfo, _ := prefetchedRpcHeader.Info(true, false) // Convert to BlockInfo
		mockEthClient.On("InfoByNumber", ctx, blockNumber+i).Return(prefetchedBlockInfo, nil).Once()
	}

	// Call to InfoByNumber for the requested block
	_, err = client.InfoByNumber(ctx, blockNumber)
	require.NoError(t, err)

	// Ensure the mock EthClient was called for the specified range of blocks
	for i := uint64(0); i < uint64(prefetchRange); i++ {
		mockEthClient.AssertCalled(t, "InfoByNumber", ctx, blockNumber+i)
	}

	// Clean up any mock expectations
	mockEthClient.AssertExpectations(t)
}

// // TestPrefetchingEthClient_NewPrefetchingEthClient tests the constructor.
// func TestPrefetchingEthClient_NewPrefetchingEthClient(t *testing.T) {
// 	mockRPC := new(MockRPC)
// 	logger := log.NewNopLogger()                            // Using a no-op logger for simplicity
// 	metrics := caching.NewMetrics(prometheus.NewRegistry()) // Mock metrics, replace with actual if needed

// 	// Valid parameters
// 	client, err := NewPrefetchingEthClient(mockRPC, logger, metrics, testEthClientConfig)
// 	require.NoError(t, err)
// 	require.NotNil(t, client)

// 	// Test case with invalid parameters, expecting an error
// 	// This will depend on what constitutes an invalid parameter for your implementation.
// 	// For example, passing a nil RPC client, a nil logger, or an invalid configuration.
// 	invalidClient, invalidErr := NewPrefetchingEthClient(nil, nil, nil, nil)
// 	require.Error(t, invalidErr)
// 	require.Nil(t, invalidClient)

// 	// Additional test cases can be added to simulate different error scenarios
// 	// and to ensure that the constructor correctly handles them.
// }

// // TestPrefetchingEthClient_ErrorHandling tests how errors are handled.
// func TestPrefetchingEthClient_ErrorHandling(t *testing.T) {
// 	ctx := context.Background()
// 	mockRPC := new(MockRPC)
// 	mockEthClient := new(MockEthClient)

// 	// Initialize PrefetchingEthClient
// 	client, err := NewPrefetchingEthClient(mockRPC, nil, nil, testEthClientConfig)
// 	require.NoError(t, err)

// 	blockNumber := uint64(1234)
// 	expectedError := errors.New("mock error")

// 	// Mock the EthClient to return an error for the requested block
// 	mockEthClient.On("InfoByNumber", ctx, blockNumber).Return(eth.BlockInfo{}, expectedError)

// 	// Call to InfoByNumber should propagate the error
// 	_, err = client.InfoByNumber(ctx, blockNumber)
// 	require.Error(t, err)
// 	require.Equal(t, expectedError, err)

// 	// Additional tests can be added to simulate different error scenarios
// 	// For instance, testing error handling during prefetching of subsequent blocks
// 	for i := uint64(1); i <= 3; i++ {
// 		prefetchedBlockNumber := blockNumber + i
// 		mockEthClient.On("InfoByNumber", ctx, prefetchedBlockNumber).Return(eth.BlockInfo{}, expectedError)
// 		_, err = client.InfoByNumber(ctx, prefetchedBlockNumber)
// 		require.Error(t, err)
// 		require.Equal(t, expectedError, err)
// 	}
// }

// // TestPrefetchingEthClient_StateInspection (Optional, might be covered nicely by previous tests ) - Test state inspection functionality.
// func TestPrefetchingEthClient_StateInspection(t *testing.T) {
// 	ctx := context.Background()
// 	mockRPC := new(MockRPC)
// 	mockEthClient := new(MockEthClient)

// 	// Initialize PrefetchingEthClient
// 	client, err := NewPrefetchingEthClient(mockRPC, nil, nil, testEthClientConfig)
// 	require.NoError(t, err)

// 	blockNumber := uint64(1234)
// 	blockInfo := eth.BlockInfo{} // Mock response for the block

// 	// Mock the response for the block
// 	mockEthClient.On("InfoByNumber", ctx, blockNumber).Return(blockInfo, nil)

// 	// Call to InfoByNumber
// 	_, err = client.InfoByNumber(ctx, blockNumber)
// 	require.NoError(t, err)

// 	// Inspect the state of prefetching
// 	// This will depend on the method you add to PrefetchingEthClient for state inspection
// 	// ...

// 	// Assertions to verify the state of prefetching
// 	// ...
// }
