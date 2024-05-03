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
		name               string
		currentBlock       uint64
		blockConfirms      uint64
		previousNonceBlock uint64
		expectedBlockNum   uint64
		expectedFound      bool
	}{
		{
			// Blocks       495 496 497 498 499 500
			// Nonce          5   5   5   6   6   6
			// call NonceAt   x   -   x   x   x   x
			name:               "NonceDiff_3Blocks",
			currentBlock:       500,
			blockConfirms:      5,
			previousNonceBlock: 497,
			expectedBlockNum:   498,
			expectedFound:      true,
		},
		{
			// Blocks       495 496 497 498 499 500
			// Nonce          5   5   5   5   5   6
			// call NonceAt   x   -   -   -   x   x
			name:               "NonceDiff_1Block",
			currentBlock:       500,
			blockConfirms:      5,
			previousNonceBlock: 499,
			expectedBlockNum:   500,
			expectedFound:      true,
		},
		{
			// Blocks       495 496 497 498 499 500
			// Nonce          6   6   6   6   6   6
			// call NonceAt   x   -   -   -   -   x
			name:               "NonceUnchanged",
			currentBlock:       500,
			blockConfirms:      5,
			previousNonceBlock: 400,
			expectedBlockNum:   495,
			expectedFound:      false,
		},
	}

	for _, tt := range tests {

		t.Run(tt.name, func(t *testing.T) {
			l1Client := new(MockL1Client)
			ctx := context.Background()

			currentNonce := uint64(6)
			previousNonce := uint64(5)

			l1Client.On("HeaderByNumber", ctx, (*big.Int)(nil)).Return(&types.Header{Number: big.NewInt(int64(tt.currentBlock))}, nil)

			// Setup mock calls for NonceAt, depending on how many times its expected to be called
			if tt.previousNonceBlock < tt.currentBlock-tt.blockConfirms {
				l1Client.On("NonceAt", ctx, common.Address{}, big.NewInt(int64(tt.currentBlock))).Return(currentNonce, nil)
				l1Client.On("NonceAt", ctx, common.Address{}, big.NewInt(int64(tt.currentBlock-tt.blockConfirms))).Return(currentNonce, nil)
			} else {
				for block := tt.currentBlock; block >= (tt.currentBlock - tt.blockConfirms); block-- {
					blockBig := big.NewInt(int64(block))
					if block > (tt.currentBlock-tt.blockConfirms) && block < tt.previousNonceBlock {
						t.Log("skipped block: ", block)
						continue
					} else if block <= tt.previousNonceBlock {
						t.Log("previousNonce set at block: ", block)
						l1Client.On("NonceAt", ctx, common.Address{}, blockBig).Return(previousNonce, nil)
					} else {
						t.Log("currentNonce set at block: ", block)
						l1Client.On("NonceAt", ctx, common.Address{}, blockBig).Return(currentNonce, nil)
					}
				}
			}

			blockNum, found, err := CheckRecentTxs(ctx, l1Client, 5, common.Address{})
			require.NoError(t, err)
			require.Equal(t, tt.expectedBlockNum, blockNum)
			require.Equal(t, tt.expectedFound, found)

			l1Client.AssertExpectations(t)
		})
	}
}
func TestTransactions_checkRecentTxs_reorg(t *testing.T) {
	l1Client := new(MockL1Client)
	ctx := context.Background()

	currentNonce := uint64(6)
	currentBlock := uint64(500)
	blockConfirms := uint64(5)

	l1Client.On("HeaderByNumber", ctx, (*big.Int)(nil)).Return(&types.Header{Number: big.NewInt(int64(currentBlock))}, nil)
	l1Client.On("NonceAt", ctx, common.Address{}, big.NewInt(int64(currentBlock))).Return(currentNonce, nil)

	l1Client.On("NonceAt", ctx, common.Address{}, big.NewInt(int64(currentBlock-blockConfirms))).Return(currentNonce+1, nil)
	l1Client.On("NonceAt", ctx, common.Address{}, big.NewInt(int64(currentBlock-1))).Return(currentNonce, nil)
	l1Client.On("NonceAt", ctx, common.Address{}, big.NewInt(int64(currentBlock-2))).Return(currentNonce, nil)
	l1Client.On("NonceAt", ctx, common.Address{}, big.NewInt(int64(currentBlock-3))).Return(currentNonce, nil)
	l1Client.On("NonceAt", ctx, common.Address{}, big.NewInt(int64(currentBlock-4))).Return(currentNonce, nil)
	l1Client.On("NonceAt", ctx, common.Address{}, big.NewInt(int64(currentBlock-5))).Return(currentNonce, nil)
	l1Client.On("NonceAt", ctx, common.Address{}, big.NewInt(int64(currentBlock-6))).Return(currentNonce, nil)

	blockNum, found, err := CheckRecentTxs(ctx, l1Client, 5, common.Address{})
	require.NoError(t, err)
	require.Equal(t, uint64(495), blockNum)
	require.Equal(t, true, found)

	l1Client.AssertExpectations(t)
}
