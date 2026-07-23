package upgrade

import (
	"github.com/urfave/cli/v2"
)

var (
	ConfigFlag = &cli.StringFlag{
		Name:  "config",
		Usage: "path to the config file",
	}
	OverrideArtifactsURLFlag = &cli.StringFlag{
		Name:  "override-artifacts-url",
		Usage: "override the artifacts URL",
	}
	OutfileFlag = &cli.StringFlag{
		Name:  "outfile",
		Usage: "path to write the output to, or - for stdout",
		Value: "-",
	}
)
