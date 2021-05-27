package fees

import (
	"errors"
	"math"
	"math/big"
	"testing"
)

var roundingGasPriceTests = map[string]struct {
	input  uint64
	expect uint64
}{
	"simple":    {10, hundredBillion},
	"one-over":  {hundredBillion + 1, 2 * hundredBillion},
	"exact":     {hundredBillion, hundredBillion},
	"one-under": {hundredBillion - 1, hundredBillion},
	"small":     {3, hundredBillion},
	"two":       {2, hundredBillion},
	"one":       {1, hundredBillion},
	"zero":      {0, 0},
}

func TestRoundGasPrice(t *testing.T) {
	for name, tt := range roundingGasPriceTests {
		t.Run(name, func(t *testing.T) {
			got := RoundGasPrice(new(big.Int).SetUint64(tt.input))
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
	"simple": {
		dataLen:    10,
		l1GasPrice: hundredBillion,
		l2GasLimit: 437118,
		l2GasPrice: hundredBillion,
		err:        nil,
	},
	"zero-l2-gasprice": {
		dataLen:    10,
		l1GasPrice: hundredBillion,
		l2GasLimit: 196205,
		l2GasPrice: 0,
		err:        nil,
	},
	"one-l2-gasprice": {
		dataLen:    10,
		l1GasPrice: hundredBillion,
		l2GasLimit: 196205,
		l2GasPrice: 1,
		err:        errInvalidGasPrice,
	},
	"zero-l1-gasprice": {
		dataLen:    10,
		l1GasPrice: 0,
		l2GasLimit: 196205,
		l2GasPrice: hundredBillion,
		err:        nil,
	},
	"one-l1-gasprice": {
		dataLen:    10,
		l1GasPrice: 1,
		l2GasLimit: 23255,
		l2GasPrice: hundredBillion,
		err:        errInvalidGasPrice,
	},
	"zero-gasprices": {
		dataLen:    10,
		l1GasPrice: 0,
		l2GasLimit: 23255,
		l2GasPrice: 0,
		err:        nil,
	},
	"bad-l2-gasprice": {
		dataLen:    10,
		l1GasPrice: 0,
		l2GasLimit: 23255,
		l2GasPrice: hundredBillion - 1,
		err:        errInvalidGasPrice,
	},
	"bad-l1-gasprice": {
		dataLen:    10,
		l1GasPrice: hundredBillion - 1,
		l2GasLimit: 44654,
		l2GasPrice: hundredBillion,
		err:        errInvalidGasPrice,
	},
	"max-gaslimit": {
		dataLen:    10,
		l1GasPrice: hundredBillion,
		l2GasLimit: 0x4ffffff,
		l2GasPrice: hundredBillion,
		err:        nil,
	},
	"larger-divisor": {
		dataLen:    10,
		l1GasPrice: 0,
		l2GasLimit: 10,
		l2GasPrice: 0,
		err:        nil,
	},
}

func TestCalculateRollupFee(t *testing.T) {
	for name, tt := range feeTests {
		t.Run(name, func(t *testing.T) {
			data := make([]byte, tt.dataLen)
			l1GasPrice := new(big.Int).SetUint64(tt.l1GasPrice)
			l2GasLimit := new(big.Int).SetUint64(tt.l2GasLimit)
			l2GasPrice := new(big.Int).SetUint64(tt.l2GasPrice)

			fee, err := CalculateRollupFee(data, l1GasPrice, l2GasLimit, l2GasPrice)
			if !errors.Is(err, tt.err) {
				t.Fatalf("Cannot calculate fee: %s", err)
			}

			if err == nil {
				decodedGasLimit := DecodeL2GasLimit(fee)
				if l2GasLimit.Cmp(decodedGasLimit) != 0 {
					t.Errorf("rollup fee check failed: expected %d, got %d", l2GasLimit.Uint64(), decodedGasLimit)
				}
			}
		})
	}
}

func pow10(x int) uint64 {
	return uint64(math.Pow10(x))
}
