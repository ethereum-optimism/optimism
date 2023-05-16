package compressor

import (
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
)

type Config struct {
	// TargetFrameSize to target when creating channel frames. Note that if the
	// realized compression ratio is worse than the approximate, more frames may
	// actually be created. This also depends on how close the target is to the
	// max frame size.
	TargetFrameSize uint64
	// TargetNumFrames to create in this channel. If the realized compression ratio
	// is worse than approxComprRatio, additional leftover frame(s) might get created.
	TargetNumFrames int
	// ApproxComprRatio to assume. Should be slightly smaller than average from
	// experiments to avoid the chances of creating a small additional leftover frame.
	ApproxComprRatio float64
	// Kind of compressor to use. Must
	Kind string
}

func (c Config) NewCompressor() (derive.Compressor, error) {
	if k, ok := Kinds[c.Kind]; ok {
		return k(c)
	}
	// default to RatioCompressor
	return Kinds[RatioKind](c)
}
