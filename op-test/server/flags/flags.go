package flags

import (
	"github.com/urfave/cli/v2"

	opservice "github.com/ethereum-optimism/optimism/op-service"
)

const EnvVarPrefix = "OP_TEST"

func prefixEnvVars(name string) []string {
	return opservice.PrefixEnvVar(EnvVarPrefix, name)
}

var (
	ConfigFlag = &cli.StringFlag{
		Name:    "config",
		Usage:   "Filepath of config defining live resources like devnets, and their constraints.",
		EnvVars: prefixEnvVars("CONFIG"),
	}
)

var Flags = []cli.Flag{
	ConfigFlag,
}
