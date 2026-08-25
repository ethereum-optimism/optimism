package types

import (
	"testing"

	"github.com/ethereum/go-ethereum/rlp"
	"github.com/stretchr/testify/require"
)

// Keep decoding behavior aligned with op-alloy.
func TestDecodePostExecPayload(t *testing.T) {
	valid := PostExecPayload{
		Version:               PostExecPayloadVersion,
		BlockNumber:           42,
		SelectedBaseFeePerGas: 123,
		GasRefundEntries: []SDMGasEntry{
			{Index: 1, GasRefund: 2500},
			{Index: 3, GasRefund: 2000},
		},
	}
	validBytes, err := rlp.EncodeToBytes(&valid)
	require.NoError(t, err)

	t.Run("round trips a well-formed payload", func(t *testing.T) {
		got, err := DecodePostExecPayload(validBytes)
		require.NoError(t, err)
		require.Equal(t, valid, *got)
	})

	t.Run("accepts an empty entry set", func(t *testing.T) {
		raw, err := rlp.EncodeToBytes(&PostExecPayload{
			Version:               PostExecPayloadVersion,
			BlockNumber:           1,
			SelectedBaseFeePerGas: 123,
			GasRefundEntries:      []SDMGasEntry{},
		})
		require.NoError(t, err)
		got, err := DecodePostExecPayload(raw)
		require.NoError(t, err)
		require.Empty(t, got.GasRefundEntries)
		require.Equal(t, uint64(1), got.BlockNumber)
		require.Equal(t, uint64(123), got.SelectedBaseFeePerGas)
	})

	t.Run("rejects empty input", func(t *testing.T) {
		_, err := DecodePostExecPayload(nil)
		require.ErrorContains(t, err, "empty post-exec payload")
	})

	t.Run("rejects an unsupported version", func(t *testing.T) {
		raw, err := rlp.EncodeToBytes(&PostExecPayload{Version: PostExecPayloadVersion + 1})
		require.NoError(t, err)
		_, err = DecodePostExecPayload(raw)
		require.ErrorContains(t, err, "unsupported post-exec payload version")
	})

	t.Run("rejects trailing bytes after the payload", func(t *testing.T) {
		_, err := DecodePostExecPayload(append(append([]byte{}, validBytes...), 0x00))
		require.ErrorContains(t, err, "decode post-exec payload")
	})

	t.Run("rejects a list with too few fields", func(t *testing.T) {
		// [1, []] is an invalid two-field shape.
		_, err := DecodePostExecPayload([]byte{0xc2, 0x01, 0xc0})
		require.ErrorContains(t, err, "decode post-exec payload")
	})

	t.Run("rejects a non-list payload", func(t *testing.T) {
		_, err := DecodePostExecPayload([]byte{0x01})
		require.ErrorContains(t, err, "decode post-exec payload")
	})
}
