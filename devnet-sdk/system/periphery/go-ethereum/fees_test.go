package goethereum

import (
	"context"
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMultiplyBigInt(t *testing.T) {
	type TestCase struct {
		value      *big.Int
		multiplier float64
		expected   *big.Int
	}

	testCases := []TestCase{
		{
			value:      big.NewInt(10),
			multiplier: 1.0,
			expected:   big.NewInt(10),
		},
		{
			value:      big.NewInt(7),
			multiplier: 0.0,
			expected:   big.NewInt(0),
		},
		{
			value:      big.NewInt(10),
			multiplier: 1.01,
			expected:   big.NewInt(11),
		},
		{
			value:      big.NewInt(10),
			multiplier: 1.11,
			expected:   big.NewInt(12),
		},
		{
			value:      big.NewInt(5),
			multiplier: 1.2,
			expected:   big.NewInt(6),
		},
	}

	for _, testCase := range testCases {
		t.Run(fmt.Sprintf("should return %d for %d multplied by %f", testCase.expected.Int64(), testCase.value.Int64(), testCase.multiplier), func(t *testing.T) {
			result := multiplyBigInt(testCase.value, testCase.multiplier)
			require.Equal(t, testCase.expected, result)
		})
	}
}

func TestEstimateEIP1559Fees(t *testing.T) {
	t.Run("if GasFeeCap and GasTipCap are not nil", func(t *testing.T) {
		opts := &bind.TransactOpts{
			GasFeeCap: big.NewInt(1),
			GasTipCap: big.NewInt(2),
		}

		t.Run("should not modify the options", func(t *testing.T) {
			feeEstimator := NewEIP1559FeeEstimator(&mockFeeEthClientImpl{})
			newOpts, err := feeEstimator.EstimateFees(context.Background(), opts)
			require.NoError(t, err)

			require.Equal(t, opts, newOpts)

			// We make sure that we get a copy of the object to prevent mutating the original
			assert.NotSame(t, opts, newOpts)
		})
	})

	t.Run("if the GasTipCap is nil", func(t *testing.T) {
		defaultOpts := &bind.TransactOpts{
			GasFeeCap: big.NewInt(1),
			From:      common.Address{},
			Nonce:     big.NewInt(64),
		}

		t.Run("should return an error if the client returns an error", func(t *testing.T) {
			tipCapErr := fmt.Errorf("tip cap error")
			feeEstimator := NewEIP1559FeeEstimator(&mockFeeEthClientImpl{
				tipCapErr: tipCapErr,
			})
			_, err := feeEstimator.EstimateFees(context.Background(), defaultOpts)
			require.Equal(t, tipCapErr, err)
		})

		t.Run("with default tip multiplier", func(t *testing.T) {
			t.Run("should set the GasTipCap to the client's suggested tip cap", func(t *testing.T) {
				tipCapValue := big.NewInt(5)
				feeEstimator := NewEIP1559FeeEstimator(&mockFeeEthClientImpl{
					tipCapValue: tipCapValue,
				})

				newOpts, err := feeEstimator.EstimateFees(context.Background(), defaultOpts)
				require.NoError(t, err)

				// We create a new opts with the expected tip cap added
				expectedOpts := *defaultOpts
				expectedOpts.GasTipCap = tipCapValue

				// We check that the tip has been added
				require.Equal(t, &expectedOpts, newOpts)

				// We make sure that we get a copy of the object to prevent mutating the original
				assert.NotSame(t, defaultOpts, newOpts)
			})
		})

		t.Run("with custom tip multiplier", func(t *testing.T) {
			t.Run("should set the GasTipCap to the client's suggested tip cap multplied by the tip multiplier", func(t *testing.T) {
				tipCapValue := big.NewInt(5)
				tipMultiplier := 10.0
				// The expected tip is a product of the tip cap and the tip multiplier
				expectedTip := big.NewInt(50)

				// We create a fee estimator with a custom tip multiplier
				feeEstimator := NewEIP1559FeeEstimator(&mockFeeEthClientImpl{
					tipCapValue: tipCapValue,
				}, WithEIP1559TipMultiplier(tipMultiplier))

				newOpts, err := feeEstimator.EstimateFees(context.Background(), defaultOpts)
				require.NoError(t, err)

				// We create a new opts with the expected tip cap added
				expectedOpts := *defaultOpts
				expectedOpts.GasTipCap = expectedTip

				// We check that the tip has been added
				require.Equal(t, &expectedOpts, newOpts)

				// We make sure that we get a copy of the object to prevent mutating the original
				assert.NotSame(t, defaultOpts, newOpts)
			})
		})
	})

	t.Run("if the GasFeeCap is nil", func(t *testing.T) {
		defaultOpts := &bind.TransactOpts{
			GasTipCap: big.NewInt(1),
			From:      common.Address{},
			Nonce:     big.NewInt(64),
		}

		t.Run("should return an error if the client returns an error", func(t *testing.T) {
			blockErr := fmt.Errorf("tip cap error")
			feeEstimator := NewEIP1559FeeEstimator(&mockFeeEthClientImpl{
				blockErr: blockErr,
			})
			_, err := feeEstimator.EstimateFees(context.Background(), defaultOpts)
			require.Equal(t, blockErr, err)
		})

		t.Run("should set the GasFeeCap to the sum of block base fee and tip", func(t *testing.T) {
			baseFeeValue := big.NewInt(5)
			blockValue := types.NewBlock(&types.Header{
				BaseFee: baseFeeValue,
				Time:    0,
			}, nil, nil, nil, &mockBlockType{})

			// We expect the total gas cap to be the base fee plus the tip cap
			expectedGas := big.NewInt(0).Add(baseFeeValue, defaultOpts.GasTipCap)

			feeEstimator := NewEIP1559FeeEstimator(&mockFeeEthClientImpl{
				blockValue: blockValue,
			})

			newOpts, err := feeEstimator.EstimateFees(context.Background(), defaultOpts)
			require.NoError(t, err)

			// We create a new opts with the expected fee cap added
			expectedOpts := *defaultOpts
			expectedOpts.GasFeeCap = expectedGas

			// We check that the tip has been added
			require.Equal(t, &expectedOpts, newOpts)

			// We make sure that we get a copy of the object to prevent mutating the original
			assert.NotSame(t, defaultOpts, newOpts)
		})

		t.Run("should set the GasFeeCap to nil if the base fee is nil", func(t *testing.T) {
			blockValue := types.NewBlock(&types.Header{
				BaseFee: nil,
				Time:    0,
			}, nil, nil, nil, &mockBlockType{})

			feeEstimator := NewEIP1559FeeEstimator(&mockFeeEthClientImpl{
				blockValue: blockValue,
			})

			newOpts, err := feeEstimator.EstimateFees(context.Background(), defaultOpts)
			require.NoError(t, err)

			// We create a new opts with the expected fee cap added
			expectedOpts := *defaultOpts
			expectedOpts.GasFeeCap = nil

			// We check that the tip has been added
			require.Equal(t, &expectedOpts, newOpts)

			// We make sure that we get a copy of the object to prevent mutating the original
			assert.NotSame(t, defaultOpts, newOpts)
		})

		t.Run("with custom base multiplier", func(t *testing.T) {
			t.Run("should set the GasFeeCap to the block base fee multplied by the base multiplier", func(t *testing.T) {
				baseMultiplier := 1.2
				baseFeeValue := big.NewInt(9)
				blockValue := types.NewBlock(&types.Header{
					BaseFee: baseFeeValue,
					Time:    0,
				}, nil, nil, nil, &mockBlockType{})

				// We expect the total gas cap to be the base fee (9) multiplied by 1.2 (= 10.8, rounded up to 11) plus the tip cap (1)
				expectedGas := big.NewInt(0).Add(big.NewInt(11), defaultOpts.GasTipCap)

				// We create a fee estimator with a custom tip multiplier
				feeEstimator := NewEIP1559FeeEstimator(&mockFeeEthClientImpl{
					blockValue: blockValue,
				}, WithEIP1559BaseMultiplier(baseMultiplier))

				newOpts, err := feeEstimator.EstimateFees(context.Background(), defaultOpts)
				require.NoError(t, err)

				// We create a new opts with the expected tip cap added
				expectedOpts := *defaultOpts
				expectedOpts.GasFeeCap = expectedGas

				// We check that the tip has been added
				require.Equal(t, &expectedOpts, newOpts)

				// We make sure that we get a copy of the object to prevent mutating the original
				assert.NotSame(t, defaultOpts, newOpts)
			})
		})
	})
}

var (
	_ EIP1159FeeEthClient = (*mockFeeEthClientImpl)(nil)

	_ types.BlockType = (*mockBlockType)(nil)
)

type mockFeeEthClientImpl struct {
	blockValue *types.Block
	blockErr   error

	tipCapValue *big.Int
	tipCapErr   error
}

func (m *mockFeeEthClientImpl) BlockByNumber(ctx context.Context, number *big.Int) (*types.Block, error) {
	return m.blockValue, m.blockErr
}

func (m *mockFeeEthClientImpl) SuggestGasTipCap(ctx context.Context) (*big.Int, error) {
	return m.tipCapValue, m.tipCapErr
}

type mockBlockType struct{}

func (m *mockBlockType) HasOptimismWithdrawalsRoot(blkTime uint64) bool {
	return false
}

func (m *mockBlockType) IsIsthmus(blkTime uint64) bool {
	return false
}
