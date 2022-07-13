package derive

import (
	"encoding/binary"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

var (
	DepositEventABI     = "TransactionDeposited(address,address,uint256,bytes)"
	DepositEventABIHash = crypto.Keccak256Hash([]byte(DepositEventABI))
)

// UnmarshalDepositLogEvent decodes an EVM log entry emitted by the deposit contract into typed deposit data.
//
// parse log data for:
//     event TransactionDeposited(
//         address indexed from,
//         address indexed to,
//         uint256 indexed version,
//         bytes opaqueData
//     );
//
// Additionally, the event log-index and
func UnmarshalDepositLogEvent(ev *types.Log) (*types.DepositTx, error) {
	if len(ev.Topics) != 4 {
		return nil, fmt.Errorf("expected 4 event topics (event identity, indexed from, indexed to, indexed version), got %d", len(ev.Topics))
	}
	if ev.Topics[0] != DepositEventABIHash {
		return nil, fmt.Errorf("invalid deposit event selector: %s, expected %s", ev.Topics[0], DepositEventABIHash)
	}
	if ev.Topics[3] != [32]byte{} {
		return nil, fmt.Errorf("invalid deposit version (only version 0 is supported), got %s", ev.Topics[3].String())

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

	// HACK: slice off the offset/length field of the singular bytes field.
	// This enables the rest of the ABI decoding logic to work.
	ev.Data = ev.Data[64:]

	// The remainder of the data is tighly packed deposit data.
	// unindexed data
	offset := uint64(0)

	// uint256 mint
	dep.Mint = new(big.Int).SetBytes(ev.Data[offset : offset+32])
	// 0 mint is represented as nil to skip minting code
	if dep.Mint.Cmp(new(big.Int)) == 0 {
		dep.Mint = nil
	}
	offset += 32

	// uint256 value
	dep.Value = new(big.Int).SetBytes(ev.Data[offset : offset+32])
	offset += 32

	// uint64 gas
	gas := new(big.Int).SetBytes(ev.Data[offset : offset+8])
	if !gas.IsUint64() {
		return nil, fmt.Errorf("bad gas value: %x", ev.Data[offset:offset+8])
	}
	dep.Gas = gas.Uint64()
	offset += 8

	// uint8 isCreation
	// isCreation: If the boolean byte is 1 then dep.To will stay nil,
	// and it will create a contract using L2 account nonce to determine the created address.
	if ev.Data[offset+1] == 0 {
		dep.To = &to
	}
	offset += 1

	// The remainder of the opaqueData is the transaction data (without length prefix).
	// The data may be padded to a multiple of 32 bytes
	txDataLen := uint64(len(ev.Data)) - offset

	// remaining bytes fill the data
	dep.Data = ev.Data[offset : offset+txDataLen]
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
		common.BigToHash(new(big.Int).SetUint64(0)),
	}

	data := make([]byte, 6*32)
	offset := 0

	// First 32 bytes are the offset, and the value will always be 0x20.
	new (big.Int).SetUint64(32).FillBytes(data[offset:32])
	offset += 32

	// Next 32 bytes are the length
	new (big.Int).SetUint64(uint64(len(deposit.Data))).FillBytes(data[offset : offset+32])
	offset += 32

	// uint256 mint
	if deposit.Mint != nil {
		deposit.Mint.FillBytes(data[offset : offset+32])
	}
	offset += 32

	// uint256 value
	deposit.Value.FillBytes(data[offset : offset+32])
	offset += 32

	// uint64 gas
	binary.BigEndian.PutUint64(data[offset:offset+8], deposit.Gas)
	offset += 8

	// uint8 isCreation
	if deposit.To == nil { // isCreation
		data[offset+1] = 1
	}
	offset += 1

	// Remaining bytes fill the event data
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
