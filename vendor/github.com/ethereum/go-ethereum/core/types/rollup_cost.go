// Copyright 2022 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package types

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
)

const (
	// The two 4-byte Ecotone fee scalar values are packed into the same storage slot as the 8-byte
	// sequence number and have the following Solidity offsets within the slot. Note that Solidity
	// offsets correspond to the last byte of the value in the slot, counting backwards from the
	// end of the slot. For example, The 8-byte sequence number has offset 0, and is therefore
	// stored as big-endian format in bytes [24:32) of the slot.
	BaseFeeScalarSlotOffset     = 12 // bytes [16:20) of the slot
	BlobBaseFeeScalarSlotOffset = 8  // bytes [20:24) of the slot

	// scalarSectionStart is the beginning of the scalar values segment in the slot
	// array. baseFeeScalar is in the first four bytes of the segment, blobBaseFeeScalar the next
	// four.
	scalarSectionStart = 32 - BaseFeeScalarSlotOffset - 4
)

func init() {
	if BlobBaseFeeScalarSlotOffset != BaseFeeScalarSlotOffset-4 {
		panic("this code assumes the scalars are at adjacent positions in the scalars slot")
	}
}

var (
	// BedrockL1AttributesSelector is the function selector indicating Bedrock style L1 gas
	// attributes.
	BedrockL1AttributesSelector = []byte{0x01, 0x5d, 0x8e, 0xb9}
	// EcotoneL1AttributesSelector is the selector indicating Ecotone style L1 gas attributes.
	EcotoneL1AttributesSelector = []byte{0x44, 0x0a, 0x5e, 0x20}

	// L1BlockAddr is the address of the L1Block contract which stores the L1 gas attributes.
	L1BlockAddr = common.HexToAddress("0x4200000000000000000000000000000000000015")

	L1BaseFeeSlot = common.BigToHash(big.NewInt(1))
	OverheadSlot  = common.BigToHash(big.NewInt(5))
	ScalarSlot    = common.BigToHash(big.NewInt(6))

	// L1BlobBaseFeeSlot was added with the Ecotone upgrade and stores the blobBaseFee L1 gas
	// attribute.
	L1BlobBaseFeeSlot = common.BigToHash(big.NewInt(7))
	// L1FeeScalarsSlot as of the Ecotone upgrade stores the 32-bit basefeeScalar and
	// blobBaseFeeScalar L1 gas attributes at offsets `BaseFeeScalarSlotOffset` and
	// `BlobBaseFeeScalarSlotOffset` respectively.
	L1FeeScalarsSlot = common.BigToHash(big.NewInt(3))

	oneMillion     = big.NewInt(1_000_000)
	ecotoneDivisor = big.NewInt(1_000_000 * 16)
	fjordDivisor   = big.NewInt(1_000_000_000_000)
	sixteen        = big.NewInt(16)

	L1CostIntercept  = big.NewInt(-42_585_600)
	L1CostFastlzCoef = big.NewInt(836_500)

	MinTransactionSize       = big.NewInt(100)
	MinTransactionSizeScaled = new(big.Int).Mul(MinTransactionSize, big.NewInt(1e6))

	emptyScalars = make([]byte, 8)
)

// RollupCostData is a transaction structure that caches data for quickly computing the data
// availability costs for the transaction.
type RollupCostData struct {
	Zeroes, Ones uint64
	FastLzSize   uint64
}

type StateGetter interface {
	GetState(common.Address, common.Hash) common.Hash
}

// L1CostFunc is used in the state transition to determine the data availability fee charged to the
// sender of non-Deposit transactions.  It returns nil if no data availability fee is charged.
type L1CostFunc func(rcd RollupCostData, blockTime uint64) *big.Int

// l1CostFunc is an internal version of L1CostFunc that also returns the gasUsed for use in
// receipts.
type l1CostFunc func(rcd RollupCostData) (fee, gasUsed *big.Int)

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

// NewL1CostFunc returns a function used for calculating data availability fees, or nil if this is
// not an op-stack chain.
func NewL1CostFunc(config *params.ChainConfig, statedb StateGetter) L1CostFunc {
	if config.Optimism == nil {
		return nil
	}
	forBlock := ^uint64(0)
	var cachedFunc l1CostFunc
	selectFunc := func(blockTime uint64) l1CostFunc {
		if !config.IsOptimismEcotone(blockTime) {
			return newL1CostFuncBedrock(config, statedb, blockTime)
		}

		// Note: the various state variables below are not initialized from the DB until this
		// point to allow deposit transactions from the block to be processed first by state
		// transition.  This behavior is consensus critical!
		l1FeeScalars := statedb.GetState(L1BlockAddr, L1FeeScalarsSlot).Bytes()
		l1BlobBaseFee := statedb.GetState(L1BlockAddr, L1BlobBaseFeeSlot).Big()
		l1BaseFee := statedb.GetState(L1BlockAddr, L1BaseFeeSlot).Big()

		// Edge case: the very first Ecotone block requires we use the Bedrock cost
		// function. We detect this scenario by checking if the Ecotone parameters are
		// unset. Note here we rely on assumption that the scalar parameters are adjacent
		// in the buffer and l1BaseFeeScalar comes first. We need to check this prior to
		// other forks, as the first block of Fjord and Ecotone could be the same block.
		firstEcotoneBlock := l1BlobBaseFee.BitLen() == 0 &&
			bytes.Equal(emptyScalars, l1FeeScalars[scalarSectionStart:scalarSectionStart+8])
		if firstEcotoneBlock {
			log.Info("using bedrock l1 cost func for first Ecotone block", "time", blockTime)
			return newL1CostFuncBedrock(config, statedb, blockTime)
		}

		l1BaseFeeScalar, l1BlobBaseFeeScalar := extractEcotoneFeeParams(l1FeeScalars)

		if config.IsOptimismFjord(blockTime) {
			return NewL1CostFuncFjord(
				l1BaseFee,
				l1BlobBaseFee,
				l1BaseFeeScalar,
				l1BlobBaseFeeScalar,
			)
		} else {
			return newL1CostFuncEcotone(l1BaseFee, l1BlobBaseFee, l1BaseFeeScalar, l1BlobBaseFeeScalar)
		}
	}

	return func(rollupCostData RollupCostData, blockTime uint64) *big.Int {
		if rollupCostData == (RollupCostData{}) {
			return nil // Do not charge if there is no rollup cost-data (e.g. RPC call or deposit).
		}
		if forBlock != blockTime {
			if forBlock != ^uint64(0) {
				// best practice is not to re-use l1 cost funcs across different blocks, but we
				// make it work just in case.
				log.Info("l1 cost func re-used for different L1 block", "oldTime", forBlock, "newTime", blockTime)
			}
			forBlock = blockTime
			cachedFunc = selectFunc(blockTime)
		}
		fee, _ := cachedFunc(rollupCostData)
		return fee
	}
}

// newL1CostFuncBedrock returns an L1 cost function suitable for Bedrock, Regolith, and the first
// block only of the Ecotone upgrade.
func newL1CostFuncBedrock(config *params.ChainConfig, statedb StateGetter, blockTime uint64) l1CostFunc {
	l1BaseFee := statedb.GetState(L1BlockAddr, L1BaseFeeSlot).Big()
	overhead := statedb.GetState(L1BlockAddr, OverheadSlot).Big()
	scalar := statedb.GetState(L1BlockAddr, ScalarSlot).Big()
	isRegolith := config.IsRegolith(blockTime)
	return newL1CostFuncBedrockHelper(l1BaseFee, overhead, scalar, isRegolith)
}

// newL1CostFuncBedrockHelper is lower level version of newL1CostFuncBedrock that expects already
// extracted parameters
func newL1CostFuncBedrockHelper(l1BaseFee, overhead, scalar *big.Int, isRegolith bool) l1CostFunc {
	return func(rollupCostData RollupCostData) (fee, gasUsed *big.Int) {
		if rollupCostData == (RollupCostData{}) {
			return nil, nil // Do not charge if there is no rollup cost-data (e.g. RPC call or deposit)
		}
		gas := rollupCostData.Zeroes * params.TxDataZeroGas
		if isRegolith {
			gas += rollupCostData.Ones * params.TxDataNonZeroGasEIP2028
		} else {
			gas += (rollupCostData.Ones + 68) * params.TxDataNonZeroGasEIP2028
		}
		gasWithOverhead := new(big.Int).SetUint64(gas)
		gasWithOverhead.Add(gasWithOverhead, overhead)
		l1Cost := l1CostHelper(gasWithOverhead, l1BaseFee, scalar)
		return l1Cost, gasWithOverhead
	}
}

// newL1CostFuncEcotone returns an l1 cost function suitable for the Ecotone upgrade except for the
// very first block of the upgrade.
func newL1CostFuncEcotone(l1BaseFee, l1BlobBaseFee, l1BaseFeeScalar, l1BlobBaseFeeScalar *big.Int) l1CostFunc {
	return func(costData RollupCostData) (fee, calldataGasUsed *big.Int) {
		calldataGasUsed = bedrockCalldataGasUsed(costData)

		// Ecotone L1 cost function:
		//
		//   (calldataGas/16)*(l1BaseFee*16*l1BaseFeeScalar + l1BlobBaseFee*l1BlobBaseFeeScalar)/1e6
		//
		// We divide "calldataGas" by 16 to change from units of calldata gas to "estimated # of bytes when
		// compressed". Known as "compressedTxSize" in the spec.
		//
		// Function is actually computed as follows for better precision under integer arithmetic:
		//
		//   calldataGas*(l1BaseFee*16*l1BaseFeeScalar + l1BlobBaseFee*l1BlobBaseFeeScalar)/16e6

		calldataCostPerByte := new(big.Int).Set(l1BaseFee)
		calldataCostPerByte = calldataCostPerByte.Mul(calldataCostPerByte, sixteen)
		calldataCostPerByte = calldataCostPerByte.Mul(calldataCostPerByte, l1BaseFeeScalar)

		blobCostPerByte := new(big.Int).Set(l1BlobBaseFee)
		blobCostPerByte = blobCostPerByte.Mul(blobCostPerByte, l1BlobBaseFeeScalar)

		fee = new(big.Int).Add(calldataCostPerByte, blobCostPerByte)
		fee = fee.Mul(fee, calldataGasUsed)
		fee = fee.Div(fee, ecotoneDivisor)

		return fee, calldataGasUsed
	}
}

type gasParams struct {
	l1BaseFee           *big.Int
	l1BlobBaseFee       *big.Int
	costFunc            l1CostFunc
	feeScalar           *big.Float // pre-ecotone
	l1BaseFeeScalar     *uint32    // post-ecotone
	l1BlobBaseFeeScalar *uint32    // post-ecotone
}

// intToScaledFloat returns scalar/10e6 as a float
func intToScaledFloat(scalar *big.Int) *big.Float {
	fscalar := new(big.Float).SetInt(scalar)
	fdivisor := new(big.Float).SetUint64(1_000_000) // 10**6, i.e. 6 decimals
	return new(big.Float).Quo(fscalar, fdivisor)
}

// extractL1GasParams extracts the gas parameters necessary to compute gas costs from L1 block info
func extractL1GasParams(config *params.ChainConfig, time uint64, data []byte) (gasParams, error) {
	// edge case: for the very first Ecotone block we still need to use the Bedrock
	// function. We detect this edge case by seeing if the function selector is the old one
	// If so, fall through to the pre-ecotone format
	// Both Ecotone and Fjord use the same function selector
	if config.IsEcotone(time) && len(data) >= 4 && !bytes.Equal(data[0:4], BedrockL1AttributesSelector) {
		p, err := extractL1GasParamsPostEcotone(data)
		if err != nil {
			return gasParams{}, err
		}

		if config.IsFjord(time) {
			p.costFunc = NewL1CostFuncFjord(
				p.l1BaseFee,
				p.l1BlobBaseFee,
				big.NewInt(int64(*p.l1BaseFeeScalar)),
				big.NewInt(int64(*p.l1BlobBaseFeeScalar)),
			)
		} else {
			p.costFunc = newL1CostFuncEcotone(
				p.l1BaseFee,
				p.l1BlobBaseFee,
				big.NewInt(int64(*p.l1BaseFeeScalar)),
				big.NewInt(int64(*p.l1BlobBaseFeeScalar)),
			)
		}

		return p, nil
	}
	return extractL1GasParamsPreEcotone(config, time, data)
}

func extractL1GasParamsPreEcotone(config *params.ChainConfig, time uint64, data []byte) (gasParams, error) {
	// data consists of func selector followed by 7 ABI-encoded parameters (32 bytes each)
	if len(data) < 4+32*8 {
		return gasParams{}, fmt.Errorf("expected at least %d L1 info bytes, got %d", 4+32*8, len(data))
	}
	data = data[4:]                                       // trim function selector
	l1BaseFee := new(big.Int).SetBytes(data[32*2 : 32*3]) // arg index 2
	overhead := new(big.Int).SetBytes(data[32*6 : 32*7])  // arg index 6
	scalar := new(big.Int).SetBytes(data[32*7 : 32*8])    // arg index 7
	feeScalar := intToScaledFloat(scalar)                 // legacy: format fee scalar as big Float
	costFunc := newL1CostFuncBedrockHelper(l1BaseFee, overhead, scalar, config.IsRegolith(time))
	return gasParams{
		l1BaseFee: l1BaseFee,
		costFunc:  costFunc,
		feeScalar: feeScalar,
	}, nil
}

// extractL1GasParamsPostEcotone extracts the gas parameters necessary to compute gas from L1 attribute
// info calldata after the Ecotone upgrade, but not for the very first Ecotone block.
func extractL1GasParamsPostEcotone(data []byte) (gasParams, error) {
	if len(data) != 164 {
		return gasParams{}, fmt.Errorf("expected 164 L1 info bytes, got %d", len(data))
	}
	// data layout assumed for Ecotone:
	// offset type varname
	// 0     <selector>
	// 4     uint32 _basefeeScalar
	// 8     uint32 _blobBaseFeeScalar
	// 12    uint64 _sequenceNumber,
	// 20    uint64 _timestamp,
	// 28    uint64 _l1BlockNumber
	// 36    uint256 _basefee,
	// 68    uint256 _blobBaseFee,
	// 100   bytes32 _hash,
	// 132   bytes32 _batcherHash,
	l1BaseFee := new(big.Int).SetBytes(data[36:68])
	l1BlobBaseFee := new(big.Int).SetBytes(data[68:100])
	l1BaseFeeScalar := binary.BigEndian.Uint32(data[4:8])
	l1BlobBaseFeeScalar := binary.BigEndian.Uint32(data[8:12])
	return gasParams{
		l1BaseFee:           l1BaseFee,
		l1BlobBaseFee:       l1BlobBaseFee,
		l1BaseFeeScalar:     &l1BaseFeeScalar,
		l1BlobBaseFeeScalar: &l1BlobBaseFeeScalar,
	}, nil
}

// L1Cost computes the the data availability fee for transactions in blocks prior to the Ecotone
// upgrade. It is used by e2e tests so must remain exported.
func L1Cost(rollupDataGas uint64, l1BaseFee, overhead, scalar *big.Int) *big.Int {
	l1GasUsed := new(big.Int).SetUint64(rollupDataGas)
	l1GasUsed.Add(l1GasUsed, overhead)
	return l1CostHelper(l1GasUsed, l1BaseFee, scalar)
}

func l1CostHelper(gasWithOverhead, l1BaseFee, scalar *big.Int) *big.Int {
	fee := new(big.Int).Set(gasWithOverhead)
	fee.Mul(fee, l1BaseFee).Mul(fee, scalar).Div(fee, oneMillion)
	return fee
}

// NewL1CostFuncFjord returns an l1 cost function suitable for the Fjord upgrade
func NewL1CostFuncFjord(l1BaseFee, l1BlobBaseFee, baseFeeScalar, blobFeeScalar *big.Int) l1CostFunc {
	return func(costData RollupCostData) (fee, calldataGasUsed *big.Int) {
		// Fjord L1 cost function:
		//l1FeeScaled = baseFeeScalar*l1BaseFee*16 + blobFeeScalar*l1BlobBaseFee
		//estimatedSize = max(minTransactionSize, intercept + fastlzCoef*fastlzSize)
		//l1Cost = estimatedSize * l1FeeScaled / 1e12

		scaledL1BaseFee := new(big.Int).Mul(baseFeeScalar, l1BaseFee)
		calldataCostPerByte := new(big.Int).Mul(scaledL1BaseFee, sixteen)
		blobCostPerByte := new(big.Int).Mul(blobFeeScalar, l1BlobBaseFee)
		l1FeeScaled := new(big.Int).Add(calldataCostPerByte, blobCostPerByte)

		fastLzSize := new(big.Int).SetUint64(costData.FastLzSize)
		estimatedSize := new(big.Int).Add(L1CostIntercept, new(big.Int).Mul(L1CostFastlzCoef, fastLzSize))

		if estimatedSize.Cmp(MinTransactionSizeScaled) < 0 {
			estimatedSize.Set(MinTransactionSizeScaled)
		}

		l1CostScaled := new(big.Int).Mul(estimatedSize, l1FeeScaled)
		l1Cost := new(big.Int).Div(l1CostScaled, fjordDivisor)

		calldataGasUsed = new(big.Int).Mul(estimatedSize, new(big.Int).SetUint64(params.TxDataNonZeroGasEIP2028))
		calldataGasUsed.Div(calldataGasUsed, big.NewInt(1e6))

		return l1Cost, calldataGasUsed
	}
}

func extractEcotoneFeeParams(l1FeeParams []byte) (l1BaseFeeScalar, l1BlobBaseFeeScalar *big.Int) {
	offset := scalarSectionStart
	l1BaseFeeScalar = new(big.Int).SetBytes(l1FeeParams[offset : offset+4])
	l1BlobBaseFeeScalar = new(big.Int).SetBytes(l1FeeParams[offset+4 : offset+8])
	return
}

func bedrockCalldataGasUsed(costData RollupCostData) (calldataGasUsed *big.Int) {
	calldataGas := (costData.Zeroes * params.TxDataZeroGas) + (costData.Ones * params.TxDataNonZeroGasEIP2028)
	return new(big.Int).SetUint64(calldataGas)
}

// FlzCompressLen returns the length of the data after compression through FastLZ, based on
// https://github.com/Vectorized/solady/blob/5315d937d79b335c668896d7533ac603adac5315/js/solady.js
func FlzCompressLen(ib []byte) uint32 {
	n := uint32(0)
	ht := make([]uint32, 8192)
	u24 := func(i uint32) uint32 {
		return uint32(ib[i]) | (uint32(ib[i+1]) << 8) | (uint32(ib[i+2]) << 16)
	}
	cmp := func(p uint32, q uint32, e uint32) uint32 {
		l := uint32(0)
		for e -= q; l < e; l++ {
			if ib[p+l] != ib[q+l] {
				e = 0
			}
		}
		return l
	}
	literals := func(r uint32) {
		n += 0x21 * (r / 0x20)
		r %= 0x20
		if r != 0 {
			n += r + 1
		}
	}
	match := func(l uint32) {
		l--
		n += 3 * (l / 262)
		if l%262 >= 6 {
			n += 3
		} else {
			n += 2
		}
	}
	hash := func(v uint32) uint32 {
		return ((2654435769 * v) >> 19) & 0x1fff
	}
	setNextHash := func(ip uint32) uint32 {
		ht[hash(u24(ip))] = ip
		return ip + 1
	}
	a := uint32(0)
	ipLimit := uint32(len(ib)) - 13
	if len(ib) < 13 {
		ipLimit = 0
	}
	for ip := a + 2; ip < ipLimit; {
		r := uint32(0)
		d := uint32(0)
		for {
			s := u24(ip)
			h := hash(s)
			r = ht[h]
			ht[h] = ip
			d = ip - r
			if ip >= ipLimit {
				break
			}
			ip++
			if d <= 0x1fff && s == u24(r) {
				break
			}
		}
		if ip >= ipLimit {
			break
		}
		ip--
		if ip > a {
			literals(ip - a)
		}
		l := cmp(r+3, ip+3, ipLimit+9)
		match(l)
		ip = setNextHash(setNextHash(ip + l))
		a = ip
	}
	literals(uint32(len(ib)) - a)
	return n
}
