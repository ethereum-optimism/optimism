package main

import (
	"github.com/urfave/cli/v2"

	opservice "github.com/ethereum-optimism/optimism/op-service"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
)

const EnvVarPrefix = "ORACLE"

var (
	ClientCommandFlag = &cli.StringFlag{
		Name:    "client",
		Usage:   "Command to create the client sub-process with. Parent process if empty.",
		EnvVars: opservice.PrefixEnvVar(EnvVarPrefix, "CLIENT"),
	}
	HostCommandFlag = &cli.StringFlag{
		Name:    "host",
		Usage:   "Command to create the host sub-process with. Parent process if empty.",
		EnvVars: opservice.PrefixEnvVar(EnvVarPrefix, "HOST"),
	}

	InfoHintsFlag = &cli.BoolFlag{
		Name:    "info.hints",
		EnvVars: opservice.PrefixEnvVar(EnvVarPrefix, "INFO_HINTS"),
	}
	InfoPreimageKeysFlag = &cli.BoolFlag{
		Name:    "info.preimage-keys",
		EnvVars: opservice.PrefixEnvVar(EnvVarPrefix, "INFO_PREIMAGE_KEYS"),
	}
	InfoPreimageValuesFlag = &cli.BoolFlag{
		Name:    "info.preimage-values",
		EnvVars: opservice.PrefixEnvVar(EnvVarPrefix, "INFO_PREIMAGE_VALUES"),
	}
)

var Flags []cli.Flag

func init() {
	Flags = append(Flags, oplog.CLIFlags(EnvVarPrefix)...)
	Flags = append(Flags, ClientCommandFlag, HostCommandFlag)
	Flags = append(Flags, InfoHintsFlag, InfoPreimageKeysFlag, InfoPreimageValuesFlag)
}
