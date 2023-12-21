package eth

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// FuzzExecutionPayloadUnmarshal checks that our SSZ decoding never panics
func FuzzExecutionPayloadUnmarshal(f *testing.F) {
	f.Fuzz(func(t *testing.T, data []byte) {
		{
			var payload ExecutionPayload
			err := payload.UnmarshalSSZ(BlockV1, uint32(len(data)), bytes.NewReader(data))
			if err != nil {
				// not every input is a valid ExecutionPayload, that's ok. Should just not panic.
				return
			}
		}

		{
			var payload ExecutionPayload
			err := payload.UnmarshalSSZ(BlockV2, uint32(len(data)), bytes.NewReader(data))
			if err != nil {
				// not every input is a valid ExecutionPayload, that's ok. Should just not panic.
				return
			}
		}
	})
}

// FuzzExecutionPayloadMarshalUnmarshal checks that our SSZ encoding>decoding round trips properly
func FuzzExecutionPayloadMarshalUnmarshalV1(f *testing.F) {
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
		err := roundTripped.UnmarshalSSZ(BlockV1, uint32(len(buf.Bytes())), bytes.NewReader(buf.Bytes()))
		if err != nil {
			t.Fatalf("failed to decode previously marshalled payload: %v", err)
		}
		if diff := cmp.Diff(payload, roundTripped); diff != "" {
			t.Fatalf("The data did not round trip correctly:\n%s", diff)
		}
	})
}

func FuzzExecutionPayloadMarshalUnmarshalV2(f *testing.F) {
	f.Fuzz(func(t *testing.T, data []byte, a, b, c, d uint64, extraData []byte, txs uint16, txsData []byte, wCount uint16) {
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

		wCount = wCount % maxWithdrawalsPerPayload
		withdrawals := make(types.Withdrawals, wCount)
		for i := 0; i < int(wCount); i++ {
			withdrawals[i] = &types.Withdrawal{
				Index:     a,
				Validator: b,
				Address:   common.BytesToAddress(data[:20]),
				Amount:    c,
			}
		}
		payload.Withdrawals = &withdrawals

		var buf bytes.Buffer
		if _, err := payload.MarshalSSZ(&buf); err != nil {
			t.Fatalf("failed to marshal ExecutionPayload: %v", err)
		}
		var roundTripped ExecutionPayload
		err := roundTripped.UnmarshalSSZ(BlockV2, uint32(len(buf.Bytes())), bytes.NewReader(buf.Bytes()))
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
		err = unmarshalled.UnmarshalSSZ(BlockV1, uint32(len(clone)), bytes.NewReader(clone))
		if err == nil {
			t.Fatalf("expected a failure, but didn't get one")
		}
	})
}

// TestOPB01 verifies that the SSZ unmarshalling code
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
	err = unmarshalled.UnmarshalSSZ(BlockV1, uint32(len(data)), bytes.NewReader(data))
	require.Equal(t, ErrBadTransactionOffset, err)
}

// TestOPB04 verifies that the SSZ marshaling code
// properly returns an error when the ExtraData field
// cannot be represented in the outputted SSZ.
func TestOPB04(t *testing.T) {
	data := make([]byte, math.MaxUint32)

	var buf bytes.Buffer
	// First, test the maximum len - which in this case is the max uint32
	// minus the execution payload fixed part.
	payload := &ExecutionPayload{
		ExtraData:   data[:math.MaxUint32-executionPayloadFixedPart(BlockV1)],
		Withdrawals: nil,
	}

	_, err := payload.MarshalSSZ(&buf)
	require.NoError(t, err)
	buf.Reset()

	tests := []struct {
		version     BlockVersion
		withdrawals *types.Withdrawals
	}{
		{BlockV1, nil},
		{BlockV2, &types.Withdrawals{}},
	}

	for _, test := range tests {
		payload := &ExecutionPayload{
			ExtraData:   data[:math.MaxUint32-executionPayloadFixedPart(test.version)+1],
			Withdrawals: test.withdrawals,
		}
		_, err := payload.MarshalSSZ(&buf)
		require.Error(t, err)
		require.Equal(t, ErrExtraDataTooLarge, err)
	}

}

func createPayloadWithWithdrawals(w *types.Withdrawals) *ExecutionPayload {
	return &ExecutionPayload{
		ParentHash:    common.HexToHash("0x123"),
		FeeRecipient:  common.HexToAddress("0x456"),
		StateRoot:     Bytes32(common.HexToHash("0x789")),
		ReceiptsRoot:  Bytes32(common.HexToHash("0xabc")),
		LogsBloom:     Bytes256{byte(13), byte(14), byte(15)},
		PrevRandao:    Bytes32(common.HexToHash("0x111")),
		BlockNumber:   Uint64Quantity(222),
		GasLimit:      Uint64Quantity(333),
		GasUsed:       Uint64Quantity(444),
		Timestamp:     Uint64Quantity(555),
		ExtraData:     common.Hex2Bytes("6666"),
		BaseFeePerGas: *uint256.NewInt(777),
		BlockHash:     common.HexToHash("0x888"),
		Withdrawals:   w,
		Transactions:  []Data{common.Hex2Bytes("9999")},
	}
}

func TestMarshalUnmarshalWithdrawals(t *testing.T) {
	emptyWithdrawal := &types.Withdrawals{}
	withdrawals := &types.Withdrawals{
		{
			Index:     987,
			Validator: 654,
			Address:   common.HexToAddress("0x898"),
			Amount:    321,
		},
	}
	maxWithdrawals := make(types.Withdrawals, maxWithdrawalsPerPayload)
	for i := 0; i < maxWithdrawalsPerPayload; i++ {
		maxWithdrawals[i] = &types.Withdrawal{
			Index:     987,
			Validator: 654,
			Address:   common.HexToAddress("0x898"),
			Amount:    321,
		}
	}
	tooManyWithdrawals := make(types.Withdrawals, maxWithdrawalsPerPayload+1)
	for i := 0; i < maxWithdrawalsPerPayload+1; i++ {
		tooManyWithdrawals[i] = &types.Withdrawal{
			Index:     987,
			Validator: 654,
			Address:   common.HexToAddress("0x898"),
			Amount:    321,
		}
	}

	tests := []struct {
		name        string
		version     BlockVersion
		hasError    bool
		withdrawals *types.Withdrawals
	}{
		{"ZeroWithdrawalsSucceeds", BlockV2, false, emptyWithdrawal},
		{"ZeroWithdrawalsFailsToDeserialize", BlockV1, true, emptyWithdrawal},
		{"WithdrawalsSucceeds", BlockV2, false, withdrawals},
		{"WithdrawalsFailsToDeserialize", BlockV1, true, withdrawals},
		{"MaxWithdrawalsSucceeds", BlockV2, false, &maxWithdrawals},
		{"TooManyWithdrawalsErrors", BlockV2, true, &tooManyWithdrawals},
	}

	for _, test := range tests {
		test := test

		t.Run(fmt.Sprintf("TestWithdrawalUnmarshalMarshal_%s", test.name), func(t *testing.T) {
			input := createPayloadWithWithdrawals(test.withdrawals)

			var buf bytes.Buffer
			_, err := input.MarshalSSZ(&buf)
			require.NoError(t, err)
			data := buf.Bytes()

			output := &ExecutionPayload{}
			err = output.UnmarshalSSZ(test.version, uint32(len(data)), bytes.NewReader(data))

			if test.hasError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, input, output)
				if test.withdrawals != nil {
					require.Equal(t, len(*test.withdrawals), len(*output.Withdrawals))
				}
			}
		})
	}
}
