package fees

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/params"
)

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
	"simple": {
		dataLen:    10,
		l1GasPrice: params.GWei,
		l2GasLimit: 437118,
		l2GasPrice: params.GWei,
	},
	"zero-l2-gasprice": {
		dataLen:    10,
		l1GasPrice: params.GWei,
		l2GasLimit: 196205,
		l2GasPrice: 0,
	},
	"one-l2-gasprice": {
		dataLen:    10,
		l1GasPrice: params.GWei,
		l2GasLimit: 196205,
		l2GasPrice: 1,
	},
	"zero-l1-gasprice": {
		dataLen:    10,
		l1GasPrice: 0,
		l2GasLimit: 196205,
		l2GasPrice: params.GWei,
	},
	"one-l1-gasprice": {
		dataLen:    10,
		l1GasPrice: 1,
		l2GasLimit: 23255,
		l2GasPrice: params.GWei,
	},
	"zero-gasprices": {
		dataLen:    10,
		l1GasPrice: 0,
		l2GasLimit: 23255,
		l2GasPrice: 0,
	},
	"max-gaslimit": {
		dataLen:    10,
		l1GasPrice: params.GWei,
		l2GasLimit: 99_970_000,
		l2GasPrice: params.GWei,
	},
	"larger-divisor": {
		dataLen:    10,
		l1GasPrice: 0,
		l2GasLimit: 10,
		l2GasPrice: 0,
	},
}

func TestCalculateRollupFee(t *testing.T) {
	for name, tt := range feeTests {
		t.Run(name, func(t *testing.T) {
			data := make([]byte, tt.dataLen)
			l1GasPrice := new(big.Int).SetUint64(tt.l1GasPrice)
			l2GasLimit := new(big.Int).SetUint64(tt.l2GasLimit)
			l2GasPrice := new(big.Int).SetUint64(tt.l2GasPrice)

			fee := EncodeTxGasLimit(data, l1GasPrice, l2GasLimit, l2GasPrice)
			decodedGasLimit := DecodeL2GasLimit(fee)
			roundedL2GasLimit := Ceilmod(l2GasLimit, BigTenThousand)
			if roundedL2GasLimit.Cmp(decodedGasLimit) != 0 {
				t.Errorf("rollup fee check failed: expected %d, got %d", l2GasLimit.Uint64(), decodedGasLimit)
			}
		})
	}
}
