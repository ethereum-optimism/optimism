package interop

import (
	"encoding/json"
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestResult_IsValid(t *testing.T) {
	t.Parallel()

	t.Run("returns true when InvalidHeads is nil", func(t *testing.T) {
		r := Result{
			Timestamp:    100,
			L1Inclusion:  eth.BlockID{Number: 1},
			L2Heads:      map[eth.ChainID]eth.BlockID{eth.ChainIDFromUInt64(10): {Number: 100}},
			InvalidHeads: nil,
		}
		require.True(t, r.IsValid())
	})

	t.Run("returns true when InvalidHeads is empty map", func(t *testing.T) {
		r := Result{
			Timestamp:    100,
			L1Inclusion:  eth.BlockID{Number: 1},
			L2Heads:      map[eth.ChainID]eth.BlockID{eth.ChainIDFromUInt64(10): {Number: 100}},
			InvalidHeads: map[eth.ChainID]InvalidHead{},
		}
		require.True(t, r.IsValid())
	})

	t.Run("returns false when InvalidHeads has entries", func(t *testing.T) {
		r := Result{
			Timestamp:   100,
			L1Inclusion: eth.BlockID{Number: 1},
			L2Heads:     map[eth.ChainID]eth.BlockID{eth.ChainIDFromUInt64(10): {Number: 100}},
			InvalidHeads: map[eth.ChainID]InvalidHead{
				eth.ChainIDFromUInt64(10): {BlockID: eth.BlockID{Number: 100, Hash: common.HexToHash("0xbad")}},
			},
		}
		require.False(t, r.IsValid())
	})

	t.Run("returns false with multiple invalid heads", func(t *testing.T) {
		r := Result{
			Timestamp: 100,
			InvalidHeads: map[eth.ChainID]InvalidHead{
				eth.ChainIDFromUInt64(10):   {BlockID: eth.BlockID{Number: 100}},
				eth.ChainIDFromUInt64(8453): {BlockID: eth.BlockID{Number: 200}},
			},
		}
		require.False(t, r.IsValid())
	})
}

func TestResult_ToVerifiedResult(t *testing.T) {
	t.Parallel()

	t.Run("copies all fields except InvalidHeads", func(t *testing.T) {
		chainID1 := eth.ChainIDFromUInt64(10)
		chainID2 := eth.ChainIDFromUInt64(8453)

		r := Result{
			Timestamp: 12345,
			L1Inclusion: eth.BlockID{
				Hash:   common.HexToHash("0x1111"),
				Number: 100,
			},
			L2Heads: map[eth.ChainID]eth.BlockID{
				chainID1: {Hash: common.HexToHash("0x2222"), Number: 200},
				chainID2: {Hash: common.HexToHash("0x3333"), Number: 300},
			},
			InvalidHeads: map[eth.ChainID]InvalidHead{
				chainID1: {BlockID: eth.BlockID{Hash: common.HexToHash("0xbad"), Number: 199}},
			},
		}

		verified := r.ToVerifiedResult()

		require.Equal(t, r.Timestamp, verified.Timestamp)
		require.Equal(t, r.L1Inclusion, verified.L1Inclusion)
		require.Equal(t, r.L2Heads, verified.L2Heads)
	})

	t.Run("handles nil L2Heads", func(t *testing.T) {
		r := Result{
			Timestamp:   100,
			L1Inclusion: eth.BlockID{Number: 1},
			L2Heads:     nil,
		}

		verified := r.ToVerifiedResult()

		require.Equal(t, r.Timestamp, verified.Timestamp)
		require.Nil(t, verified.L2Heads)
	})

	t.Run("handles empty L2Heads", func(t *testing.T) {
		r := Result{
			Timestamp:   100,
			L1Inclusion: eth.BlockID{Number: 1},
			L2Heads:     map[eth.ChainID]eth.BlockID{},
		}

		verified := r.ToVerifiedResult()

		require.Empty(t, verified.L2Heads)
	})

	t.Run("original Result unchanged after conversion", func(t *testing.T) {
		chainID := eth.ChainIDFromUInt64(10)
		r := Result{
			Timestamp:   100,
			L1Inclusion: eth.BlockID{Number: 1},
			L2Heads: map[eth.ChainID]eth.BlockID{
				chainID: {Number: 200},
			},
			InvalidHeads: map[eth.ChainID]InvalidHead{
				chainID: {BlockID: eth.BlockID{Number: 199}},
			},
		}

		_ = r.ToVerifiedResult()

		// Original should still have InvalidHeads
		require.Len(t, r.InvalidHeads, 1)
	})
}

func TestInvalidHead_JSONRoundTrip(t *testing.T) {
	t.Parallel()

	original := InvalidHead{
		BlockID: eth.BlockID{
			Hash:   common.HexToHash("0xdead"),
			Number: 500,
		},
		StateRoot:                eth.Bytes32(common.HexToHash("0xstate")),
		MessagePasserStorageRoot: eth.Bytes32(common.HexToHash("0xmsgpasser")),
	}

	data, err := json.Marshal(original)
	require.NoError(t, err)

	var decoded InvalidHead
	require.NoError(t, json.Unmarshal(data, &decoded))

	require.Equal(t, original.BlockID, decoded.BlockID)
	require.Equal(t, original.StateRoot, decoded.StateRoot)
	require.Equal(t, original.MessagePasserStorageRoot, decoded.MessagePasserStorageRoot)
}

func TestInvalidHead_JSONRoundTrip_ZeroRoots(t *testing.T) {
	t.Parallel()

	original := InvalidHead{
		BlockID: eth.BlockID{
			Hash:   common.HexToHash("0xbeef"),
			Number: 42,
		},
		StateRoot:                eth.Bytes32{},
		MessagePasserStorageRoot: eth.Bytes32{},
	}

	data, err := json.Marshal(original)
	require.NoError(t, err)

	var decoded InvalidHead
	require.NoError(t, json.Unmarshal(data, &decoded))

	require.Equal(t, original.BlockID, decoded.BlockID)
	require.Equal(t, original.StateRoot, decoded.StateRoot)
	require.Equal(t, original.MessagePasserStorageRoot, decoded.MessagePasserStorageRoot)
}

func TestPendingInvalidation_JSONRoundTrip(t *testing.T) {
	t.Parallel()

	original := PendingInvalidation{
		ChainID:                  eth.ChainIDFromUInt64(10),
		BlockID:                  eth.BlockID{Hash: common.HexToHash("0xbad"), Number: 100},
		Timestamp:                42,
		StateRoot:                eth.Bytes32(common.HexToHash("0xstate")),
		MessagePasserStorageRoot: eth.Bytes32(common.HexToHash("0xmsgpasser")),
	}

	data, err := json.Marshal(original)
	require.NoError(t, err)

	var decoded PendingInvalidation
	require.NoError(t, json.Unmarshal(data, &decoded))

	require.Equal(t, original.ChainID, decoded.ChainID)
	require.Equal(t, original.BlockID, decoded.BlockID)
	require.Equal(t, original.Timestamp, decoded.Timestamp)
	require.Equal(t, original.StateRoot, decoded.StateRoot)
	require.Equal(t, original.MessagePasserStorageRoot, decoded.MessagePasserStorageRoot)
}
