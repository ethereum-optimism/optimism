package batcher

import (
	"bytes"
	"compress/zlib"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
)

type TargetSizeCompressor struct {
	// The target number of frames to create per channel. Note that if the
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

func NewTargetSizeCompressor(targetFrameSize uint64, targetNumFrames int, approxCompRatio float64) (derive.Compressor, error) {
	c := &TargetSizeCompressor{
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

func (t *TargetSizeCompressor) Write(p []byte) (int, error) {
	if err := t.FullErr(); err != nil {
		return 0, err
	}
	t.inputBytes += len(p)
	return t.compress.Write(p)
}

func (t *TargetSizeCompressor) Close() error {
	return t.compress.Close()
}

func (t *TargetSizeCompressor) Read(p []byte) (int, error) {
	return t.buf.Read(p)
}

func (t *TargetSizeCompressor) Reset() {
	t.buf.Reset()
	t.compress.Reset(&t.buf)
	t.inputBytes = 0
}

func (t *TargetSizeCompressor) Len() int {
	return t.buf.Len()
}

func (t *TargetSizeCompressor) Flush() error {
	return t.compress.Flush()
}

func (t *TargetSizeCompressor) FullErr() error {
	if t.inputTargetReached() {
		return derive.CompressorFullErr
	}
	return nil
}

// InputThreshold calculates the input data threshold in bytes from the given
// parameters.
func (t *TargetSizeCompressor) InputThreshold() uint64 {
	return uint64(float64(t.TargetNumFrames) * float64(t.TargetFrameSize) / t.ApproxComprRatio)
}

// inputTargetReached says whether the target amount of input data has been
// reached in this channel builder. No more blocks can be added afterwards.
func (t *TargetSizeCompressor) inputTargetReached() bool {
	return uint64(t.inputBytes) >= t.InputThreshold()
}
