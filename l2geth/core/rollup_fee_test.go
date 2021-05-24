package core

import (
	"errors"
	"math/big"
	"testing"
)

var roundingL1GasPriceTests = map[string]struct {
	input  uint64
	expect uint64
}{
	"simple":    {10, pow10(8)},
	"one-over":  {pow10(8) + 1, 2 * pow10(8)},
	"exact":     {pow10(8), pow10(8)},
	"one-under": {pow10(8) - 1, pow10(8)},
	"small":     {3, pow10(8)},
	"two":       {2, pow10(8)},
	"one":       {1, pow10(8)},
	"zero":      {0, 0},
}

func TestRoundL1GasPrice(t *testing.T) {
	for name, tt := range roundingL1GasPriceTests {
		t.Run(name, func(t *testing.T) {
			got := RoundL1GasPrice(new(big.Int).SetUint64(tt.input))
			if got.Uint64() != tt.expect {
				t.Fatalf("Mismatched rounding to nearest, got %d expected %d", got, tt.expect)
			}
		})
	}
}

var roundingL2GasPriceTests = map[string]struct {
	input  uint64
	expect uint64
}{
	"simple":    {10, pow10(8) + 1},
	"one-over":  {pow10(8) + 2, 2*pow10(8) + 1},
	"exact":     {pow10(8) + 1, pow10(8) + 1},
	"one-under": {pow10(8), pow10(8) + 1},
	"small":     {3, pow10(8) + 1},
	"two":       {2, pow10(8) + 1},
	"one":       {1, pow10(8) + 1},
	"zero":      {0, 1},
}

func TestRoundL2GasPrice(t *testing.T) {
	for name, tt := range roundingL2GasPriceTests {
		t.Run(name, func(t *testing.T) {
			got := RoundL2GasPrice(new(big.Int).SetUint64(tt.input))
			if got.Uint64() != tt.expect {
				t.Fatalf("Mismatched rounding to nearest, got %d expected %d", got, tt.expect)
			}
		})
	}
}

var l1GasLimitTests = map[string]struct {
	data     []byte
	overhead uint64
	expect   *big.Int
}{
	"simple":          {[]byte{}, 0, big.NewInt(0)},
	"simple-overhead": {[]byte{}, 10, big.NewInt(10)},
	"zeros":           {[]byte{0x00, 0x00, 0x00, 0x00}, 10, big.NewInt(26)},
	"ones":            {[]byte{0x01, 0x02, 0x03, 0x04}, 200, big.NewInt(16*4 + 200)},
}

func TestL1GasLimit(t *testing.T) {
	for name, tt := range l1GasLimitTests {
		t.Run(name, func(t *testing.T) {
			got := calculateL1GasLimit(tt.data, tt.overhead)
			if got.Cmp(tt.expect) != 0 {
				t.Fatal("Calculated gas limit does not match")
			}
		})
	}
}

var feeTests = map[string]struct {
	dataLen    int
	l1GasPrice uint64
	l2GasLimit uint64
	l2GasPrice uint64
	err        error
}{
	"simple":           {100, 100_000_000, 437118, 100_000_001, nil},
	"zero-l2-gasprice": {10, 100_000_000, 196205, 0, errInvalidGasPrice},
	"one-l2-gasprice":  {10, 100_000_000, 196205, 1, nil},
	"zero-l1-gasprice": {10, 0, 196205, 100_000_001, nil},
	"one-l1-gasprice":  {10, 1, 23255, 23254, errInvalidGasPrice},
}

func TestCalculateRollupFee(t *testing.T) {
	for name, tt := range feeTests {
		t.Run(name, func(t *testing.T) {
			data := make([]byte, 0, tt.dataLen)
			l1GasPrice := new(big.Int).SetUint64(tt.l1GasPrice)
			l2GasLimit := new(big.Int).SetUint64(tt.l2GasLimit)
			l2GasPrice := new(big.Int).SetUint64(tt.l2GasPrice)

			fee, err := CalculateRollupFee(data, l1GasPrice, l2GasLimit, l2GasPrice)
			if !errors.Is(err, tt.err) {
				t.Fatalf("Cannot calculate fee: %s", err)
			}

			if err == nil {
				decodedGasLimit := DecodeL2GasLimit(fee)
				if l2GasLimit.Uint64() != decodedGasLimit {
					t.Errorf("rollup fee check failed: expected %d, got %d", l2GasLimit.Uint64(), decodedGasLimit)
				}
			}
		})
	}
}
