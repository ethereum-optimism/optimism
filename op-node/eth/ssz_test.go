package eth

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/google/go-cmp/cmp"
)

// FuzzExecutionPayloadUnmarshal checks that our SSZ decoding never panics
func FuzzExecutionPayloadUnmarshal(f *testing.F) {
	f.Fuzz(func(t *testing.T, data []byte) {
		var payload ExecutionPayload
		err := payload.UnmarshalSSZ(uint32(len(data)), bytes.NewReader(data))
		if err != nil {
			// not every input is a valid ExecutionPayload, that's ok. Should just not panic.
			return
		}
	})
}

// FuzzExecutionPayloadMarshalUnmarshal checks that our SSZ encoding>decoding round trips properly
func FuzzExecutionPayloadMarshalUnmarshal(f *testing.F) {
	f.Fuzz(func(t *testing.T, data []byte, a, b, c, d uint64, extraData []byte, txs uint16, txsData []byte) {
		if len(data) < 32+20+32+32+256+32+32+32 {
			return
		}
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
		payload.BlockNumber = Uint64Quantity(a)
		payload.GasLimit = Uint64Quantity(a)
		payload.GasUsed = Uint64Quantity(a)
		payload.Timestamp = Uint64Quantity(a)
		if len(extraData) > 32 {
			extraData = extraData[:32]
		}
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
		var roundTripped ExecutionPayload
		err := roundTripped.UnmarshalSSZ(uint32(len(buf.Bytes())), bytes.NewReader(buf.Bytes()))
		if err != nil {
			t.Fatalf("failed to decode previously marshalled payload: %v", err)
		}
		if diff := cmp.Diff(payload, roundTripped); diff != "" {
			t.Fatalf("The data did not round trip correctly:\n%s", diff)
		}
	})
}
