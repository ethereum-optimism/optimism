package fees

import (
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/l2geth/common"
)

func TestPaysEnough(t *testing.T) {
	tests := map[string]struct {
		opts *PaysEnoughOpts
		err  error
	}{
		"missing-gas-price": {
			opts: &PaysEnoughOpts{
				UserGasPrice:     nil,
				ExpectedGasPrice: new(big.Int),
				ThresholdUp:      nil,
				ThresholdDown:    nil,
			},
			err: errMissingInput,
		},
		"missing-fee": {
			opts: &PaysEnoughOpts{
				UserGasPrice:     nil,
				ExpectedGasPrice: nil,
				ThresholdUp:      nil,
				ThresholdDown:    nil,
			},
			err: errMissingInput,
		},
		"equal-fee": {
			opts: &PaysEnoughOpts{
				UserGasPrice:     common.Big1,
				ExpectedGasPrice: common.Big1,
				ThresholdUp:      nil,
				ThresholdDown:    nil,
			},
			err: nil,
		},
		"fee-too-low": {
			opts: &PaysEnoughOpts{
				UserGasPrice:     common.Big1,
				ExpectedGasPrice: common.Big2,
				ThresholdUp:      nil,
				ThresholdDown:    nil,
			},
			err: ErrGasPriceTooLow,
		},
		"fee-threshold-down": {
			opts: &PaysEnoughOpts{
				UserGasPrice:     common.Big1,
				ExpectedGasPrice: common.Big2,
				ThresholdUp:      nil,
				ThresholdDown:    new(big.Float).SetFloat64(0.5),
			},
			err: nil,
		},
		"fee-threshold-up": {
			opts: &PaysEnoughOpts{
				UserGasPrice:     common.Big256,
				ExpectedGasPrice: common.Big1,
				ThresholdUp:      new(big.Float).SetFloat64(1.5),
				ThresholdDown:    nil,
			},
			err: ErrGasPriceTooHigh,
		},
		"fee-too-low-high": {
			opts: &PaysEnoughOpts{
				UserGasPrice:     new(big.Int).SetUint64(10_000),
				ExpectedGasPrice: new(big.Int).SetUint64(1),
				ThresholdUp:      new(big.Float).SetFloat64(3),
				ThresholdDown:    new(big.Float).SetFloat64(0.8),
			},
			err: ErrGasPriceTooHigh,
		},
		"fee-too-low-down": {
			opts: &PaysEnoughOpts{
				UserGasPrice:     new(big.Int).SetUint64(1),
				ExpectedGasPrice: new(big.Int).SetUint64(10_000),
				ThresholdUp:      new(big.Float).SetFloat64(3),
				ThresholdDown:    new(big.Float).SetFloat64(0.8),
			},
			err: ErrGasPriceTooLow,
		},
		"fee-too-low-down-2": {
			opts: &PaysEnoughOpts{
				UserGasPrice:     new(big.Int).SetUint64(0),
				ExpectedGasPrice: new(big.Int).SetUint64(10_000),
				ThresholdUp:      new(big.Float).SetFloat64(3),
				ThresholdDown:    new(big.Float).SetFloat64(0.8),
			},
			err: ErrGasPriceTooLow,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			err := PaysEnough(tt.opts)
			if !errors.Is(err, tt.err) {
				t.Fatalf("%s: got %s, expected %s", name, err, tt.err)
			}
		})
	}
}
