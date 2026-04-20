package sdmreplay

import (
	"strconv"
	"testing"

	"github.com/ethereum/go-ethereum/rlp"
	"github.com/stretchr/testify/require"
)

func TestDecodePayload_AcceptsCurrentShape(t *testing.T) {
	payload := PostExecPayload{
		Version:     PostExecPayloadVersion,
		BlockNumber: 42,
		GasRefundEntries: []SDMGasEntry{
			{Index: 3, GasRefund: 7},
			{Index: 5, GasRefund: 11},
		},
	}
	encoded, err := rlp.EncodeToBytes(&payload)
	require.NoError(t, err)

	decoded, err := DecodePayload(encoded)
	require.NoError(t, err)
	require.Equal(t, &payload, decoded)
}

func TestDecodePayload_RejectsUnknownVersion(t *testing.T) {
	// Any non-1 version must be rejected to stay in lock-step with the Rust decoder in
	// rust/op-alloy, which gates on POST_EXEC_PAYLOAD_VERSION. Cross-language divergence
	// here would let a Go-based replay pipeline accept payloads the Rust node rejects.
	for _, version := range []uint64{0, 2, 99} {
		t.Run("version_"+strconv.FormatUint(version, 10), func(t *testing.T) {
			payload := PostExecPayload{
				Version:          version,
				BlockNumber:      1,
				GasRefundEntries: []SDMGasEntry{{Index: 0, GasRefund: 1}},
			}
			encoded, err := rlp.EncodeToBytes(&payload)
			require.NoError(t, err)

			_, err = DecodePayload(encoded)
			require.Error(t, err)
			require.Contains(t, err.Error(), "unsupported post-exec payload version")
		})
	}
}

func TestDecodePayload_RejectsUnknownVersionOnLegacyShape(t *testing.T) {
	// Same version check on the legacy two-field shape — the fallback decoder must not
	// become an escape hatch for payloads whose version is wrong.
	legacy := struct {
		Version          uint64
		GasRefundEntries []SDMGasEntry
	}{
		Version:          7,
		GasRefundEntries: []SDMGasEntry{{Index: 0, GasRefund: 1}},
	}
	encoded, err := rlp.EncodeToBytes(&legacy)
	require.NoError(t, err)

	_, err = DecodePayload(encoded)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unsupported post-exec payload version")
}

func TestDecodePayload_EmptyInputRejected(t *testing.T) {
	_, err := DecodePayload(nil)
	require.Error(t, err)

	_, err = DecodePayload([]byte{})
	require.Error(t, err)
}
