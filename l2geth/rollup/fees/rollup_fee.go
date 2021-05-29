package fees

import (
	"math/big"

	"github.com/ethereum/go-ethereum/params"
)

// overhead represents the fixed cost of batch submission of a single
// transaction in gas.
const overhead uint64 = 4200 + 200*params.TxDataNonZeroGasEIP2028

// hundredMillion is a constant used in the gas encoding formula
const hundredMillion uint64 = 100_000_000

// feeScalar is used to scale the calculations in EncodeL2GasLimit
// to prevent them from being too large
const feeScalar uint64 = 1000

// L2GasPrice is a constant that determines the result of `eth_gasPrice`
// It is scaled upwards by 50%
const L2GasPrice uint64 = feeScalar + (feeScalar / 2)

// BigL2GasPrice is the L2GasPrice as type big.Int
var BigL2GasPrice = new(big.Int).SetUint64(L2GasPrice)
var bigFeeScalar = new(big.Int).SetUint64(feeScalar)
var bigHundredMillion = new(big.Int).SetUint64(hundredMillion)

// EncodeTxGasLimit computes the `tx.gasLimit` based on the L1/L2 gas prices and
// the L2 gas limit. The L2 gas limit is encoded inside of the lower order bits
// of the number like so: [          | l2GasLimit ]
//                        [      tx.gaslimit      ]
// The lower order bits must be large enough to fit the L2 gas limit, so 10**8
// is chosen. If higher order bits collide with any bits from the L2 gas limit,
// the L2 gas limit will not be able to be decoded.
// An explicit design goal of this scheme was to make the L2 gas limit be human
// readable. The entire number is interpreted as the gas limit when computing
// the fee, so increasing the L2 Gas limit will increase the fee paid.
// The calculation is:
// floor((l1GasLimit*l1GasPrice + l2GasLimit*l2GasPrice) / max(tx.gasPrice, 1)) + l2GasLimit
// where tx.gasPrice is hard coded to 1000 * wei
// Note that for simplicity purposes, only the calldata is passed into this
// function when in reality the RLP encoded transaction should be. The
// additional cost is added to the overhead constant to prevent the need to RLP
// encode transactions during calls to `eth_estimateGas`
func EncodeTxGasLimit(data []byte, l1GasPrice, l2GasLimit, l2GasPrice *big.Int) *big.Int {
	l1GasLimit := calculateL1GasLimit(data, overhead)
	l1Fee := new(big.Int).Mul(l1GasPrice, l1GasLimit)
	l2Fee := new(big.Int).Mul(l2GasLimit, l2GasPrice)
	sum := new(big.Int).Add(l1Fee, l2Fee)
	scaled := new(big.Int).Div(sum, bigFeeScalar)
	remainder := new(big.Int).Mod(scaled, bigHundredMillion)
	scaledSum := new(big.Int).Add(scaled, bigHundredMillion)
	rounded := new(big.Int).Sub(scaledSum, remainder)
	result := new(big.Int).Add(rounded, l2GasLimit)
	return result
}

// DecodeL2GasLimit decodes the L2 gas limit from an encoded L2 gas limit
func DecodeL2GasLimit(gasLimit *big.Int) *big.Int {
	return new(big.Int).Mod(gasLimit, bigHundredMillion)
}

// calculateL1GasLimit computes the L1 gasLimit based on the calldata and
// constant sized overhead. The overhead can be decreased as the cost of the
// batch submission goes down via contract optimizations. This will not overflow
// under standard network conditions.
func calculateL1GasLimit(data []byte, overhead uint64) *big.Int {
	zeroes, ones := zeroesAndOnes(data)
	zeroesCost := zeroes * params.TxDataZeroGas
	onesCost := ones * params.TxDataNonZeroGasEIP2028
	gasLimit := zeroesCost + onesCost + overhead
	return new(big.Int).SetUint64(gasLimit)
}

func zeroesAndOnes(data []byte) (uint64, uint64) {
	var zeroes uint64
	var ones uint64
	for _, byt := range data {
		if byt == 0 {
			zeroes++
		} else {
			ones++
		}
	}
	return zeroes, ones
}
