package eth

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type MockL1Client struct {
	mock.Mock
}

func (m *MockL1Client) NonceAt(ctx context.Context, account common.Address, blockNumber *big.Int) (uint64, error) {
	args := m.Called(ctx, account, blockNumber)
	return args.Get(0).(uint64), args.Error(1)
}

func (m *MockL1Client) HeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error) {
	args := m.Called(ctx, number)
	if header, ok := args.Get(0).(*types.Header); ok {
		return header, args.Error(1)
	}
	return nil, args.Error(1)
}

func TestTransactions_checkRecentTxs(t *testing.T) {
	tests := []struct {
		name             string
		currentBlock     int64
		blockConfirms    uint64
		expectedBlockNum uint64
		expectedFound    bool
		blocks           map[int64][]uint64 // maps blockNum --> nonceVal (one for each stubbed call)
	}{
		{
			name:             "nonceDiff_lowerBound",
			currentBlock:     500,
			blockConfirms:    5,
			expectedBlockNum: 496,
			expectedFound:    true,
			blocks: map[int64][]uint64{
				495: {5, 5},
				496: {6, 6},
				497: {6},
				500: {6},
			},
		},
		{
			name:             "nonceDiff_midRange",
			currentBlock:     500,
			blockConfirms:    5,
			expectedBlockNum: 497,
			expectedFound:    true,
			blocks: map[int64][]uint64{
				495: {5},
				496: {5},
				497: {6, 6},
				500: {6},
			},
		},
		{
			name:             "nonceDiff_upperBound",
			currentBlock:     500,
			blockConfirms:    5,
			expectedBlockNum: 500,
			expectedFound:    true,
			blocks: map[int64][]uint64{
				495: {5},
				497: {5},
				498: {5},
				499: {5},
				500: {6, 6},
			},
		},
		{
			name:             "nonce_unchanged",
			currentBlock:     500,
			blockConfirms:    5,
			expectedBlockNum: 495,
			expectedFound:    false,
			blocks: map[int64][]uint64{
				495: {6},
				500: {6},
			},
		},
		{
			name:             "reorg",
			currentBlock:     500,
			blockConfirms:    5,
			expectedBlockNum: 496,
			expectedFound:    true,
			blocks: map[int64][]uint64{
				495: {5, 5, 5},
				496: {7, 7, 7},
				497: {6, 7},
				500: {6, 7},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l1Client := new(MockL1Client)
			ctx := context.Background()

			// Setup mock responses
			l1Client.On("HeaderByNumber", ctx, (*big.Int)(nil)).Return(&types.Header{Number: big.NewInt(tt.currentBlock)}, nil)
			for blockNum, block := range tt.blocks {
				for _, nonce := range block {
					l1Client.On("NonceAt", ctx, common.Address{}, big.NewInt(blockNum)).Return(nonce, nil).Once()
				}
			}

			blockNum, found, err := CheckRecentTxs(ctx, l1Client, 5, common.Address{})
			require.NoError(t, err)
			require.Equal(t, tt.expectedFound, found)
			require.Equal(t, tt.expectedBlockNum, blockNum)

			l1Client.AssertExpectations(t)
		})
	}
}
