package derive

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

// L1BlockInfo presents the information stored in a L1Block.setL1BlockValues call
type L1BlockInfo struct {
	Number    uint64
	Time      uint64
	BaseFee   *big.Int
	BlockHash common.Hash
	// Not strictly a piece of L1 information. Represents the number of L2 blocks since the start of the epoch,
	// i.e. when the actual L1 info was first introduced.
	SequenceNumber uint64
}

func (info *L1BlockInfo) MarshalBinary() ([]byte, error) {
	data := make([]byte, 4+32+32+32+32+32)
	offset := 0
	copy(data[offset:4], L1InfoFuncBytes4)
	offset += 4
	binary.BigEndian.PutUint64(data[offset+24:offset+32], info.Number)
	offset += 32
	binary.BigEndian.PutUint64(data[offset+24:offset+32], info.Time)
	offset += 32
	info.BaseFee.FillBytes(data[offset : offset+32])
	offset += 32
	copy(data[offset:offset+32], info.BlockHash.Bytes())
	offset += 32
	binary.BigEndian.PutUint64(data[offset+24:offset+32], info.SequenceNumber)
	return data, nil
}

func (info *L1BlockInfo) UnmarshalBinary(data []byte) error {
	if len(data) != 4+32+32+32+32+32 {
		return fmt.Errorf("data is unexpected length: %d", len(data))
	}
	var padding [24]byte
	offset := 4
	info.Number = binary.BigEndian.Uint64(data[offset+24 : offset+32])
	if !bytes.Equal(data[offset:offset+24], padding[:]) {
		return fmt.Errorf("l1 info number exceeds uint64 bounds: %x", data[offset:offset+32])
	}
	offset += 32
	info.Time = binary.BigEndian.Uint64(data[offset+24 : offset+32])
	if !bytes.Equal(data[offset:offset+24], padding[:]) {
		return fmt.Errorf("l1 info time exceeds uint64 bounds: %x", data[offset:offset+32])
	}
	offset += 32
	info.BaseFee = new(big.Int).SetBytes(data[offset : offset+32])
	offset += 32
	info.BlockHash.SetBytes(data[offset : offset+32])
	offset += 32
	info.SequenceNumber = binary.BigEndian.Uint64(data[offset+24 : offset+32])
	if !bytes.Equal(data[offset:offset+24], padding[:]) {
		return fmt.Errorf("l1 info sequence number exceeds uint64 bounds: %x", data[offset:offset+32])
	}
	return nil
}

// L1InfoDepositTxData is the inverse of L1InfoDeposit, to see where the L2 chain is derived from
func L1InfoDepositTxData(data []byte) (L1BlockInfo, error) {
	var info L1BlockInfo
	err := info.UnmarshalBinary(data)
	return info, err
}
