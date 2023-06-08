package compressor

import (
	"strings"

	opservice "github.com/ethereum-optimism/optimism/op-service"
	"github.com/urfave/cli"
)

const (
	TargetL1TxSizeBytesFlagName = "target-l1-tx-size-bytes"
	TargetNumFramesFlagName     = "target-num-frames"
	ApproxComprRatioFlagName    = "approx-compr-ratio"
	KindFlagName                = "compressor"
)

func CLIFlags(envPrefix string) []cli.Flag {
	return []cli.Flag{
		cli.Uint64Flag{
			Name:   TargetL1TxSizeBytesFlagName,
			Usage:  "The target size of a batch tx submitted to L1.",
			Value:  100_000,
			EnvVar: opservice.PrefixEnvVar(envPrefix, "TARGET_L1_TX_SIZE_BYTES"),
		},
		cli.IntFlag{
			Name:   TargetNumFramesFlagName,
			Usage:  "The target number of frames to create per channel",
			Value:  1,
			EnvVar: opservice.PrefixEnvVar(envPrefix, "TARGET_NUM_FRAMES"),
		},
		cli.Float64Flag{
			Name:   ApproxComprRatioFlagName,
			Usage:  "The approximate compression ratio (<= 1.0)",
			Value:  0.4,
			EnvVar: opservice.PrefixEnvVar(envPrefix, "APPROX_COMPR_RATIO"),
		},
		cli.StringFlag{
			Name:   KindFlagName,
			Usage:  "The type of compressor. Valid options: " + strings.Join(KindKeys, ", "),
			EnvVar: opservice.PrefixEnvVar(envPrefix, "COMPRESSOR"),
			Value:  RatioKind,
		},
	}
}

type CLIConfig struct {
	// TargetL1TxSizeBytes to target when creating channel frames. Note that if the
	// realized compression ratio is worse than the approximate, more frames may
	// actually be created. This also depends on how close the target is to the
	// max frame size.
	TargetL1TxSizeBytes uint64
	// TargetNumFrames to create in this channel. If the realized compression ratio
	// is worse than approxComprRatio, additional leftover frame(s) might get created.
	TargetNumFrames int
	// ApproxComprRatio to assume. Should be slightly smaller than average from
	// experiments to avoid the chances of creating a small additional leftover frame.
	ApproxComprRatio float64
	// Type of compressor to use. Must be one of KindKeys.
	Kind string
}

func (c *CLIConfig) Config() Config {
	return Config{
		TargetFrameSize:  c.TargetL1TxSizeBytes - 1, // subtract 1 byte for version
		TargetNumFrames:  c.TargetNumFrames,
		ApproxComprRatio: c.ApproxComprRatio,
		Kind:             c.Kind,
	}
}

func ReadCLIConfig(ctx *cli.Context) CLIConfig {
	return CLIConfig{
		Kind:                ctx.GlobalString(KindFlagName),
		TargetL1TxSizeBytes: ctx.GlobalUint64(TargetL1TxSizeBytesFlagName),
		TargetNumFrames:     ctx.GlobalInt(TargetNumFramesFlagName),
		ApproxComprRatio:    ctx.GlobalFloat64(ApproxComprRatioFlagName),
	}
}
