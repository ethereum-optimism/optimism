package types

import (
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func clockFromParts(duration, timestamp uint64) *Clock {
	bigDuration := new(big.Int).SetUint64(duration)
	encoded := new(big.Int).Lsh(bigDuration, 64)
	raw := new(big.Int).Or(encoded, new(big.Int).SetUint64(timestamp))
	return NewClock(raw)
}

func TestClaim_RemainingDuration(t *testing.T) {
	tests := []struct {
		name      string
		duration  uint64
		timestamp uint64
		now       int64
		expected  uint64
	}{
		{
			name:      "AllZeros",
			duration:  0,
			timestamp: 0,
			now:       0,
			expected:  0,
		},
		{
			name:      "ZeroTimestamp",
			duration:  5,
			timestamp: 0,
			now:       0,
			expected:  5,
		},
		{
			name:      "ZeroTimestampWithNow",
			duration:  5,
			timestamp: 0,
			now:       10,
			expected:  15,
		},
		{
			name:      "ZeroNow",
			duration:  5,
			timestamp: 10,
			now:       0,
			expected:  5,
		},
		{
			name:      "ValidTimeSinze",
			duration:  20,
			timestamp: 10,
			now:       15,
			expected:  25,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			claim := &Claim{
				Clock: clockFromParts(test.duration, test.timestamp),
			}
			require.Equal(t, time.Duration(test.expected), claim.ChessTime(time.Unix(test.now, 0)))
		})
	}
}

func TestClock(t *testing.T) {
	t.Run("DurationAndTimestamp", func(t *testing.T) {
		by := common.Hex2Bytes("00000000000000050000000000000002")
		encoded := new(big.Int).SetBytes(by)
		clock := NewClock(encoded)
		require.Equal(t, uint64(5), clock.Duration)
		require.Equal(t, uint64(2), clock.Timestamp)
		require.Equal(t, encoded, clock.Packed())
	})

	t.Run("ZeroDuration", func(t *testing.T) {
		by := common.Hex2Bytes("00000000000000000000000000000002")
		encoded := new(big.Int).SetBytes(by)
		clock := NewClock(encoded)
		require.Equal(t, uint64(0), clock.Duration)
		require.Equal(t, uint64(2), clock.Timestamp)
		require.Equal(t, encoded, clock.Packed())
	})

	t.Run("ZeroTimestamp", func(t *testing.T) {
		by := common.Hex2Bytes("00000000000000050000000000000000")
		encoded := new(big.Int).SetBytes(by)
		clock := NewClock(encoded)
		require.Equal(t, uint64(5), clock.Duration)
		require.Equal(t, uint64(0), clock.Timestamp)
		require.Equal(t, encoded, clock.Packed())
	})

	t.Run("ZeroClock", func(t *testing.T) {
		by := common.Hex2Bytes("00000000000000000000000000000000")
		encoded := new(big.Int).SetBytes(by)
		clock := NewClock(encoded)
		require.Equal(t, uint64(0), clock.Duration)
		require.Equal(t, uint64(0), clock.Timestamp)
		require.Equal(t, encoded.Uint64(), clock.Packed().Uint64())
	})
}

func TestNewPreimageOracleData(t *testing.T) {
	t.Run("LocalData", func(t *testing.T) {
		data := NewPreimageOracleData([]byte{1, 2, 3}, []byte{4, 5, 6}, 7)
		require.True(t, data.IsLocal)
		require.Equal(t, []byte{1, 2, 3}, data.OracleKey)
		require.Equal(t, []byte{4, 5, 6}, data.GetPreimageWithSize())
		require.Equal(t, uint32(7), data.OracleOffset)
	})

	t.Run("GlobalData", func(t *testing.T) {
		data := NewPreimageOracleData([]byte{0, 2, 3}, []byte{4, 5, 6}, 7)
		require.False(t, data.IsLocal)
		require.Equal(t, []byte{0, 2, 3}, data.OracleKey)
		require.Equal(t, []byte{4, 5, 6}, data.GetPreimageWithSize())
		require.Equal(t, uint32(7), data.OracleOffset)
	})
}

func TestIsRootPosition(t *testing.T) {
	tests := []struct {
		name     string
		position Position
		expected bool
	}{
		{
			name:     "ZeroRoot",
			position: NewPositionFromGIndex(big.NewInt(0)),
			expected: true,
		},
		{
			name:     "ValidRoot",
			position: NewPositionFromGIndex(big.NewInt(1)),
			expected: true,
		},
		{
			name:     "NotRoot",
			position: NewPositionFromGIndex(big.NewInt(2)),
			expected: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.expected, test.position.IsRootPosition())
		})
	}
}
