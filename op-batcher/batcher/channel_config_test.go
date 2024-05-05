package batcher

import (
	"fmt"
	"math"
	"testing"

	"github.com/ethereum-optimism/optimism/op-batcher/compressor"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"

	"github.com/stretchr/testify/require"
)

func defaultTestChannelConfig(algo derive.CompressionAlgo) ChannelConfig {
	c := ChannelConfig{
		SeqWindowSize:      15,
		ChannelTimeout:     40,
		MaxChannelDuration: 1,
		SubSafetyMargin:    4,
		MaxFrameSize:       120_000,
		TargetNumFrames:    1,
		BatchType:          derive.SingularBatchType,
		CompressorConfig: compressor.Config{
			CompressionAlgo: algo,
		},
	}
	c.InitRatioCompressor(0.4, algo)
	return c
}

func TestChannelConfig_Check(t *testing.T) {
	type test struct {
		input     func(algo derive.CompressionAlgo) ChannelConfig
		assertion func(error)
	}

	tests := []test{
		{
			input: func(algo derive.CompressionAlgo) ChannelConfig { return defaultTestChannelConfig(algo) },
			assertion: func(output error) {
				require.NoError(t, output)
			},
		},
		{
			input: func(algo derive.CompressionAlgo) ChannelConfig {
				cfg := defaultTestChannelConfig(algo)
				cfg.ChannelTimeout = 0
				cfg.SubSafetyMargin = 1
				return cfg
			},
			assertion: func(output error) {
				require.ErrorIs(t, output, ErrInvalidChannelTimeout)
			},
		},
	}
	for i := 0; i < derive.FrameV0OverHeadSize; i++ {
		expectedErr := fmt.Sprintf("max frame size %d is less than the minimum 23", i)
		i := i // need to udpate Go version...
		tests = append(tests, test{
			input: func(algo derive.CompressionAlgo) ChannelConfig {
				cfg := defaultTestChannelConfig(algo)
				cfg.MaxFrameSize = uint64(i)
				return cfg
			},
			assertion: func(output error) {
				require.EqualError(t, output, expectedErr)
			},
		})
	}

	// Run the table tests
	for _, test := range tests {
		for _, algo := range derive.CompressionAlgoTypes {
			cfg := test.input(algo)
			test.assertion(cfg.Check())
		}
	}
}

// FuzzChannelConfig_CheckTimeout tests the [ChannelConfig.Check] function
// with fuzzing to make sure that a [ErrInvalidChannelTimeout] is thrown when
// the ChannelTimeout is less than the SubSafetyMargin.
func FuzzChannelConfig_CheckTimeout(f *testing.F) {
	for i := range [10]int{} {
		for _, algo := range derive.CompressionAlgoTypes {
			f.Add(uint64(i+1), uint64(i), algo.String())
		}

	}
	f.Fuzz(func(t *testing.T, channelTimeout uint64, subSafetyMargin uint64, algo string) {
		// We only test where [ChannelTimeout] is less than the [SubSafetyMargin]
		// So we cannot have [ChannelTimeout] be [math.MaxUint64]
		if channelTimeout == math.MaxUint64 {
			channelTimeout = math.MaxUint64 - 1
		}
		if subSafetyMargin <= channelTimeout {
			subSafetyMargin = channelTimeout + 1
		}

		channelConfig := defaultTestChannelConfig(derive.CompressionAlgo(algo))
		channelConfig.ChannelTimeout = channelTimeout
		channelConfig.SubSafetyMargin = subSafetyMargin
		require.ErrorIs(t, channelConfig.Check(), ErrInvalidChannelTimeout)
	})
}
