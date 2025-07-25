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
		Name:    "conductor-config-path",
		Value:   "",
		EnvVars: opservice.PrefixEnvVar(EnvVarPrefix, "CONDUCTOR_CONFIG"),
		Usage:   "Path where conductor configuration TOML file will be written (overwrites existing file)",
	}
	Environment = &cli.StringFlag{
		Name:    "environment-path",
		Value:   "",
		EnvVars: opservice.PrefixEnvVar(EnvVarPrefix, "ENVIRONMENT"),
		Usage:   "Path where environment JSON file will be written (overwrites existing file)",
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
