package flags

import (
	"github.com/urfave/cli/v2"

	opservice "github.com/ethereum-optimism/optimism/op-service"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
)

const envVarPrefix = "OP_DEVNET"

func prefixEnvVars(name string) []string {
	return opservice.PrefixEnvVar(envVarPrefix, name)
}

var (
	MonorepoDir = &cli.StringFlag{
		Name:    "monorepo.dir",
		Usage:   "Directory of the monorepo",
		EnvVars: prefixEnvVars("MONOREPO_DIR"),
	}
	Deploy = &cli.StringFlag{
		Name:    "deploy",
		Usage:   "Whether the contracts should be predeployed or deployed",
		EnvVars: prefixEnvVars("DEPLOY"),
	}
)

var Flags = []cli.Flag{
	MonorepoDir,
	Deploy,
}

func init() {
	Flags = append(Flags, oplog.CLIFlags(envVarPrefix)...)
}
