package compressor

import (
	"bytes"
	"compress/zlib"
	"io"
	"math/rand"
	"testing"

	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
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

func TestShadowCompressor(t *testing.T) {
	tests := []struct {
		name             string
		targetOutputSize uint64
		data             [][]byte
		errs             []error
		fullErr          error
	}{{
		name:             "no data",
		targetOutputSize: 1 + derive.FrameV0OverHeadSize,
		data:             [][]byte{},
		errs:             []error{},
		fullErr:          nil,
	}, {
		name:             "large first block",
		targetOutputSize: 1 + derive.FrameV0OverHeadSize,
		data:             [][]byte{bytes.Repeat([]byte{0}, 1024)},
		errs:             []error{nil},
		fullErr:          derive.ErrCompressorFull,
	}, {
		name:             "large second block",
		targetOutputSize: 1 + derive.FrameV0OverHeadSize,
		data:             [][]byte{bytes.Repeat([]byte{0}, 512), bytes.Repeat([]byte{0}, 1024)},
		errs:             []error{nil, derive.ErrCompressorFull},
		fullErr:          derive.ErrCompressorFull,
	}, {
		name:             "random data",
		targetOutputSize: 1 << 17,
		data:             [][]byte{randomBytes((1 << 17) - 1000), randomBytes(512), randomBytes(512)},
		errs:             []error{nil, nil, derive.ErrCompressorFull},
		fullErr:          derive.ErrCompressorFull,
	}}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, len(test.errs), len(test.data), "invalid test case: len(data) != len(errs)")

			sc, err := NewShadowCompressor(Config{
				TargetOutputSize: test.targetOutputSize,
				CompressionAlgo:  derive.Zlib,
			})
			require.NoError(t, err)

			for i, d := range test.data {
				_, err = sc.Write(d)
				if test.errs[i] != nil {
					require.ErrorIs(t, err, test.errs[i])
					require.Equal(t, i, len(test.data)-1)
				} else {
					require.NoError(t, err)
				}
			}

			if test.fullErr != nil {
				require.ErrorIs(t, sc.FullErr(), test.fullErr)
			} else {
				require.NoError(t, sc.FullErr())
			}

			err = sc.Close()
			require.NoError(t, err)
			require.LessOrEqual(t, uint64(sc.Len()), sc.(*ShadowCompressor).bound)

			buf, err := io.ReadAll(sc)
			require.NoError(t, err)

			r, err := zlib.NewReader(bytes.NewBuffer(buf))
			require.NoError(t, err)

			uncompressed, err := io.ReadAll(r)
			require.NoError(t, err)

			concat := make([]byte, 0)
			for i, d := range test.data {
				if test.errs[i] != nil {
					break
				}
				concat = append(concat, d...)
			}

			require.Equal(t, concat, uncompressed)
		})
	}
}

// TestBoundInaccurateForLargeRandomData documents where our bounding heuristic starts to fail
// (writing at least 128k of random data)
func TestBoundInaccurateForLargeRandomData(t *testing.T) {
	const sizeLimit = 1 << 17

	sc, err := NewShadowCompressor(Config{
		TargetOutputSize: sizeLimit + 100,
		CompressionAlgo:  derive.Zlib,
	})
	require.NoError(t, err)

	_, err = sc.Write(randomBytes(sizeLimit + 1))
	require.NoError(t, err)
	err = sc.Close()
	require.NoError(t, err)
	require.Greater(t, uint64(sc.Len()), sc.(*ShadowCompressor).bound)

	sc.Reset()
	_, err = sc.Write(randomBytes(sizeLimit))
	require.NoError(t, err)
	err = sc.Close()
	require.NoError(t, err)
	require.LessOrEqual(t, uint64(sc.Len()), sc.(*ShadowCompressor).bound)
}
