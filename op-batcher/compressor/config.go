package compressor

import (
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
)

type Config struct {
	// TargetOutputSize is the target size that the compressed data should reach.
	// The shadow compressor guarantees that the compressed data stays below
	// this bound. The ratio compressor might go over.
	TargetOutputSize uint64
	// ApproxComprRatio to assume (only ratio compressor). Should be slightly smaller
	// than average from experiments to avoid the chances of creating a small
	// additional leftover frame.
	ApproxComprRatio float64
	// Kind of compressor to use. Must be one of KindKeys. If unset, NewCompressor
	// will default to RatioKind.
	Kind string

	// Type of compression algorithm to use. Must be one of [zlib, brotli-(9|10|11)]
	CompressionAlgo derive.CompressionAlgo
}

func (c Config) NewCompressor() (derive.Compressor, error) {
	if k, ok := Kinds[c.Kind]; ok {
		return k(c)
	}
	// default to RatioCompressor
	return Kinds[RatioKind](c)
}
