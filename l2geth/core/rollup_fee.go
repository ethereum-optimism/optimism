package core

import (
	"math/big"

	"github.com/ethereum/go-ethereum/params"
)

// RollupBatchOverhead is the number of additional bytes of overhead that a
// transaction batch requires in addition to the transactions.

var (
	// Assuming 200 txs in a batch, 2688 gas per transaction
	// Assuming 250 stateroots in a batch, 1473 gas per stateroot
	overhead       = new(big.Int).SetUint64(2688 + 1473)
	maxGasLimit    = new(big.Int).SetUint64(10_000_000)
	scalarValue    = new(big.Int).SetUint64(1)
	scalarDecimals = new(big.Int).SetUint64(0)
	big10          = new(big.Int).SetUint64(10)
)

// CalculateFee calculates the fee that must be paid to the Rollup sequencer, taking into
// account the cost of publishing data to L1.
// The following formula is used:
// overhead = 2688 + 1473
// dataCost = (4 * zeroDataBytes) + (16 * nonZeroDataBytes)
// l1GasCost = dataCost + overhead
// l1Fee = l1GasCost * l1GasPrice
// executionFee = executionPrice * gasLimit
// scalar = scalarValue / 10 ** scalarDecimals
// estimateGas = scalar * (l1Fee + executionFee) * (maxGasLimit + gasLimit)
// final fee = estimateGas * gasPrice
func CalculateRollupFee(data []byte, gasUsed, dataPrice, executionPrice *big.Int) uint64 {
	zeroes, ones := zeroesAndOnes(data)
	zeroesCost := new(big.Int).SetUint64(zeroes * params.TxDataZeroGas)
	onesCost := new(big.Int).SetUint64(ones * params.TxDataNonZeroGasEIP2028)
	dataCost := new(big.Int).Add(zeroesCost, onesCost)
	l1GasCost := new(big.Int).Add(dataCost, overhead)

	// dataPrice is l1GasPrice
	l1Fee := new(big.Int).Mul(l1GasCost, dataPrice)
	executionFee := new(big.Int).Mul(executionPrice, gasUsed)

	fee1 := new(big.Int).Mul(l1Fee, executionFee)
	fee2 := new(big.Int).Mul(maxGasLimit, gasUsed)
	fee := new(big.Int).Add(fee1, fee2)

	scalar := new(big.Int).Exp(big10, scalarDecimals, nil)
	scalar = scalar.Mul(scalar, scalarValue)
	result := new(big.Int).Mul(fee, scalar)

	return result.Uint64()
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
