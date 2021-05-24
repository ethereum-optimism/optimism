package core

import (
	"errors"
	"fmt"
	"math"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"
)

// overhead represents the fixed cost of batch submission of a single
// transaction in gas
const overhead uint64 = 4200

// hundredMillion is a constant used in the gas encoding formula
const hundredMillion uint64 = 100_000_000

var bigHundredMillion = new(big.Int).SetUint64(hundredMillion)

// errInvalidGasPrice is the error returned when a user submits an incorrect gas
// price. The gas price must satisfy a particular equation depending on if it
// is a L1 gas price or a L2 gas price
var errInvalidGasPrice = errors.New("rollup fee: invalid gas price")

// CalculateFee calculates the fee that must be paid to the Rollup sequencer, taking into
// account the cost of publishing data to L1.
// l2_gas_price * l2_gas_limit + l1_gas_price * l1_gas_used
// where the l2 gas price must satisfy the equation `x * (10**8)` + 1
// and the l1 gas price must satisfy the equation `x * (10**8)`
func CalculateRollupFee(data []byte, l1GasPrice, l2GasLimit, l2GasPrice *big.Int) (uint64, error) {
	if err := VerifyL1GasPrice(l1GasPrice); err != nil {
		return 0, fmt.Errorf("invalid L1 gas price %d: %w", l1GasPrice, err)
	}
	if err := VerifyL2GasPrice(l2GasPrice); err != nil {
		return 0, fmt.Errorf("invalid L2 gas price %d: %w", l2GasPrice, err)
	}
	l1GasLimit := calculateL1GasLimit(data, overhead)
	l1Fee := new(big.Int).Mul(l1GasPrice, l1GasLimit)
	l2Fee := new(big.Int).Mul(l2GasLimit, l2GasPrice)
	fee := new(big.Int).Add(l1Fee, l2Fee)
	return fee.Uint64(), nil
}

// calculateL1GasLimit computes the L1 gasLimit based on the calldata and
// constant sized overhead. The overhead can be decreased as the cost of the
// batch submission goes down via contract optimizations.
func calculateL1GasLimit(data []byte, overhead uint64) *big.Int {
	zeroes, ones := zeroesAndOnes(data)
	zeroesCost := zeroes * params.TxDataZeroGas
	onesCost := ones * params.TxDataNonZeroGasEIP2028
	gasLimit := zeroesCost + onesCost + overhead
	return new(big.Int).SetUint64(gasLimit)
}

// ceilModOneHundredMillion rounds the input integer up to the nearest modulus
// of one hundred million
func ceilModOneHundredMillion(num *big.Int) *big.Int {
	if new(big.Int).Mod(num, bigHundredMillion).Cmp(common.Big0) == 0 {
		return num
	}
	sum := new(big.Int).Add(num, bigHundredMillion)
	mod := new(big.Int).Mod(num, bigHundredMillion)
	return new(big.Int).Sub(sum, mod)
}

// VerifyL1GasPrice returns an error if the number is an invalid possible L1 gas
// price
func VerifyL1GasPrice(l1GasPrice *big.Int) error {
	if new(big.Int).Mod(l1GasPrice, bigHundredMillion).Cmp(common.Big0) != 0 {
		return errInvalidGasPrice
	}
	return nil
}

// VerifyL2GasPrice returns an error if the number is an invalid possible L2 gas
// price
func VerifyL2GasPrice(l2GasPrice *big.Int) error {
	isNonZero := l2GasPrice.Cmp(common.Big0) != 0
	isNotModHundredMillion := new(big.Int).Mod(l2GasPrice, bigHundredMillion).Cmp(common.Big1) != 0
	if isNonZero && isNotModHundredMillion {
		return errInvalidGasPrice
	}
	if l2GasPrice.Cmp(common.Big0) == 0 {
		return errInvalidGasPrice
	}
	return nil
}

// RoundL1GasPrice returns a ceilModOneHundredMillion where 0
// is the identity function
func RoundL1GasPrice(gasPrice *big.Int) *big.Int {
	return ceilModOneHundredMillion(gasPrice)
}

// RoundL2GasPriceBig implements the algorithm:
// if gasPrice is 0; return 1
// if gasPrice is 1; return 1**9 + 1
// return ceilModOneHundredMillion(gasPrice-1)+1
func RoundL2GasPrice(gasPrice *big.Int) *big.Int {
	if gasPrice.Cmp(common.Big0) == 0 {
		return big.NewInt(1)
	}
	if gasPrice.Cmp(common.Big1) == 0 {
		return new(big.Int).Add(bigHundredMillion, common.Big1)
	}
	gp := new(big.Int).Sub(gasPrice, common.Big1)
	mod := ceilModOneHundredMillion(gp)
	return new(big.Int).Add(mod, common.Big1)
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
