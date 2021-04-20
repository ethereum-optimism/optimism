package core

import (
	"math/big"
)

// RollupBaseTxSize is the encoded rollup transaction's compressed size excluding
// the variable length data.
// Ref: https://github.com/ethereum-optimism/optimism/blob/91a9a3dcddf534ae1c906133b6d8e015a23c463b/packages/contracts/contracts/optimistic-ethereum/OVM/predeploys/OVM_SequencerEntrypoint.sol#L47
const RollupBaseTxSize int = 96

// CalculateFee calculates the fee that must be paid to the Rollup sequencer, taking into
// account the cost of publishing data to L1.
// Returns: (RollupBaseTxSize + len(data)) * dataPrice + executionPrice * gasUsed
func CalculateRollupFee(data []byte, gasUsed uint64, dataPrice, executionPrice *big.Int) *big.Int {
	dataLen := int64(RollupBaseTxSize + len(data))
	// get the data fee
	dataFee := new(big.Int).Mul(dataPrice, big.NewInt(dataLen))
	executionFee := new(big.Int).Mul(executionPrice, new(big.Int).SetUint64(gasUsed))
	fee := new(big.Int).Add(dataFee, executionFee)
	return fee
}
