package batcher_test

import (
	"bytes"
	"compress/zlib"
	"io"
	"math/rand"
	"testing"

	"github.com/ethereum-optimism/optimism/op-batcher/batcher"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/stretchr/testify/require"
)

func randomBytes(t *testing.T, length int) []byte {
	b := make([]byte, length)
	_, err := rand.Read(b)
	require.NoError(t, err)
	return b
}

func TestShadowCompressor(t *testing.T) {
	type test struct {
		name            string
		targetFrameSize uint64
		targetNumFrames int
		data            [][]byte
		errs            []error
		fullErr         error
		minRatio        float64
	}

	tests := []test{{
		name:            "no data",
		targetFrameSize: 1,
		targetNumFrames: 1,
		data:            [][]byte{},
		errs:            []error{},
		fullErr:         nil,
	}, {
		name:            "large first block",
		targetFrameSize: 1,
		targetNumFrames: 1,
		data:            [][]byte{bytes.Repeat([]byte{0}, 1024)},
		errs:            []error{nil},
		fullErr:         derive.CompressorFullErr,
		minRatio:        0.02,
	}, {
		name:            "large second block",
		targetFrameSize: 1,
		targetNumFrames: 1,
		data:            [][]byte{bytes.Repeat([]byte{0}, 512), bytes.Repeat([]byte{0}, 1024)},
		errs:            []error{nil, derive.CompressorFullErr},
		fullErr:         derive.CompressorFullErr,
		minRatio:        0.02,
	}, {
		name:            "random data",
		targetFrameSize: 1200,
		targetNumFrames: 1,
		data:            [][]byte{randomBytes(t, 512), randomBytes(t, 512), randomBytes(t, 512)},
		errs:            []error{nil, nil, derive.CompressorFullErr},
		fullErr:         derive.CompressorFullErr,
		minRatio:        0.8,
	}}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, len(test.errs), len(test.data), "invalid test case: len(data) != len(errs)")

			sc, err := batcher.NewShadowCompressor(test.targetFrameSize, test.targetNumFrames)
			require.NoError(t, err)

			inputSize := 0
			for i, d := range test.data {
				_, err = sc.Write(d)
				inputSize += len(d)
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

			buf, err := io.ReadAll(sc)
			require.NoError(t, err)

			if inputSize > 0 {
				require.LessOrEqual(t, float64(len(buf))/float64(inputSize), test.minRatio)
			}

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
