package derive

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/require"
)

var r = rand.New(rand.NewSource(99))

func randomBytes(length int) []byte {
	b := make([]byte, length)
	_, err := r.Read(b)
	// Rand.Read always returns nil error
	if err != nil {
		panic(err)
	}
	return b
}

func TestSpanChannelCompressor(t *testing.T) {
	testCases := []struct {
		name                   string
		algo                   CompressionAlgo
		expectedCompressedSize int
	}{{
		name:                   "zlib",
		algo:                   Zlib,
		expectedCompressedSize: 0,
	},
		{
			name:                   "brotli10",
			algo:                   Brotli10,
			expectedCompressedSize: 1,
		},
		{
			name:                   "zstd",
			algo:                   CompressionAlgo("zstd"),
			expectedCompressedSize: 0,
		}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			scc, err := NewChannelCompressor(tc.algo)
			if tc.name == "zstd" {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.expectedCompressedSize, scc.Len())

			scc.Write(randomBytes(10000000))
			require.Greater(t, scc.Len(), tc.expectedCompressedSize)

			scc.Reset()
			require.Equal(t, tc.expectedCompressedSize, scc.Len())
		})
	}
}
