package core

import (
	"errors"
	"fmt"
	"math"
	"math/big"

	"github.com/ethereum/go-ethereum/params"
)

// overhead represents the fixed cost of batch submission of a single
// transaction in gas
const overhead uint64 = 4200

// hundredMillion is a constant used in the gas encoding formula
const hundredMillion uint64 = 100_000_000

// errInvalidGasPrice is the error returned when a user submits an incorrect gas
// price. The gas price must satisfy a particular equation depending on if it
// is a L1 gas price or a L2 gas price
var errInvalidGasPrice = errors.New("rollup fee: invalid gas price")

// CalculateFee calculates the fee that must be paid to the Rollup sequencer, taking into
// account the cost of publishing data to L1.
// l2_gas_price * l2_gas_limit + l1_gas_price * l1_gas_used
// where the l2 gas price must satisfy the equation `x * (10**8)`
// and the l1 gas price must satisfy the equation `x * (10**8) + 1`
func CalculateRollupFee(data []byte, l1GasPrice, l2GasLimit, l2GasPrice *big.Int) (uint64, error) {
	if RoundL1GasPrice(l1GasPrice.Uint64()) != l1GasPrice.Uint64() {
		return 0, fmt.Errorf("invalid L1 gas price: %w", errInvalidGasPrice)
	}
	if l2GasPrice.Uint64() > 1 {
		if RoundL2GasPrice(l2GasPrice.Uint64()) != l2GasPrice.Uint64() {
			return 0, fmt.Errorf("invalid L2 gas price: %w", errInvalidGasPrice)
		}
	}
	l1GasLimit := calculateL1GasLimit(data, overhead)
	l1Fee := new(big.Int).Mul(l1GasPrice, l1GasLimit)
	l2Fee := new(big.Int).Mul(l2GasLimit, l2GasPrice)
	fee := new(big.Int).Add(l1Fee, l2Fee)
	return fee.Uint64(), nil
}

func calculateL1GasLimit(data []byte, overhead uint64) *big.Int {
	zeroes, ones := zeroesAndOnes(data)
	zeroesCost := zeroes * params.TxDataZeroGas
	onesCost := ones * params.TxDataNonZeroGasEIP2028
	gasLimit := zeroesCost + onesCost + overhead
	return new(big.Int).SetUint64(gasLimit)
}

func RoundL1GasPrice(gasPrice uint64) uint64 {
	if gasPrice == 0 {
		return gasPrice
	}
	if gasPrice == 1 {
		return hundredMillion
	}
	if gasPrice%hundredMillion < 2 {
		gasPrice += hundredMillion - 2
	} else {
		gasPrice += hundredMillion
	}
	return gasPrice - gasPrice%hundredMillion
}

func RoundL2GasPrice(gasPrice uint64) uint64 {
	if gasPrice == 0 {
		return 1
	}
	if gasPrice == 1 {
		return hundredMillion + 1
	}
	if gasPrice%hundredMillion < 2 {
		gasPrice += hundredMillion - 2
	} else {
		gasPrice += hundredMillion
	}
	return gasPrice - gasPrice%hundredMillion + 1
}

func DecodeL2GasLimit(gasLimit uint64) uint64 {
	return gasLimit % hundredMillion
}

func zeroesAndOnes(data []byte) (uint64, uint64) {
	var zeroes uint64
	for _, byt := range data {
		if byt == 0 {
			zeroes++
		}
	}
	ones := uint64(len(data)) - zeroes
	return zeroes, ones
}

func pow10(x int) uint64 {
	return uint64(math.Pow10(x))
}
