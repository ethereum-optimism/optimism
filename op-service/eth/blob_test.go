package eth

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBlobEncodeDecode(t *testing.T) {
	cases := []string{
		"this is a test of blob encoding/decoding",
		"short",
		"\x00",
		"\x00\x01\x00",
		"\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00",
		"\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00",
		"\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00",
		"",
	}

	var b Blob
	for _, c := range cases {
		data := Data(c)
		err := b.FromData(data)
		require.NoError(t, err)
		decoded, err := b.ToData()
		require.NoError(t, err)
		require.Equal(t, c, string(decoded))
	}
}

func TestSmallBlobEncoding(t *testing.T) {
	// the first field element is filled and no data remains
	data := Data(make([]byte, 128))
	data[127] = 0xFF

	var b Blob
	err := b.FromData(data)
	require.NoError(t, err)

	decoded, err := b.ToData()
	require.NoError(t, err)
	require.Equal(t, data, decoded)

	// only 10 bytes of data
	data[9] = 0xFF
	err = b.FromData(data)
	require.NoError(t, err)
	decoded, err = b.ToData()
	require.NoError(t, err)
	require.Equal(t, data, decoded)

	// no 3 bytes of extra data left to encode after the first 4 field elements
	data = Data(make([]byte, 27+31*3))
	data[27+31*3-1] = 0xFF
	err = b.FromData(data)
	require.NoError(t, err)
	decoded, err = b.ToData()
	require.NoError(t, err)
	require.Equal(t, data, decoded)
}

func TestBigBlobEncoding(t *testing.T) {
	r := rand.New(rand.NewSource(99))
	bigData := Data(make([]byte, MaxBlobDataSize))
	for i := range bigData {
		bigData[i] = byte(r.Intn(256))
	}
	var b Blob
	// test the maximum size of data that can be encoded
	err := b.FromData(bigData)
	require.NoError(t, err)
	decoded, err := b.ToData()
	require.NoError(t, err)
	require.Equal(t, bigData, decoded)

	// perform encode/decode test on progressively smaller inputs to exercise boundary conditions
	// pertaining to length of the input data
	for i := 1; i < 256; i++ {
		tempBigData := bigData[i:]
		err := b.FromData(tempBigData)
		require.NoError(t, err)
		decoded, err := b.ToData()
		require.NoError(t, err)
		require.Equal(t, len(tempBigData), len(decoded))
		require.Equal(t, tempBigData, decoded)
	}
}

func TestInvalidBlobDecoding(t *testing.T) {
	data := Data("this is a test of invalid blob decoding")
	var b Blob
	if err := b.FromData(data); err != nil {
		t.Fatalf("failed to encode bytes: %v", err)
	}

	b[32] = 0x80 // field elements should never have their highest order bit set
	_, err := b.ToData()
	require.ErrorIs(t, err, ErrBlobInvalidFieldElement)
	b[32] = 0x0

	b[VersionOffset] = 0x01 // invalid encoding version
	_, err = b.ToData()
	require.ErrorIs(t, err, ErrBlobInvalidEncodingVersion)
	b[VersionOffset] = EncodingVersion

	b[2] = 0xFF // encode an invalid (much too long) length prefix
	_, err = b.ToData()
	require.ErrorIs(t, err, ErrBlobInvalidLength)
}

func TestTooLongDataEncoding(t *testing.T) {
	// should never be able to encode data that has size the same as that of the blob due to < 256
	// bit precision of each field element
	data := Data(make([]byte, BlobSize))
	var b Blob
	err := b.FromData(data)
	require.ErrorIs(t, err, ErrBlobInputTooLarge)
}

func FuzzEncodeDecodeBlob(f *testing.F) {
	var b Blob
	f.Fuzz(func(t *testing.T, d []byte) {
		b.Clear()
		data := Data(d)
		err := b.FromData(data)
		require.NoError(t, err)
		decoded, err := b.ToData()
		require.NoError(t, err)
		require.Equal(t, data, decoded)
	})
}

// TODO(optimism#8872): Create test vectors to implement one-way tests confirming that specific inputs yield
// desired outputs.
