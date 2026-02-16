package derive

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFrameSize(t *testing.T) {
	t.Run("empty frame data", func(t *testing.T) {
		f := Frame{Data: []byte{}}
		assert.Equal(t, uint64(frameOverhead), frameSize(f))
	})

	t.Run("non-empty frame data", func(t *testing.T) {
		data := make([]byte, 1000)
		f := Frame{Data: data}
		assert.Equal(t, uint64(1000+frameOverhead), frameSize(f))
	})
}

func TestChannelIDString(t *testing.T) {
	var id ChannelID
	// All zeros
	assert.Equal(t, "00000000000000000000000000000000", id.String())
}

func TestChannelIDTerminalString(t *testing.T) {
	var id ChannelID
	for i := range id {
		id[i] = byte(i + 1)
	}
	s := id.TerminalString()
	assert.Contains(t, s, "..")
}

func TestChannelIDMarshalText(t *testing.T) {
	var id ChannelID
	id[0] = 0xab
	id[15] = 0xcd
	text, err := id.MarshalText()
	require.NoError(t, err)
	assert.Equal(t, "ab0000000000000000000000000000cd", string(text))
}

func TestChannelIDUnmarshalText(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		var id ChannelID
		err := id.UnmarshalText([]byte("ab0000000000000000000000000000cd"))
		require.NoError(t, err)
		assert.Equal(t, byte(0xab), id[0])
		assert.Equal(t, byte(0xcd), id[15])
	})

	t.Run("invalid hex", func(t *testing.T) {
		var id ChannelID
		err := id.UnmarshalText([]byte("zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz"))
		require.Error(t, err)
	})

	t.Run("wrong length", func(t *testing.T) {
		var id ChannelID
		err := id.UnmarshalText([]byte("abcd"))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid length")
	})
}

func TestChannelIDRoundTrip(t *testing.T) {
	var original ChannelID
	for i := range original {
		original[i] = byte(i * 17)
	}
	text, err := original.MarshalText()
	require.NoError(t, err)

	var decoded ChannelID
	err = decoded.UnmarshalText(text)
	require.NoError(t, err)
	assert.Equal(t, original, decoded)
}

func TestDuplicateErr(t *testing.T) {
	assert.EqualError(t, DuplicateErr, "duplicate frame")
}

func TestMaxSpanBatchElementCount(t *testing.T) {
	assert.Equal(t, 10_000_000, MaxSpanBatchElementCount)
}
