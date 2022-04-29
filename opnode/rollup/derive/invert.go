package derive

import (
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

// setL1BlockValues(uint256 _number, uint256 _timestamp, uint256 _basefee, bytes32 _hash)
type L1BlockInfo struct {
	Number    uint64
	Time      uint64
	BaseFee   *big.Int
	BlockHash common.Hash
}

// L1InfoDepositTxData is the inverse of L1InfoDeposit, to see where the L2 chain is derived from
func L1InfoDepositTxData(data []byte) (nr uint64, time uint64, baseFee *big.Int, blockHash common.Hash, err error) {
	info, err := L1InfoDepositTxDataToStruct(data)
	return info.Number, info.Time, info.BaseFee, info.BlockHash, err
}

// L1InfoDepositTxDataToStruct is the inverse of L1InfoDeposit, to see where the L2 chain is derived from
func L1InfoDepositTxDataToStruct(data []byte) (L1BlockInfo, error) {
	out := L1BlockInfo{}
	if len(data) != 4+32+32+32+32 {
		return out, fmt.Errorf("data is unexpected length: %d", len(data))
	}

	// Number
	offset := 4 // Selector hash. Should check
	number := new(big.Int)
	number.SetBytes(data[offset : offset+32])
	if !number.IsUint64() {
		return L1BlockInfo{}, errors.New("number does not fit in uint64")
	}
	out.Number = number.Uint64()

	// Timestamp
	offset += 32
	timestamp := new(big.Int)
	timestamp.SetBytes(data[offset : offset+32])
	if !timestamp.IsUint64() {
		return L1BlockInfo{}, errors.New("timestamp does not fit in uint64")
	}
	out.Time = timestamp.Uint64()

	// BaseFee
	offset += 32
	out.BaseFee = new(big.Int).SetBytes(data[offset : offset+32])

	// Hash
	offset += 32
	out.BlockHash.SetBytes(data[offset : offset+32])
	return out, nil
}
