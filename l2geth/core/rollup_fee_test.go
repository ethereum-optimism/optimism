package core

import (
	"math/big"
	"testing"
)

var roundingL1GasPriceTests = map[string]struct {
	input  uint64
	expect uint64
}{
	"simple":    {10, pow10(8)},
	"one-over":  {pow10(8) + 1, pow10(8)},
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
			got := RoundL1GasPrice(tt.input)
			if got != tt.expect {
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
	"one-over":  {pow10(8), pow10(8) + 1},
	"exact":     {pow10(8) + 1, pow10(8) + 1},
	"one-under": {pow10(8), pow10(8) + 1},
	"small":     {3, pow10(8) + 1},
	"two":       {2, pow10(8) + 1},
	"one":       {1, pow10(8) + 1},
	"zero":      {0, 0},
}

func TestRoundL2GasPrice(t *testing.T) {
	for name, tt := range roundingL2GasPriceTests {
		t.Run(name, func(t *testing.T) {
			got := RoundL2GasPrice(tt.input)
			if got != tt.expect {
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
}{
	"simple": {100, RoundL1GasPrice(2000), 437118, RoundL2GasPrice(5000)},
}

func TestCalculateRollupFee(t *testing.T) {
	for name, tt := range feeTests {
		t.Run(name, func(t *testing.T) {
			data := make([]byte, 0, tt.dataLen)
			l1GasPrice := new(big.Int).SetUint64(tt.l1GasPrice)
			l2GasLimit := new(big.Int).SetUint64(tt.l2GasLimit)
			l2GasPrice := new(big.Int).SetUint64(tt.l2GasPrice)

			fee, err := CalculateRollupFee(data, l1GasPrice, l2GasLimit, l2GasPrice)
			if err != nil {
				t.Fatal("Cannot calculate fee")
			}

			decodedGasLimit := DecodeL2GasLimit(fee)

			if l2GasLimit.Uint64() != decodedGasLimit {
				t.Errorf("rollup fee check failed: expected %d, got %d", l2GasLimit.Uint64(), decodedGasLimit)
			}
		})
	}
}
