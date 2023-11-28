package sources

import (
	"context"
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

// Define a test type for each method
type testFunction func(ctx context.Context, client *PrefetchingEthClient, mockRPC *mockRPC) error

// Define test cases
var testCases = []struct {
	name              string
	testFunc          testFunction
	prefetchingRanges []uint64
}{
	{
		name: "InfoByNumber",
		testFunc: func(ctx context.Context, client *PrefetchingEthClient, mockRPC *mockRPC) error {
			number := uint64(1234)
			block := &types.Block{}
			blockInfo := eth.BlockToInfo(block)
			mockRPC.On("InfoByNumber", ctx, number).Return(blockInfo, nil).Once()
			_, err := client.InfoByNumber(ctx, number)
			return err
		},
		prefetchingRanges: []uint64{0, 1, 5},
	},
	{
		name: "InfoByHash",
		testFunc: func(ctx context.Context, client *PrefetchingEthClient, mockRPC *mockRPC) error {
			hash := common.Hash{}
			block := &types.Block{} // Fill in necessary fields for the block
			blockInfo := eth.BlockToInfo(block)
			mockRPC.On("InfoByHash", ctx, hash).Return(blockInfo, nil).Once()
			_, err := client.InfoByHash(ctx, hash)
			return err
		},
		prefetchingRanges: []uint64{0, 1, 5},
	},
	// Additional test cases for each method...
}

// runTest runs a given test function with a specific prefetching range.
func runTest(t *testing.T, name string, testFunc testFunction, prefetchingRange uint64) {
	ctx := context.Background()
	mockRPC := new(mockRPC)
	ethClient, err := NewEthClient(mockRPC, nil, nil, testEthClientConfig)
	require.NoError(t, err)
	client, err := NewPrefetchingEthClient(ethClient, prefetchingRange)
	require.NoError(t, err)

	err = testFunc(ctx, client, mockRPC)
	require.NoError(t, err)

	mockRPC.AssertExpectations(t)
}

// TestPrefetchingEthClient runs all test cases for each prefetching range.
func TestPrefetchingEthClient(t *testing.T) {
	for _, tc := range testCases {
		for _, prefetchingRange := range tc.prefetchingRanges {
			t.Run(tc.name+string(prefetchingRange), func(t *testing.T) {
				runTest(t, tc.name, tc.testFunc, prefetchingRange)
			})
		}
	}
}
