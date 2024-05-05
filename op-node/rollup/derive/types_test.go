package derive

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsBrotli(t *testing.T) {
	testCases := []struct {
		name                       string
		algo                       CompressionAlgo
		expectedResult             bool
		isValidCompressionAlgoType bool
	}{{
		name:                       "zlib",
		algo:                       Zlib,
		expectedResult:             false,
		isValidCompressionAlgoType: true,
	},
		{
			name:                       "brotli-9",
			algo:                       Brotli9,
			expectedResult:             true,
			isValidCompressionAlgoType: true,
		},
		{
			name:                       "brotli-10",
			algo:                       Brotli10,
			expectedResult:             true,
			isValidCompressionAlgoType: true,
		},
		{
			name:                       "brotli-11",
			algo:                       Brotli11,
			expectedResult:             true,
			isValidCompressionAlgoType: true,
		},
		{
			name:                       "invalid",
			algo:                       CompressionAlgo("invalid"),
			expectedResult:             false,
			isValidCompressionAlgoType: false,
		}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.expectedResult, tc.algo.IsBrotli())
			require.Equal(t, tc.isValidCompressionAlgoType, ValidCompressionAlgoType(tc.algo))
		})
	}
}
