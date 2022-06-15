package derive

import (
	"encoding/binary"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/holiman/uint256"
)

var (
	DepositEventABI     = "TransactionDeposited(address,address,uint256,uint256,uint64,bool,bytes)"
	DepositEventABIHash = crypto.Keccak256Hash([]byte(DepositEventABI))
)

// UnmarshalDepositLogEvent decodes an EVM log entry emitted by the deposit contract into typed deposit data.
//
// parse log data for:
//     event TransactionDeposited(
//    	 address indexed from,
//    	 address indexed to,
//       uint256 mint,
//    	 uint256 value,
//    	 uint64 gasLimit,
//    	 bool isCreation,
//    	 data data
//     );
//
// Additionally, the event log-index and
func UnmarshalDepositLogEvent(ev *types.Log) (*types.DepositTx, error) {
	if len(ev.Topics) != 3 {
		return nil, fmt.Errorf("expected 3 event topics (event identity, indexed from, indexed to)")
	}
	if ev.Topics[0] != DepositEventABIHash {
		return nil, fmt.Errorf("invalid deposit event selector: %s, expected %s", ev.Topics[0], DepositEventABIHash)
	}
	if len(ev.Data) < 6*32 {
		return nil, fmt.Errorf("deposit event data too small (%d bytes): %x", len(ev.Data), ev.Data)
	}

	var dep types.DepositTx

	source := UserDepositSource{
		L1BlockHash: ev.BlockHash,
		LogIndex:    uint64(ev.Index),
	}
	dep.SourceHash = source.SourceHash()

	// indexed 0
	dep.From = common.BytesToAddress(ev.Topics[1][12:])
	// indexed 1
	to := common.BytesToAddress(ev.Topics[2][12:])

	// unindexed data
	offset := uint64(0)

	dep.Mint = new(big.Int).SetBytes(ev.Data[offset : offset+32])
	// 0 mint is represented as nil to skip minting code
	if dep.Mint.Cmp(new(big.Int)) == 0 {
		dep.Mint = nil
	}
	offset += 32

	dep.Value = new(big.Int).SetBytes(ev.Data[offset : offset+32])
	offset += 32

	gas := new(big.Int).SetBytes(ev.Data[offset : offset+32])
	if !gas.IsUint64() {
		return nil, fmt.Errorf("bad gas value: %x", ev.Data[offset:offset+32])
	}
	offset += 32
	dep.Gas = gas.Uint64()
	// isCreation: If the boolean byte is 1 then dep.To will stay nil,
	// and it will create a contract using L2 account nonce to determine the created address.
	if ev.Data[offset+31] == 0 {
		dep.To = &to
	}
	offset += 32
	// dynamic fields are encoded in three parts. The fixed size portion is the offset of the start of the
	// data. The first 32 bytes of a `bytes` object is the length of the bytes. Then are the actual bytes
	// padded out to 32 byte increments.
	var dataOffset uint256.Int
	dataOffset.SetBytes(ev.Data[offset : offset+32])
	offset += 32
	if !dataOffset.Eq(uint256.NewInt(offset)) {
		return nil, fmt.Errorf("incorrect data offset: %v", dataOffset[0])
	}

	var dataLen uint256.Int
	dataLen.SetBytes(ev.Data[offset : offset+32])
	offset += 32

	if !dataLen.IsUint64() {
		return nil, fmt.Errorf("data too large: %s", dataLen.String())
	}
	// The data may be padded to a multiple of 32 bytes
	maxExpectedLen := uint64(len(ev.Data)) - offset
	dataLenU64 := dataLen.Uint64()
	if dataLenU64 > maxExpectedLen {
		return nil, fmt.Errorf("data length too long: %d, expected max %d", dataLenU64, maxExpectedLen)
	}

	// remaining bytes fill the data
	dep.Data = ev.Data[offset : offset+dataLenU64]

	return &dep, nil
}

// MarshalDepositLogEvent returns an EVM log entry that encodes a TransactionDeposited event from the deposit contract.
// This is the reverse of the deposit transaction derivation.
func MarshalDepositLogEvent(depositContractAddr common.Address, deposit *types.DepositTx) *types.Log {
	toBytes := common.Hash{}
	if deposit.To != nil {
		toBytes = deposit.To.Hash()
	}
	topics := []common.Hash{
		DepositEventABIHash,
		deposit.From.Hash(),
		toBytes,
	}

	data := make([]byte, 6*32)
	offset := 0
	if deposit.Mint != nil {
		deposit.Mint.FillBytes(data[offset : offset+32])
	}
	offset += 32

	deposit.Value.FillBytes(data[offset : offset+32])
	offset += 32

	binary.BigEndian.PutUint64(data[offset+24:offset+32], deposit.Gas)
	offset += 32
	if deposit.To == nil { // isCreation
		data[offset+31] = 1
	}
	offset += 32
	binary.BigEndian.PutUint64(data[offset+24:offset+32], 5*32)
	offset += 32
	binary.BigEndian.PutUint64(data[offset+24:offset+32], uint64(len(deposit.Data)))
	data = append(data, deposit.Data...)
	if len(data)%32 != 0 { // pad to multiple of 32
		data = append(data, make([]byte, 32-(len(data)%32))...)
	}

	return &types.Log{
		Address: depositContractAddr,
		Topics:  topics,
		Data:    data,
		Removed: false,

		// ignored (zeroed):
		BlockNumber: 0,
		TxHash:      common.Hash{},
		TxIndex:     0,
		BlockHash:   common.Hash{},
		Index:       0,
	}
}
