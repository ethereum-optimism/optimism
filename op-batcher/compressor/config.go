package compressor

type Config struct {
	// FrameSizeTarget to target when creating channel frames. Note that if the
	// realized compression ratio is worse than the approximate, more frames may
	// actually be created. This also depends on how close the target is to the
	// max frame size.
	TargetFrameSize uint64
	// NumFramesTarget to create in this channel. If the realized compression ratio
	// is worse than approxComprRatio, additional leftover frame(s) might get created.
	TargetNumFrames int
	// ApproxCompRatio to assume. Should be slightly smaller than average from
	// experiments to avoid the chances of creating a small additional leftover frame.
	ApproxComprRatio float64
}
