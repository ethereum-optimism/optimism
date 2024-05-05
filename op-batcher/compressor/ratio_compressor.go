package compressor

import (
	"bytes"
	"compress/zlib"
	"fmt"
	"io"

	"github.com/andybalholm/brotli"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
)

type CompressorWriter interface {
	Write([]byte) (int, error)
	Flush() error
	Close() error
	Reset(io.Writer)
}

type AlgoCompressor struct {
	compressed *bytes.Buffer
	writer     CompressorWriter
	algo       derive.CompressionAlgo
}

func NewAlgoCompressor(algo derive.CompressionAlgo) (*AlgoCompressor, error) {
	var writer CompressorWriter
	var err error
	compressed := &bytes.Buffer{}
	if algo == derive.Zlib {
		writer, err = zlib.NewWriterLevel(compressed, zlib.BestCompression)
	} else if algo.IsBrotli() {
		compressed.WriteByte(derive.ChannelVersionBrotli)
		writer = brotli.NewWriterLevel(compressed, derive.GetBrotliLevel(algo))
	} else {
		return nil, fmt.Errorf("unsupported compression algorithm: %s", algo)
	}

	if err != nil {
		return nil, err
	}

	return &AlgoCompressor{
		writer:     writer,
		compressed: compressed,
		algo:       algo,
	}, nil
}

func (ac *AlgoCompressor) Write(data []byte) (int, error) {
	return ac.writer.Write(data)
}

func (ac *AlgoCompressor) Flush() error {
	return ac.writer.Flush()
}

func (ac *AlgoCompressor) Close() error {
	return ac.writer.Close()
}

func (ac *AlgoCompressor) Reset() {
	ac.compressed.Reset()
	if ac.algo.IsBrotli() {
		// always add channal version for brotli
		ac.compressed.WriteByte(derive.ChannelVersionBrotli)
	}
	ac.writer.Reset(ac.compressed)
}

func (ac *AlgoCompressor) Len() int {
	return ac.compressed.Len()
}

func (ac *AlgoCompressor) Read(p []byte) (int, error) {
	return ac.compressed.Read(p)
}

type RatioCompressor struct {
	config Config

	inputBytes int
	compressor *AlgoCompressor
}

// NewRatioCompressor creates a new derive.Compressor implementation that uses the target
// size and a compression ratio parameter to determine how much data can be written to
// the compressor before it's considered full. The full calculation is as follows:
//
//	full = uncompressedLength * approxCompRatio >= targetFrameSize * targetNumFrames
func NewRatioCompressor(config Config) (derive.Compressor, error) {
	c := &RatioCompressor{
		config: config,
	}

	compressor, err := NewAlgoCompressor(config.CompressionAlgo)
	if err != nil {
		return nil, err
	}
	c.compressor = compressor

	return c, nil
}

func (t *RatioCompressor) Write(p []byte) (int, error) {
	if err := t.FullErr(); err != nil {
		return 0, err
	}
	t.inputBytes += len(p)
	return t.compressor.Write(p)
}

func (t *RatioCompressor) Close() error {
	return t.compressor.Close()
}

func (t *RatioCompressor) Read(p []byte) (int, error) {
	return t.compressor.Read(p)
}

func (t *RatioCompressor) Reset() {
	t.compressor.Reset()
	t.inputBytes = 0
}

func (t *RatioCompressor) Len() int {
	return t.compressor.Len()
}

func (t *RatioCompressor) Flush() error {
	return t.compressor.Flush()
}

func (t *RatioCompressor) FullErr() error {
	if t.inputTargetReached() {
		return derive.ErrCompressorFull
	}
	return nil
}

// InputThreshold calculates the input data threshold in bytes from the given
// parameters.
func (t *RatioCompressor) InputThreshold() uint64 {
	return uint64(float64(t.config.TargetOutputSize) / t.config.ApproxComprRatio)
}

// inputTargetReached says whether the target amount of input data has been
// reached in this channel builder. No more blocks can be added afterwards.
func (t *RatioCompressor) inputTargetReached() bool {
	return uint64(t.inputBytes) >= t.InputThreshold()
}
