package interop

import (
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
			InvalidHeads: map[eth.ChainID]eth.BlockID{},
		}
		require.True(t, r.IsValid())
	})

	t.Run("returns false when InvalidHeads has entries", func(t *testing.T) {
		r := Result{
			Timestamp:   100,
			L1Inclusion: eth.BlockID{Number: 1},
			L2Heads:     map[eth.ChainID]eth.BlockID{eth.ChainIDFromUInt64(10): {Number: 100}},
			InvalidHeads: map[eth.ChainID]eth.BlockID{
				eth.ChainIDFromUInt64(10): {Number: 100, Hash: common.HexToHash("0xbad")},
			},
		}
		require.False(t, r.IsValid())
	})

	t.Run("returns false with multiple invalid heads", func(t *testing.T) {
		r := Result{
			Timestamp: 100,
			InvalidHeads: map[eth.ChainID]eth.BlockID{
				eth.ChainIDFromUInt64(10):   {Number: 100},
				eth.ChainIDFromUInt64(8453): {Number: 200},
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
			InvalidHeads: map[eth.ChainID]eth.BlockID{
				chainID1: {Hash: common.HexToHash("0xbad"), Number: 199},
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
			InvalidHeads: map[eth.ChainID]eth.BlockID{
				chainID: {Number: 199},
			},
		}

		_ = r.ToVerifiedResult()

		// Original should still have InvalidHeads
		require.Len(t, r.InvalidHeads, 1)
	})
}
