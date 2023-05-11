package batcher

import (
	"bytes"
	"compress/zlib"

	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
)

type RatioCompressor struct {
	// The frame size to target when creating channel frames. Note that if the
	// realized compression ratio is worse than the approximate, more frames may
	// actually be created. This also depends on how close TargetFrameSize is to
	// MaxFrameSize.
	TargetFrameSize uint64
	// The target number of frames to create in this channel. If the realized
	// compression ratio is worse than approxComprRatio, additional leftover
	// frame(s) might get created.
	TargetNumFrames int
	// Approximated compression ratio to assume. Should be slightly smaller than
	// average from experiments to avoid the chances of creating a small
	// additional leftover frame.
	ApproxComprRatio float64

	inputBytes int
	buf        bytes.Buffer
	compress   *zlib.Writer
}

// NewRatioCompressor creates a new derive.Compressor implementation that uses the target
// size and a compression ratio parameter to determine how much data can be written to
// the compressor before it's considered full. The full calculation is as follows:
//
//	full = uncompressedLength * approxCompRatio >= targetFrameSize * targetNumFrames
func NewRatioCompressor(targetFrameSize uint64, targetNumFrames int, approxCompRatio float64) (derive.Compressor, error) {
	c := &RatioCompressor{
		TargetFrameSize:  targetFrameSize,
		TargetNumFrames:  targetNumFrames,
		ApproxComprRatio: approxCompRatio,
	}

	compress, err := zlib.NewWriterLevel(&c.buf, zlib.BestCompression)
	if err != nil {
		return nil, err
	}
	c.compress = compress

	return c, nil
}

func (t *RatioCompressor) Write(p []byte) (int, error) {
	if err := t.FullErr(); err != nil {
		return 0, err
	}
	t.inputBytes += len(p)
	return t.compress.Write(p)
}

func (t *RatioCompressor) Close() error {
	return t.compress.Close()
}

func (t *RatioCompressor) Read(p []byte) (int, error) {
	return t.buf.Read(p)
}

func (t *RatioCompressor) Reset() {
	t.buf.Reset()
	t.compress.Reset(&t.buf)
	t.inputBytes = 0
}

func (t *RatioCompressor) Len() int {
	return t.buf.Len()
}

func (t *RatioCompressor) Flush() error {
	return t.compress.Flush()
}

func (t *RatioCompressor) FullErr() error {
	if t.inputTargetReached() {
		return derive.CompressorFullErr
	}
	return nil
}

// InputThreshold calculates the input data threshold in bytes from the given
// parameters.
func (t *RatioCompressor) InputThreshold() uint64 {
	return uint64(float64(t.TargetNumFrames) * float64(t.TargetFrameSize) / t.ApproxComprRatio)
}

// inputTargetReached says whether the target amount of input data has been
// reached in this channel builder. No more blocks can be added afterwards.
func (t *RatioCompressor) inputTargetReached() bool {
	return uint64(t.inputBytes) >= t.InputThreshold()
}
