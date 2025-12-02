package derive

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCompressionAlgo(t *testing.T) {
	testCases := []struct {
		name                       string
		algo                       CompressionAlgo
		isValidCompressionAlgoType bool
		isBrotli                   bool
		brotliLevel                int
	}{
		{
			name:                       "zlib",
			algo:                       Zlib,
			isValidCompressionAlgoType: true,
			isBrotli:                   false,
		},
		{
			name:                       "brotli",
			algo:                       Brotli,
			isValidCompressionAlgoType: true,
			isBrotli:                   true,
			brotliLevel:                10,
		},
		{
			name:                       "brotli-9",
			algo:                       Brotli9,
			isValidCompressionAlgoType: true,
			isBrotli:                   true,
			brotliLevel:                9,
		},
		{
			name:                       "brotli-10",
			algo:                       Brotli10,
			isValidCompressionAlgoType: true,
			isBrotli:                   true,
			brotliLevel:                10,
		},
		{
			name:                       "brotli-11",
			algo:                       Brotli11,
			isValidCompressionAlgoType: true,
			isBrotli:                   true,
			brotliLevel:                11,
		},
		{
			name:                       "invalid",
			algo:                       CompressionAlgo("invalid"),
			isValidCompressionAlgoType: false,
			isBrotli:                   false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.isBrotli, tc.algo.IsBrotli())
			if tc.isBrotli {
				require.NotPanics(t, func() {
					blvl := GetBrotliLevel((tc.algo))
					require.Equal(t, tc.brotliLevel, blvl)
				})
			} else {
				require.Panics(t, func() { GetBrotliLevel(tc.algo) })
			}
			require.Equal(t, tc.isValidCompressionAlgoType, ValidCompressionAlgo(tc.algo))
		})
	}
}
