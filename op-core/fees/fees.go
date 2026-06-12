// Package fees implements the OP-Stack L1 data-availability cost and operator-fee
// calculations. It is pure arithmetic over byte counts and big integers and depends only on
// the standard library and go-ethereum's core/types (for [TxRollupCostData]) and params (for
// the EIP-2028 gas schedule); it has no dependency on op-core/types, so the two packages
// remain cycle-free siblings.
//
// It covers the L1 cost functions from Bedrock through Fjord and the Isthmus and Jovian
// operator-fee formulas. Consumers import the package as opfees.
package fees

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
)

var (
	oneHundred     = big.NewInt(100)
	oneMillion     = big.NewInt(1_000_000)
	sixteen        = big.NewInt(16)
	ecotoneDivisor = big.NewInt(1_000_000 * 16)
	fjordDivisor   = big.NewInt(1_000_000_000_000)

	// L1CostIntercept and L1CostFastlzCoef are the coefficients of the Fjord linear
	// regression that estimates a transaction's compressed DA size from its FastLZ size.
	L1CostIntercept  = big.NewInt(-42_585_600)
	L1CostFastlzCoef = big.NewInt(836_500)

	// MinTransactionSize is the lower bound the Fjord DA-size estimate is clamped to.
	MinTransactionSize       = big.NewInt(100)
	MinTransactionSizeScaled = new(big.Int).Mul(MinTransactionSize, big.NewInt(1e6))
)

// RollupCostData carries the byte counts of a transaction's L1 representation needed to
// compute its data-availability cost.
type RollupCostData struct {
	Zeroes, Ones uint64
	FastLzSize   uint64
}

// L1CostFunc returns the data-availability fee charged to the sender of a non-deposit
// transaction. It returns nil when no DA fee applies.
type L1CostFunc func(rcd RollupCostData, blockTime uint64) *big.Int

// NewRollupCostData computes the RollupCostData for the given L1 transaction bytes.
func NewRollupCostData(data []byte) (out RollupCostData) {
	for _, b := range data {
		if b == 0 {
			out.Zeroes++
		} else {
			out.Ones++
		}
	}
	out.FastLzSize = uint64(FlzCompressLen(data))
	return out
}

// TxRollupCostData computes the RollupCostData for a transaction. It replaces op-geth's
// (*types.Transaction).RollupCostData() method: deposits incur no DA cost, and the cost
// data is derived from the full binary-encoded transaction, not just its calldata.
func TxRollupCostData(tx *types.Transaction) RollupCostData {
	if tx.Type() == types.DepositTxType {
		return RollupCostData{}
	}
	data, err := tx.MarshalBinary()
	if err != nil {
		// A value that exists as a *types.Transaction was decoded or constructed
		// successfully, so re-encoding it cannot fail.
		panic(fmt.Errorf("marshaling transaction for rollup cost data: %w", err))
	}
	return NewRollupCostData(data)
}

// L1CostBedrock computes the Bedrock-era data-availability fee: (rollupDataGas + overhead) *
// l1BaseFee * scalar / 1_000_000. rollupDataGas is the calldata gas of the transaction.
func L1CostBedrock(rollupDataGas uint64, l1BaseFee, overhead, scalar *big.Int) *big.Int {
	fee := new(big.Int).SetUint64(rollupDataGas)
	fee.Add(fee, overhead)
	fee.Mul(fee, l1BaseFee).Mul(fee, scalar).Div(fee, oneMillion)
	return fee
}

// NewL1CostFuncEcotone returns an L1CostFunc for the Ecotone upgrade (excluding the very
// first Ecotone block, which still uses the Bedrock function). The returned function ignores
// blockTime: the fee parameters are fixed at construction.
func NewL1CostFuncEcotone(l1BaseFee, l1BlobBaseFee, l1BaseFeeScalar, l1BlobBaseFeeScalar *big.Int) L1CostFunc {
	return func(costData RollupCostData, _ uint64) *big.Int {
		// Ecotone L1 cost function, computed as
		//   calldataGas * (l1BaseFee*16*l1BaseFeeScalar + l1BlobBaseFee*l1BlobBaseFeeScalar) / 16e6
		// for better integer-arithmetic precision.
		calldataGasUsed := bedrockCalldataGasUsed(costData)
		calldataCostPerByte := new(big.Int).Mul(l1BaseFee, sixteen)
		calldataCostPerByte.Mul(calldataCostPerByte, l1BaseFeeScalar)
		blobCostPerByte := new(big.Int).Mul(l1BlobBaseFee, l1BlobBaseFeeScalar)
		fee := new(big.Int).Add(calldataCostPerByte, blobCostPerByte)
		fee.Mul(fee, calldataGasUsed)
		return fee.Div(fee, ecotoneDivisor)
	}
}

// bedrockCalldataGasUsed returns the calldata gas of a transaction under the EIP-2028 schedule.
func bedrockCalldataGasUsed(costData RollupCostData) *big.Int {
	calldataGas := (costData.Zeroes * params.TxDataZeroGas) + (costData.Ones * params.TxDataNonZeroGasEIP2028)
	return new(big.Int).SetUint64(calldataGas)
}

// NewL1CostFuncFjord returns an L1CostFunc for the Fjord upgrade. The returned function
// ignores blockTime: the fee parameters are fixed at construction.
func NewL1CostFuncFjord(l1BaseFee, l1BlobBaseFee, baseFeeScalar, blobFeeScalar *big.Int) L1CostFunc {
	return func(costData RollupCostData, _ uint64) *big.Int {
		// Fjord L1 cost function:
		//   l1FeeScaled   = baseFeeScalar*l1BaseFee*16 + blobFeeScalar*l1BlobBaseFee
		//   estimatedSize = max(minTransactionSize, intercept + fastlzCoef*fastlzSize)
		//   l1Cost        = estimatedSize * l1FeeScaled / 1e12
		scaledL1BaseFee := new(big.Int).Mul(baseFeeScalar, l1BaseFee)
		calldataCostPerByte := new(big.Int).Mul(scaledL1BaseFee, sixteen)
		blobCostPerByte := new(big.Int).Mul(blobFeeScalar, l1BlobBaseFee)
		l1FeeScaled := new(big.Int).Add(calldataCostPerByte, blobCostPerByte)
		estimatedSize := costData.estimatedDASizeScaled()
		l1CostScaled := new(big.Int).Mul(estimatedSize, l1FeeScaled)
		return new(big.Int).Div(l1CostScaled, fjordDivisor)
	}
}

// OperatorCostIsthmus computes the Isthmus operator fee charged to a transaction:
// (gasUsed * scalar / 1_000_000) + constant, where scalar and constant are the
// operatorFeeScalar and operatorFeeConstant L1 attributes.
func OperatorCostIsthmus(gasUsed, scalar, constant uint64) *big.Int {
	fee := new(big.Int).SetUint64(gasUsed)
	fee.Mul(fee, new(big.Int).SetUint64(scalar))
	fee.Div(fee, oneMillion)
	return fee.Add(fee, new(big.Int).SetUint64(constant))
}

// OperatorCostJovian computes the Jovian operator fee charged to a transaction:
// (gasUsed * scalar * 100) + constant, where scalar and constant are the
// operatorFeeScalar and operatorFeeConstant L1 attributes. Jovian replaces the
// Isthmus division by 1_000_000 with a multiplication by 100.
func OperatorCostJovian(gasUsed, scalar, constant uint64) *big.Int {
	fee := new(big.Int).SetUint64(gasUsed)
	fee.Mul(fee, new(big.Int).SetUint64(scalar))
	fee.Mul(fee, oneHundred)
	return fee.Add(fee, new(big.Int).SetUint64(constant))
}

// estimatedDASizeScaled estimates the number of bytes the transaction occupies in its DA
// batch using the Fjord linear-regression model, scaled up by 1e6.
func (cd RollupCostData) estimatedDASizeScaled() *big.Int {
	fastLzSize := new(big.Int).SetUint64(cd.FastLzSize)
	estimatedSize := new(big.Int).Add(L1CostIntercept, new(big.Int).Mul(L1CostFastlzCoef, fastLzSize))
	if estimatedSize.Cmp(MinTransactionSizeScaled) < 0 {
		estimatedSize.Set(MinTransactionSizeScaled)
	}
	return estimatedSize
}

// EstimatedDASize estimates the number of bytes the transaction occupies in its DA batch
// using the Fjord linear-regression model.
func (cd RollupCostData) EstimatedDASize() *big.Int {
	b := cd.estimatedDASizeScaled()
	return b.Div(b, big.NewInt(1e6))
}
