package flags

import (
	"github.com/urfave/cli/v2"

	opservice "github.com/ethereum-optimism/optimism/op-service"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
)

const EnvVarPrefix = "KURTOSIS_INSPECT"

var (
	FixTraefik = &cli.BoolFlag{
		Name:    "fix-traefik",
		Value:   false,
		EnvVars: opservice.PrefixEnvVar(EnvVarPrefix, "FIX_TRAEFIK"),
		Usage:   "Fix missing Traefik labels on containers",
	}
	ConductorConfig = &cli.StringFlag{
		Name:    "conductor-config",
		Value:   "",
		EnvVars: opservice.PrefixEnvVar(EnvVarPrefix, "CONDUCTOR_CONFIG"),
		Usage:   "Path to write conductor configuration TOML file",
	}
	Environment = &cli.StringFlag{
		Name:    "environment",
		Value:   "",
		EnvVars: opservice.PrefixEnvVar(EnvVarPrefix, "ENVIRONMENT"),
		Usage:   "Path to write environment JSON file",
	}
)

var requiredFlags = []cli.Flag{
	// No required flags
}

var optionalFlags = []cli.Flag{
	FixTraefik,
	ConductorConfig,
	Environment,
}

var Flags []cli.Flag

func init() {
	// Add common op-service flags
	optionalFlags = append(optionalFlags, oplog.CLIFlags(EnvVarPrefix)...)

	Flags = append(requiredFlags, optionalFlags...)
}
