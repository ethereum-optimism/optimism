package derive

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCompressionAlgo(t *testing.T) {
	testCases := []struct {
		name                       string
		algo                       CompressionAlgo
		isBrotli                   bool
		isValidCompressionAlgoType bool
	}{
		{
			name:                       "zlib",
			algo:                       Zlib,
			isBrotli:                   false,
			isValidCompressionAlgoType: true,
		},
		{
			name:                       "brotli-9",
			algo:                       Brotli9,
			isBrotli:                   true,
			isValidCompressionAlgoType: true,
		},
		{
			name:                       "brotli-10",
			algo:                       Brotli10,
			isBrotli:                   true,
			isValidCompressionAlgoType: true,
		},
		{
			name:                       "brotli-11",
			algo:                       Brotli11,
			isBrotli:                   true,
			isValidCompressionAlgoType: true,
		},
		{
			name:                       "invalid",
			algo:                       CompressionAlgo("invalid"),
			isBrotli:                   false,
			isValidCompressionAlgoType: false,
		}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.isBrotli, tc.algo.IsBrotli())
			if tc.isBrotli {
				require.NotPanics(t, func() { GetBrotliLevel((tc.algo)) })
			} else {
				require.Panics(t, func() { GetBrotliLevel(tc.algo) })
			}
			require.Equal(t, tc.isValidCompressionAlgoType, ValidCompressionAlgoType(tc.algo))
		})
	}
}
