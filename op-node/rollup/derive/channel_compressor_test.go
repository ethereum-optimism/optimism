package derive

import (
	"math/rand"
	"testing"

	"github.com/ethereum-optimism/optimism/op-node/rollup/derive/compression"
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

func TestChannelCompressor_NewReset(t *testing.T) {
	testCases := []struct {
		name              string
		algo              compression.CompressionAlgo
		expectedResetSize int
		expectErr         bool
	}{
		{
			name:              "zlib",
			algo:              compression.Zlib,
			expectedResetSize: 0,
		},
		{
			name:              "brotli10",
			algo:              compression.Brotli10,
			expectedResetSize: 1,
		},
		{
			name:              "zstd",
			algo:              compression.CompressionAlgo("zstd"),
			expectedResetSize: 0,
			expectErr:         true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			scc, err := NewChannelCompressor(tc.algo)
			if tc.expectErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.expectedResetSize, scc.Len())

			_, err = scc.Write(randomBytes(10))
			require.NoError(t, err)
			err = scc.Flush()
			require.NoError(t, err)
			require.Greater(t, scc.Len(), tc.expectedResetSize)

			scc.Reset()
			require.Equal(t, tc.expectedResetSize, scc.Len())
		})
	}
}
