package flags

import "github.com/urfave/cli/v2"

func init() {
	cli.HelpFlag.(*cli.BoolFlag).Category = MiscCategory
	cli.VersionFlag.(*cli.BoolFlag).Category = MiscCategory
}

var (
	ExperimentalOPStackAPI = &cli.BoolFlag{
		Name:     "experimental.sequencer-api",
		Usage:    "Enables experimental test sequencer RPC functionality",
		Required: false,
		EnvVars:  prefixEnvVars("EXPERIMENTAL_SEQUENCER_API"),
		Category: MiscCategory,
	}
)

var MiscFlags = []cli.Flag{
	ExperimentalOPStackAPI,
}
