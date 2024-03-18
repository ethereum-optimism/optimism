package compressor

import (
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
)

type Config struct {
	// TargetFrameSize to target when creating channel frames.
	// It is guaranteed that a frame will never be larger.
	TargetFrameSize uint64
	// TargetNumFrames to create in this channel. If the first block that is added
	// doesn't fit within a single frame, more frames might be created.
	TargetNumFrames int
	// ApproxComprRatio to assume. Should be slightly smaller than average from
	// experiments to avoid the chances of creating a small additional leftover frame.
	ApproxComprRatio float64
	// Kind of compressor to use. Must be one of KindKeys. If unset, NewCompressor
	// will default to RatioKind.
	Kind string
}

func (c Config) NewCompressor() (derive.Compressor, error) {
	if k, ok := Kinds[c.Kind]; ok {
		return k(c)
	}
	// default to RatioCompressor
	return Kinds[RatioKind](c)
}
