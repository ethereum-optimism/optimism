package derive

import (
	"encoding/binary"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

// L1InfoDepositTxData is the inverse of L1InfoDeposit, to see where the L2 chain is derived from
func L1InfoDepositTxData(data []byte) (nr uint64, time uint64, baseFee *big.Int, blockHash common.Hash, err error) {
	if len(data) != 4+8+8+32+32 {
		err = fmt.Errorf("data is unexpected length: %d", len(data))
		return
	}
	offset := 4
	nr = binary.BigEndian.Uint64(data[offset : offset+8])
	offset += 8
	time = binary.BigEndian.Uint64(data[offset : offset+8])
	offset += 8
	baseFee = new(big.Int).SetBytes(data[offset : offset+32])
	offset += 32
	blockHash.SetBytes(data[offset : offset+32])
	return
}
