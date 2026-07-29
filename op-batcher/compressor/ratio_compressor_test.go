package compressor_test

import (
	"fmt"
	"math"
	"testing"

	"github.com/ethereum-optimism/optimism/op-batcher/compressor"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/stretchr/testify/require"
)

func TestChannelConfig_InputThreshold(t *testing.T) {
	tests := []struct {
		targetOutputSize  uint64
		approxComprRatio  float64
		expInputThreshold uint64
		assertion         func(uint64) // optional, for more complex assertion
	}{
		{
			targetOutputSize:  1,
			approxComprRatio:  0.4,
			expInputThreshold: 2,
		},
		{
			targetOutputSize:  1,
			approxComprRatio:  1,
			expInputThreshold: 1,
		},
		{
			targetOutputSize:  100_000,
			approxComprRatio:  0.4,
			expInputThreshold: 250_000,
		},
		{
			targetOutputSize:  1,
			approxComprRatio:  0.4,
			expInputThreshold: 2,
		},
		{
			targetOutputSize:  100_000,
			approxComprRatio:  0.4,
			expInputThreshold: 250_000,
		},
		{
			targetOutputSize:  1,
			approxComprRatio:  0.000001,
			expInputThreshold: 1_000_000,
		},
		{
			targetOutputSize: 0,
			approxComprRatio: 0,
			assertion: func(output uint64) {
				// Need to allow for NaN depending on the machine architecture
				require.True(t, output == uint64(0) || output == uint64(math.NaN()))
			},
		},
	}

	// Validate each test case
	for i, tt := range tests {
		t.Run(fmt.Sprintf("test-%d", i), func(t *testing.T) {
			comp, err := compressor.NewRatioCompressor(compressor.Config{
				TargetOutputSize: tt.targetOutputSize,
				ApproxComprRatio: tt.approxComprRatio,
				CompressionAlgo:  derive.Zlib,
			})
			require.NoError(t, err)
			got := comp.(*compressor.RatioCompressor).InputThreshold()
			if tt.assertion != nil {
				tt.assertion(got)
			} else {
				require.Equal(t, tt.expInputThreshold, got)
			}
		})
	}
}
