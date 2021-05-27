package fees

import (
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"
)

// overhead represents the fixed cost of batch submission of a single
// transaction in gas
const overhead uint64 = 4200

// hundredMillion is a constant used in the gas encoding formula
const hundredMillion uint64 = 100_000_000
const hundredBillion uint64 = 100_000_000_000
const FeeScalar uint64 = 1000

var bigHundredMillion = new(big.Int).SetUint64(hundredMillion)
var bigHundredBillion = new(big.Int).SetUint64(hundredBillion)
var bigFeeScalar = new(big.Int).SetUint64(FeeScalar)

// errInvalidGasPrice is the error returned when a user submits an incorrect gas
// price. The gas price must satisfy a particular equation depending on if it
// is a L1 gas price or a L2 gas price
var errInvalidGasPrice = errors.New("rollup fee: invalid gas price")

// CalculateFee calculates the fee that must be paid to the Rollup sequencer, taking into
// account both the cost of submitting the transaction to L1 as well as
// executing the transaction on L2
// fee = (floor((l1GasLimit*l1GasPrice + l2GasLimit*l2GasPrice) / max(tx.gasPrice, 1)) + l2GasLimit) * tx.gasPrice
// where tx.gasPrice is hard coded to 1000 * wei
func CalculateRollupFee(data []byte, l1GasPrice, l2GasLimit, l2GasPrice *big.Int) (*big.Int, error) {
	if err := VerifyGasPrice(l1GasPrice); err != nil {
		return nil, fmt.Errorf("invalid L1 gas price %d: %w", l1GasPrice, err)
	}
	if err := VerifyGasPrice(l2GasPrice); err != nil {
		return nil, fmt.Errorf("invalid L2 gas price %d: %w", l2GasPrice, err)
	}
	l1GasLimit := calculateL1GasLimit(data, overhead)
	l1Fee := new(big.Int).Mul(l1GasPrice, l1GasLimit)
	l2Fee := new(big.Int).Mul(l2GasLimit, l2GasPrice)
	sum := new(big.Int).Add(l1Fee, l2Fee)
	scaled := new(big.Int).Div(sum, bigFeeScalar)
	result := new(big.Int).Add(scaled, l2GasLimit)
	return result, nil
}

// VerifyGasPrice returns an error if the gas price doesn't satisfy
// the constraints.
func VerifyGasPrice(gasPrice *big.Int) error {
	if gasPrice.Cmp(common.Big0) == 0 {
		return nil
	}
	if gasPrice.Cmp(bigHundredBillion) < 0 {
		return fmt.Errorf("too small: %w", errInvalidGasPrice)
	}
	mod := new(big.Int).Mod(gasPrice, bigHundredMillion)
	if mod.Cmp(common.Big0) != 0 {
		return fmt.Errorf("incorrect multiple: %w", errInvalidGasPrice)
	}
	return nil
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

// RoundGasPrice rounds the gas price up to the next valid gas price
func RoundGasPrice(gasPrice *big.Int) *big.Int {
	if gasPrice.Cmp(common.Big0) == 0 {
		return new(big.Int)
	}
	mod := new(big.Int).Mod(gasPrice, bigHundredBillion)
	if mod.Cmp(common.Big0) == 0 {
		return gasPrice
	}
	sum := new(big.Int).Add(gasPrice, bigHundredBillion)
	return new(big.Int).Sub(sum, mod)
}

func DecodeL2GasLimit(gasLimit *big.Int) *big.Int {
	return new(big.Int).Mod(gasLimit, bigHundredMillion)
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
