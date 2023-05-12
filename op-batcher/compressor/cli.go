package compressor

import (
	"fmt"

	opservice "github.com/ethereum-optimism/optimism/op-service"
	"github.com/urfave/cli"
)

const (
	TargetL1TxSizeBytesFlagName = "target-l1-tx-size-bytes"
	TargetNumFramesFlagName     = "target-num-frames"
	ApproxComprRatioFlagName    = "approx-compr-ratio"
	TypeFlagName                = "compressor"
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
			Name:   TypeFlagName,
			Usage:  "The type of compressor. Valid options: " + FactoryFlags(),
			EnvVar: opservice.PrefixEnvVar(envPrefix, "COMPRESSOR"),
			Value:  Ratio.FlagValue,
		},
	}
}

type CLIConfig struct {
	Type   string
	Config Config
}

func (c *CLIConfig) Check() error {
	_, err := c.Factory()
	return err
}

func (c *CLIConfig) Factory() (FactoryFunc, error) {
	for _, f := range Factories {
		if f.FlagValue == c.Type {
			return f.FactoryFunc, nil
		}
	}
	return nil, fmt.Errorf("unknown compressor kind: %q", c.Type)
}

func ReadCLIConfig(ctx *cli.Context) CLIConfig {
	return CLIConfig{
		Type: ctx.GlobalString(TypeFlagName),
		Config: Config{
			TargetFrameSize:  ctx.GlobalUint64(TargetL1TxSizeBytesFlagName) - 1, // subtract 1 byte for version,
			TargetNumFrames:  ctx.GlobalInt(TargetNumFramesFlagName),
			ApproxComprRatio: ctx.GlobalFloat64(ApproxComprRatioFlagName),
		},
	}
}
