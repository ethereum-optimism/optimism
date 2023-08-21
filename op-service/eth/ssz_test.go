package eth

import (
	"bytes"
	"encoding/binary"
	"math"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
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

func FuzzOBP01(f *testing.F) {
	payload := &ExecutionPayload{
		ExtraData: make([]byte, 32),
	}
	var buf bytes.Buffer
	_, err := payload.MarshalSSZ(&buf)
	require.NoError(f, err)
	data := buf.Bytes()

	f.Fuzz(func(t *testing.T, edOffset uint32, txOffset uint32) {
		clone := make([]byte, len(data))
		copy(clone, data)

		binary.LittleEndian.PutUint32(clone[436:440], edOffset)
		binary.LittleEndian.PutUint32(clone[504:508], txOffset)

		var unmarshalled ExecutionPayload
		err = unmarshalled.UnmarshalSSZ(uint32(len(clone)), bytes.NewReader(clone))
		if err == nil {
			t.Fatalf("expected a failure, but didn't get one")
		}
	})
}

// TestOPB01 verifies that the SSZ unmarshaling code
// properly checks for the transactionOffset being larger
// than the extraDataOffset.
func TestOPB01(t *testing.T) {
	payload := &ExecutionPayload{
		ExtraData: make([]byte, 32),
	}
	var buf bytes.Buffer
	_, err := payload.MarshalSSZ(&buf)
	require.NoError(t, err)
	data := buf.Bytes()

	// transactions offset is set between indices 504 and 508
	copy(data[504:508], make([]byte, 4))

	var unmarshalled ExecutionPayload
	err = unmarshalled.UnmarshalSSZ(uint32(len(data)), bytes.NewReader(data))
	require.Equal(t, ErrBadTransactionOffset, err)
}

// TestOPB04 verifies that the SSZ marshaling code
// properly returns an error when the ExtraData field
// cannot be represented in the outputted SSZ.
func TestOPB04(t *testing.T) {
	// First, test the maximum len - which in this case is the max uint32
	// minus the execution payload fixed part.
	payload := &ExecutionPayload{
		ExtraData: make([]byte, math.MaxUint32-executionPayloadFixedPart),
	}
	var buf bytes.Buffer
	_, err := payload.MarshalSSZ(&buf)
	require.NoError(t, err)
	buf.Reset()

	payload = &ExecutionPayload{
		ExtraData: make([]byte, math.MaxUint32-executionPayloadFixedPart+1),
	}
	_, err = payload.MarshalSSZ(&buf)
	require.Error(t, err)
	require.Equal(t, ErrExtraDataTooLarge, err)
}
