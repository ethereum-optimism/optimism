package derive

import (
	"bytes"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDecodeSpanBatchBits(t *testing.T) {
	t.Run("decode single byte aligned", func(t *testing.T) {
		// 8 bits: 0b10101010 = 0xAA
		r := bytes.NewReader([]byte{0xAA})
		bits, err := decodeSpanBatchBits(r, 8)
		require.NoError(t, err)
		assert.Equal(t, big.NewInt(0xAA), bits)
	})

	t.Run("decode non-byte-aligned", func(t *testing.T) {
		// 5 bits need 1 byte. 0b00010100 = 0x14, top 3 bits should be zero
		r := bytes.NewReader([]byte{0x14})
		bits, err := decodeSpanBatchBits(r, 5)
		require.NoError(t, err)
		assert.Equal(t, big.NewInt(0x14), bits)
	})

	t.Run("decode zero bits", func(t *testing.T) {
		r := bytes.NewReader([]byte{})
		bits, err := decodeSpanBatchBits(r, 0)
		require.NoError(t, err)
		assert.Equal(t, big.NewInt(0), bits)
	})

	t.Run("decode multi-byte", func(t *testing.T) {
		// 16 bits
		r := bytes.NewReader([]byte{0x01, 0x02})
		bits, err := decodeSpanBatchBits(r, 16)
		require.NoError(t, err)
		assert.Equal(t, big.NewInt(0x0102), bits)
	})

	t.Run("error on insufficient data", func(t *testing.T) {
		r := bytes.NewReader([]byte{0x01})
		_, err := decodeSpanBatchBits(r, 16)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read bits")
	})

	t.Run("error on trailing bits beyond bitLength", func(t *testing.T) {
		// 3 bits max, but byte has bit 7 set (value 128 > 3 bits)
		r := bytes.NewReader([]byte{0x80})
		_, err := decodeSpanBatchBits(r, 3)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "bitfield has")
	})
}

func TestEncodeSpanBatchBits(t *testing.T) {
	t.Run("encode single byte aligned", func(t *testing.T) {
		var buf bytes.Buffer
		err := encodeSpanBatchBits(&buf, 8, big.NewInt(0xAA))
		require.NoError(t, err)
		assert.Equal(t, []byte{0xAA}, buf.Bytes())
	})

	t.Run("encode non-byte-aligned with padding", func(t *testing.T) {
		var buf bytes.Buffer
		// 5 bits, value 0b10101 = 21
		err := encodeSpanBatchBits(&buf, 5, big.NewInt(21))
		require.NoError(t, err)
		// 1 byte, left-padded: 0b00010101 = 0x15
		assert.Equal(t, []byte{0x15}, buf.Bytes())
	})

	t.Run("encode zero", func(t *testing.T) {
		var buf bytes.Buffer
		err := encodeSpanBatchBits(&buf, 8, big.NewInt(0))
		require.NoError(t, err)
		assert.Equal(t, []byte{0x00}, buf.Bytes())
	})

	t.Run("encode zero bitLength", func(t *testing.T) {
		var buf bytes.Buffer
		err := encodeSpanBatchBits(&buf, 0, big.NewInt(0))
		require.NoError(t, err)
		assert.Empty(t, buf.Bytes())
	})

	t.Run("error when bits exceed bitLength", func(t *testing.T) {
		var buf bytes.Buffer
		// 3 bits max, but value needs 8 bits
		err := encodeSpanBatchBits(&buf, 3, big.NewInt(0xFF))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "bitfield is larger than bitLength")
	})
}

func TestSpanBatchBitsRoundTrip(t *testing.T) {
	testCases := []struct {
		name      string
		bitLength uint64
		value     *big.Int
	}{
		{"8 bits", 8, big.NewInt(0xFF)},
		{"16 bits", 16, big.NewInt(0xBEEF)},
		{"1 bit set", 1, big.NewInt(1)},
		{"1 bit zero", 1, big.NewInt(0)},
		{"13 bits", 13, big.NewInt(0x1FFF)},
		{"24 bits sparse", 24, big.NewInt(0x010001)},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := encodeSpanBatchBits(&buf, tc.bitLength, tc.value)
			require.NoError(t, err)

			r := bytes.NewReader(buf.Bytes())
			decoded, err := decodeSpanBatchBits(r, tc.bitLength)
			require.NoError(t, err)
			assert.Equal(t, 0, tc.value.Cmp(decoded), "expected %s, got %s", tc.value, decoded)
		})
	}
}
