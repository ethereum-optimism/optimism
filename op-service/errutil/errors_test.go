package errutil

import (
	"errors"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/stretchr/testify/require"
)

func TestTryAddRevertReason(t *testing.T) {
	t.Run("AddsReason", func(t *testing.T) {
		err := stubError{}
		result := TryAddRevertReason(err)
		require.Contains(t, result.Error(), "kaboom")
	})

	t.Run("ReturnOriginalWhenNoErrorDataMethod", func(t *testing.T) {
		err := errors.New("boom")
		result := TryAddRevertReason(err)
		require.Same(t, err, result)
	})
}

func TestDecodeRevertReason(t *testing.T) {
	t.Run("NonStringData", func(t *testing.T) {
		result := DecodeRevertReason(42)
		require.Equal(t, "42", result)
	})

	t.Run("EmptyString", func(t *testing.T) {
		result := DecodeRevertReason("")
		require.Equal(t, "", result)
	})

	t.Run("InvalidHex", func(t *testing.T) {
		result := DecodeRevertReason("not-hex")
		require.Equal(t, "not-hex", result)
	})

	t.Run("TooShort", func(t *testing.T) {
		result := DecodeRevertReason("0x1234")
		require.Equal(t, "0x1234", result)
	})

	t.Run("StandardErrorString", func(t *testing.T) {
		// Error("insufficient balance") encoded
		data := "0x08c379a0" +
			"0000000000000000000000000000000000000000000000000000000000000020" +
			"0000000000000000000000000000000000000000000000000000000000000014" +
			"696e73756666696369656e742062616c616e6365000000000000000000000000"
		result := DecodeRevertReason(data)
		require.Equal(t, `Error("insufficient balance")`, result)
	})

	t.Run("StandardPanic", func(t *testing.T) {
		// Panic(0x11) — arithmetic overflow
		data := "0x4e487b71" +
			"0000000000000000000000000000000000000000000000000000000000000011"
		result := DecodeRevertReason(data)
		require.Equal(t, "Panic(17: arithmetic overflow/underflow)", result)
	})

	t.Run("StandardPanicDivByZero", func(t *testing.T) {
		// Panic(0x12) — division by zero
		data := "0x4e487b71" +
			"0000000000000000000000000000000000000000000000000000000000000012"
		result := DecodeRevertReason(data)
		require.Equal(t, "Panic(18: division or modulo by zero)", result)
	})

	t.Run("UnknownSelector", func(t *testing.T) {
		// Some unknown 4-byte selector + data
		data := "0x031c6de40000000000000000000000000000000000000000000000000000000000000001"
		result := DecodeRevertReason(data)
		// Falls through to returning raw hex
		require.Equal(t, data, result)
	})

	t.Run("NoPrefix", func(t *testing.T) {
		// Error("hi") without 0x prefix
		data := "08c379a0" +
			"0000000000000000000000000000000000000000000000000000000000000020" +
			"0000000000000000000000000000000000000000000000000000000000000002" +
			"6869000000000000000000000000000000000000000000000000000000000000"
		result := DecodeRevertReason(data)
		require.Equal(t, `Error("hi")`, result)
	})
}

func TestDecodeRevertReasonWithABIs(t *testing.T) {
	t.Run("FallsBackToStandard", func(t *testing.T) {
		// Error("test") — should decode even with no ABIs provided
		data := "0x08c379a0" +
			"0000000000000000000000000000000000000000000000000000000000000020" +
			"0000000000000000000000000000000000000000000000000000000000000004" +
			"7465737400000000000000000000000000000000000000000000000000000000"
		result := DecodeRevertReasonWithABIs(data)
		require.Equal(t, `Error("test")`, result)
	})

	t.Run("NilABIs", func(t *testing.T) {
		data := "0xdeadbeef"
		result := DecodeRevertReasonWithABIs(data, nil, nil)
		require.Equal(t, "0xdeadbeef", result)
	})

	t.Run("CustomABIError", func(t *testing.T) {
		// NoImplementation(uint32 gameType) — selector 0x031c6de4, gameType=1
		data := "0x031c6de4" +
			"0000000000000000000000000000000000000000000000000000000000000001"
		abiJSON := `[{"type":"error","name":"NoImplementation","inputs":[{"name":"gameType","type":"uint32"}]}]`
		parsed, err := abi.JSON(strings.NewReader(abiJSON))
		require.NoError(t, err)
		result := DecodeRevertReasonWithABIs(data, &parsed)
		require.Equal(t, "NoImplementation(gameType=1)", result)
	})
}

func TestExtractRevertData(t *testing.T) {
	t.Run("HasData", func(t *testing.T) {
		err := stubError{}
		data, ok := ExtractRevertData(err)
		require.True(t, ok)
		require.Equal(t, "kaboom", data)
	})

	t.Run("NoData", func(t *testing.T) {
		err := errors.New("plain error")
		_, ok := ExtractRevertData(err)
		require.False(t, ok)
	})
}

type stubError struct{}

func (s stubError) Error() string {
	return "where's the"
}

func (s stubError) ErrorData() interface{} {
	return "kaboom"
}
